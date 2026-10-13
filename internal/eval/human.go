package eval

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
	"unicode"
)

const humanProtocol = "writing-human-v1"

type HumanPair struct {
	ID         string `json:"id"`
	Case       Case   `json:"case"`
	Run        string `json:"run"`
	Repetition int    `json:"repetition"`
	AVariant   string `json:"a_variant"`
	A          string `json:"a"`
	B          string `json:"b"`
}
type HumanSuite struct {
	Protocol       string              `json:"protocol"`
	Reviewer       string              `json:"reviewer"`
	Created        time.Time           `json:"created"`
	Seed           string              `json:"seed"`
	AllRepetitions bool                `json:"all_repetitions"`
	Writer         Model               `json:"writer"`
	Sources        map[string]Snapshot `json:"sources"`
	Pairs          []HumanPair         `json:"pairs"`
}
type HumanVote struct {
	PairID  string   `json:"pair_id"`
	Choice  string   `json:"choice"`
	IssuesA []string `json:"issues_a,omitempty"`
	IssuesB []string `json:"issues_b,omitempty"`
	Note    string   `json:"note,omitempty"`
}
type HumanEvent struct {
	At     time.Time `json:"at"`
	Action string    `json:"action"`
	Vote   HumanVote `json:"vote"`
}
type HumanJournal struct {
	SuiteHash string       `json:"suite_hash"`
	Events    []HumanEvent `json:"events"`
}
type HumanOptions struct {
	Session        string
	Runs           []string
	Reviewer       string
	AllRepetitions bool
	Init           bool
	Undo           bool
	Limit          int
}

// File locks are released by the OS on Ctrl-C or process failure (macOS/Linux).
func humanLock(dir string) (*os.File, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(dir, ".human.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, fmt.Errorf("another human-review command is using this session")
	}
	return f, nil
}
func loadHuman(dir string) (HumanSuite, HumanJournal, error) {
	var s HumanSuite
	var j HumanJournal
	b, err := os.ReadFile(filepath.Join(dir, "suite.json"))
	if err != nil {
		return s, j, err
	}
	if err = json.Unmarshal(b, &s); err != nil {
		return s, j, err
	}
	if s.Protocol != humanProtocol {
		return s, j, fmt.Errorf("unsupported human-review protocol")
	}
	b, err = os.ReadFile(filepath.Join(dir, "reviews.json"))
	if os.IsNotExist(err) {
		j.SuiteHash = digest(s)
		return s, j, nil
	}
	if err != nil {
		return s, j, err
	}
	if err = json.Unmarshal(b, &j); err != nil {
		return s, j, err
	}
	if j.SuiteHash != digest(s) {
		return s, j, fmt.Errorf("review inputs changed; restore suite.json or use a new session")
	}
	ids := map[string]bool{}
	for _, p := range s.Pairs {
		ids[p.ID] = true
	}
	for _, e := range j.Events {
		if !ids[e.Vote.PairID] || (e.Action != "vote" && e.Action != "undo") {
			return s, j, fmt.Errorf("invalid review event")
		}
		if e.Action == "vote" {
			if err = validateVote(e.Vote); err != nil {
				return s, j, err
			}
		}
	}
	return s, j, nil
}
func newHuman(o HumanOptions) (HumanSuite, error) {
	if o.Reviewer == "" {
		o.Reviewer = "owner"
	}
	s := HumanSuite{Protocol: humanProtocol, Reviewer: o.Reviewer, Created: time.Now().UTC(), AllRepetitions: o.AllRepetitions, Sources: map[string]Snapshot{}}
	seed := make([]byte, 16)
	if _, err := rand.Read(seed); err != nil {
		return s, err
	}
	s.Seed = hex.EncodeToString(seed)
	if len(o.Runs) == 0 {
		return s, fmt.Errorf("new session requires at least one --run")
	}
	if strings.TrimSpace(o.Reviewer) == "" {
		return s, fmt.Errorf("reviewer must not be empty")
	}
	seen := map[string]bool{}
	for n, dir := range o.Runs {
		m, err := LoadManifest(dir)
		if err != nil {
			return s, err
		}
		if m.Config.Repetitions < 1 || len(m.Cases) == 0 {
			return s, fmt.Errorf("invalid generation manifest")
		}
		if n == 0 {
			s.Writer = m.Config.Writer
			for _, key := range []string{"baseline", "candidate", "samples"} {
				s.Sources[key] = m.Sources[key]
			}
		} else {
			if digest(s.Writer) != digest(m.Config.Writer) {
				return s, fmt.Errorf("writer settings differ; use separate sessions")
			}
			for _, key := range []string{"baseline", "candidate", "samples"} {
				if s.Sources[key].Content != m.Sources[key].Content {
					return s, fmt.Errorf("%s differs across runs; use separate sessions", key)
				}
			}
		}
		rows, err := readRecords(filepath.Join(dir, "generations.jsonl"))
		if err != nil {
			return s, err
		}
		// Verify saved responses belong to the frozen requests before selecting any.
		for _, job := range generationJobs(m) {
			r, ok := rows[job.Record.ID]
			if !ok || r.Error != "" || r.Response.Status != "completed" {
				return s, fmt.Errorf("%s: writing results incomplete; finish generation first", dir)
			}
			if r.Hash != job.Record.Hash {
				return s, fmt.Errorf("%s: generation does not match frozen inputs", dir)
			}
		}
		for _, c := range m.Cases {
			if seen[c.ID] {
				return s, fmt.Errorf("duplicate case %s across runs; omit overlapping runs (such as the pilot)", c.ID)
			}
			seen[c.ID] = true
			var reps []int
			for i := 0; i < m.Config.Repetitions; i++ {
				reps = append(reps, i)
			}
			sort.Slice(reps, func(i, j int) bool {
				return digest([]any{s.Seed, c.ID, reps[i]}) < digest([]any{s.Seed, c.ID, reps[j]})
			})
			if !o.AllRepetitions {
				reps = reps[:1]
			}
			for _, rep := range reps {
				id := fmt.Sprintf("%s/%d", c.ID, rep)
				a, b := rows[id+"/baseline"].Response.Text, rows[id+"/candidate"].Response.Text
				variant := "baseline"
				if digest([]string{s.Seed, id, "side"})[0] < '8' {
					a, b = b, a
					variant = "candidate"
				}
				abs, err := filepath.Abs(dir)
				if err != nil {
					return s, err
				}
				s.Pairs = append(s.Pairs, HumanPair{ID: id, Case: c, Run: abs, Repetition: rep, AVariant: variant, A: a, B: b})
			}
		}
	}
	sort.Slice(s.Pairs, func(i, j int) bool {
		return digest([]string{s.Seed, s.Pairs[i].ID, "queue"}) < digest([]string{s.Seed, s.Pairs[j].ID, "queue"})
	})
	if len(s.Pairs) == 0 {
		return s, fmt.Errorf("no comparison pairs")
	}
	return s, nil
}
func humanVotes(j HumanJournal) map[string]HumanVote {
	v := map[string]HumanVote{}
	for _, e := range j.Events {
		if e.Action == "undo" {
			delete(v, e.Vote.PairID)
		} else {
			v[e.Vote.PairID] = e.Vote
		}
	}
	return v
}

var humanIssues = map[string]string{
	"meaning":     "Changed facts, scope, or uncertainty",
	"unsupported": "Added unsupported claims or context",
	"completion":  "Did not complete the requested task",
	"verbose":     "Repeated or unnecessary explanation",
	"agency":      "Assigned an unsuitable action or property to a subject",
	"vague":       "Vague or inflated wording",
	"voice":       "Tone does not fit my voice",
	"structure":   "Weak organization or connections",
	"overedit":    "Changed effective wording unnecessarily",
	"other":       "Another issue (describe in the note)",
}

func validateVote(v HumanVote) error {
	if v.Choice != "a" && v.Choice != "b" && v.Choice != "tie" && v.Choice != "neither" && v.Choice != "skip" {
		return fmt.Errorf("invalid preference")
	}
	for _, issues := range [][]string{v.IssuesA, v.IssuesB} {
		for _, tag := range issues {
			if _, ok := humanIssues[tag]; !ok {
				return fmt.Errorf("unknown issue %q", tag)
			}
		}
	}
	return nil
}
func parseIssues(line string) ([]string, []string, error) {
	var a, b []string
	for _, item := range strings.Fields(strings.ReplaceAll(strings.ToLower(line), ",", " ")) {
		parts := strings.Split(item, ":")
		if len(parts) != 2 {
			return nil, nil, fmt.Errorf("use a:verbose b:meaning, or Enter for none")
		}
		if _, ok := humanIssues[parts[1]]; !ok {
			return nil, nil, fmt.Errorf("unknown issue %q", parts[1])
		}
		if parts[0] == "both" {
			a = append(a, parts[1])
			b = append(b, parts[1])
		} else if parts[0] == "a" {
			a = append(a, parts[1])
		} else if parts[0] == "b" {
			b = append(b, parts[1])
		} else {
			return nil, nil, fmt.Errorf("prefix each issue with a:, b:, or both:")
		}
	}
	return uniqueStrings(a), uniqueStrings(b), nil
}
func uniqueStrings(v []string) []string {
	sort.Strings(v)
	returnList := []string{}
	for _, x := range v {
		if len(returnList) == 0 || returnList[len(returnList)-1] != x {
			returnList = append(returnList, x)
		}
	}
	return returnList
}

// Strip terminal controls from model text and notes; never execute escape sequences.
func humanText(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' || (!unicode.IsControl(r) && !unicode.Is(unicode.Cf, r)) {
			return r
		}
		return -1
	}, s)
}
func printHumanText(out io.Writer, s string) {
	// Let the terminal wrap at its actual width; preserve indentation and spacing.
	fmt.Fprintln(out, humanText(s))
}
func lastHumanVote(j HumanJournal) string {
	votes := humanVotes(j)
	for i := len(j.Events) - 1; i >= 0; i-- {
		e := j.Events[i]
		if e.Action == "vote" {
			if _, active := votes[e.Vote.PairID]; active {
				return e.Vote.PairID
			}
		}
	}
	return ""
}
func appendHuman(dir string, s HumanSuite, j *HumanJournal, e HumanEvent) error {
	next := HumanJournal{SuiteHash: digest(s), Events: append(append([]HumanEvent(nil), j.Events...), e)}
	if err := WriteJSON(filepath.Join(dir, "reviews.json"), next); err != nil {
		return err
	}
	*j = next
	return nil
}
func humanProgress(s HumanSuite, j HumanJournal) (int, int) {
	v := humanVotes(j)
	done, skipped := 0, 0
	for _, p := range s.Pairs {
		if r, ok := v[p.ID]; ok {
			if r.Choice == "skip" {
				skipped++
			} else {
				done++
			}
		}
	}
	return done, skipped
}

// HumanReview does not load model judgments or make model calls.
func HumanReview(o HumanOptions, in io.Reader, out io.Writer) error {
	if o.Init && o.Undo {
		return fmt.Errorf("--init and --undo cannot be combined")
	}
	if o.Limit < 0 {
		return fmt.Errorf("limit must be nonnegative (0 means no limit)")
	}
	lock, err := humanLock(o.Session)
	if err != nil {
		return err
	}
	defer lock.Close()
	_, err = os.Stat(filepath.Join(o.Session, "suite.json"))
	if os.IsNotExist(err) {
		s, e := newHuman(o)
		if e != nil {
			return e
		}
		if e = WriteJSON(filepath.Join(o.Session, "suite.json"), s); e != nil {
			return e
		}
	} else if err != nil {
		return err
	} else if len(o.Runs) > 0 || o.AllRepetitions {
		return fmt.Errorf("session already exists; omit --run and --all-repetitions to resume")
	}
	s, j, err := loadHuman(o.Session)
	if err != nil {
		return err
	}
	if o.Reviewer != "" && o.Reviewer != s.Reviewer {
		return fmt.Errorf("session belongs to reviewer %q; create a separate session for another reviewer", s.Reviewer)
	}
	if o.Init {
		fmt.Fprintf(out, "Prepared %d anonymous comparisons. Start with: eval review --session %s\n", len(s.Pairs), o.Session)
		return nil
	}
	fmt.Fprintln(out, "Choose the writing you would keep: accuracy first, then clarity and your voice.")
	fmt.Fprintln(out, "A/B choose a response; tie = equally acceptable; neither = both unacceptable.")
	fmt.Fprintln(out, "One judgment is saved after its optional issue tags and note. Ctrl-C keeps earlier judgments.")
	reader := bufio.NewScanner(in)
	reader.Buffer(make([]byte, 4096), 1024*1024)
	read := func(prompt string) (string, bool) {
		fmt.Fprint(out, prompt)
		if !reader.Scan() {
			return "", false
		}
		return strings.TrimSpace(reader.Text()), true
	}
	seen := map[string]bool{}
	completed := 0
	focus := ""
	if o.Undo {
		focus = lastHumanVote(j)
		if focus == "" {
			return fmt.Errorf("no saved judgment to undo")
		}
		if err = appendHuman(o.Session, s, &j, HumanEvent{At: time.Now().UTC(), Action: "undo", Vote: HumanVote{PairID: focus}}); err != nil {
			return err
		}
		fmt.Fprintln(out, "Undid the last saved judgment.")
	}
	finish := func() error {
		if reader.Err() != nil {
			return reader.Err()
		}
		done, skipped := humanProgress(s, j)
		fmt.Fprintf(out, "\nSaved: %d/%d reviewed; %d skipped. Resume with: eval review --session %s\n", done, len(s.Pairs), skipped, o.Session)
		fmt.Fprintln(out, "Interpret your results with: eval human-report --session "+o.Session)
		return writeHumanReport(o.Session, s, j, io.Discard)
	}
	for o.Limit == 0 || completed < o.Limit {
		votes := humanVotes(j)
		var current *HumanPair
		for i := range s.Pairs {
			p := &s.Pairs[i]
			if p.ID == focus {
				current = p
				break
			}
			if focus != "" {
				continue
			}
			v, ok := votes[p.ID]
			if !seen[p.ID] && (!ok || v.Choice == "skip") {
				current = p
				break
			}
		}
		focus = ""
		if current == nil {
			break
		}
		p := *current
		done, _ := humanProgress(s, j)
		fmt.Fprintf(out, "\n%s\nComparison • %s • %d/%d reviewed\n\nTASK\n", strings.Repeat("=", 72), p.Case.Task, done, len(s.Pairs))
		printHumanText(out, p.Case.Input)
		if p.A == p.B {
			fmt.Fprintln(out, "\nRESPONSES A AND B (identical)")
			printHumanText(out, p.A)
		} else {
			fmt.Fprintln(out, "\nRESPONSE A")
			printHumanText(out, p.A)
			fmt.Fprintln(out, "\nRESPONSE B")
			printHumanText(out, p.B)
		}
	choiceLoop:
		for {
			answer, ok := read("\n[a/b/tie/neither]  c criteria · v voice samples · s skip · u undo · q quit > ")
			if !ok {
				return finish()
			}
			answer = strings.ToLower(answer)
			switch answer {
			case "q", "quit":
				return finish()
			case "c":
				for i, c := range p.Case.Criteria {
					fmt.Fprintf(out, "%d. %s\n", i+1, humanText(c))
				}
				continue
			case "v":
				printHumanText(out, s.Sources["samples"].Content)
				continue
			case "u", "undo":
				undone := lastHumanVote(j)
				if undone == "" {
					fmt.Fprintln(out, "No saved judgment to undo.")
					continue
				}
				if err = appendHuman(o.Session, s, &j, HumanEvent{At: time.Now().UTC(), Action: "undo", Vote: HumanVote{PairID: undone}}); err != nil {
					return err
				}
				focus = undone
				delete(seen, undone)
				fmt.Fprintln(out, "Undid the last saved judgment.")
				break choiceLoop
			case "s", "skip":
				answer = "skip"
			case "t":
				answer = "tie"
			case "n":
				answer = "neither"
			}
			vote := HumanVote{PairID: p.ID, Choice: answer}
			if validateVote(vote) != nil {
				fmt.Fprintln(out, "Choose a, b, tie, neither, or a listed command.")
				continue
			}
			if answer != "skip" {
				fmt.Fprintln(out, "Optional issues: meaning unsupported completion verbose agency vague voice structure overedit other")
				for {
					line, ok := read("Issues (e.g. a:verbose b:meaning; Enter = none; ? = help): ")
					if !ok {
						return finish()
					}
					if line == "?" {
						keys := []string{}
						for k := range humanIssues {
							keys = append(keys, k)
						}
						sort.Strings(keys)
						for _, k := range keys {
							fmt.Fprintf(out, "  %s: %s\n", k, humanIssues[k])
						}
						continue
					}
					vote.IssuesA, vote.IssuesB, err = parseIssues(line)
					if err == nil {
						break
					}
					fmt.Fprintln(out, err)
				}
				note, ok := read("Why, or a suggested edit (optional): ")
				if !ok {
					return finish()
				}
				vote.Note = note
			}
			if err = appendHuman(o.Session, s, &j, HumanEvent{At: time.Now().UTC(), Action: "vote", Vote: vote}); err != nil {
				return err
			}
			seen[p.ID] = true
			if answer != "skip" {
				completed++
			}
			fmt.Fprintln(out, "Saved.")
			break
		}
	}
	return finish()
}
