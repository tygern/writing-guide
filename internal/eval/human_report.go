package eval

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type HumanCaseResult struct {
	ID        string              `json:"id"`
	Split     string              `json:"split"`
	Task      string              `json:"task"`
	Expected  int                 `json:"expected"`
	Reviewed  int                 `json:"reviewed"`
	Baseline  int                 `json:"baseline"`
	Candidate int                 `json:"candidate"`
	Ties      int                 `json:"ties"`
	Neither   int                 `json:"neither"`
	Winner    string              `json:"winner"`
	Issues    map[string][]string `json:"issues"`
}
type HumanCounts struct {
	Cases        int                       `json:"cases"`
	Complete     int                       `json:"complete"`
	Baseline     int                       `json:"baseline"`
	Candidate    int                       `json:"candidate"`
	Ties         int                       `json:"ties"`
	Mixed        int                       `json:"mixed"`
	Neither      int                       `json:"neither"`
	HardFailures map[string]int            `json:"hard_failures"`
	ByTask       map[string]map[string]int `json:"by_task"`
}
type HumanFinding struct {
	CaseID  string `json:"case_id"`
	PairID  string `json:"pair_id"`
	Split   string `json:"split"`
	Variant string `json:"variant"`
	Issue   string `json:"issue"`
}
type HumanAnalysis struct {
	Policy      string            `json:"policy"`
	Reviewer    string            `json:"reviewer"`
	Reviewed    int               `json:"reviewed_pairs"`
	Expected    int               `json:"expected_pairs"`
	Skipped     int               `json:"skipped_pairs"`
	Direction   string            `json:"direction"`
	Reason      string            `json:"reason"`
	Development HumanCounts       `json:"development"`
	HeldOut     HumanCounts       `json:"held_out"`
	Cases       []HumanCaseResult `json:"cases"`
	Findings    []HumanFinding    `json:"findings"`
}

func hardIssue(tag string) bool {
	return tag == "meaning" || tag == "unsupported" || tag == "completion"
}
func summarizeHumanCases(cases []HumanCaseResult, split string) HumanCounts {
	c := HumanCounts{HardFailures: map[string]int{}, ByTask: map[string]map[string]int{}}
	for _, r := range cases {
		if r.Split != split {
			continue
		}
		c.Cases++
		for variant, tags := range r.Issues {
			for _, tag := range tags {
				if hardIssue(tag) {
					c.HardFailures[variant]++
					break
				}
			}
		}
		if r.Reviewed != r.Expected {
			continue
		}
		c.Complete++
		if c.ByTask[r.Task] == nil {
			c.ByTask[r.Task] = map[string]int{}
		}
		c.ByTask[r.Task][r.Winner]++
		switch r.Winner {
		case "baseline":
			c.Baseline++
		case "candidate":
			c.Candidate++
		case "tie":
			c.Ties++
		case "neither":
			c.Neither++
		case "mixed":
			c.Mixed++
		}
	}
	return c
}
func strongHumanPreference(c HumanCounts, variant string) bool {
	wins, losses := c.Candidate, c.Baseline
	if variant == "baseline" {
		wins, losses = losses, wins
	}
	if wins < 3 || wins*3 < 2*(wins+losses) || c.HardFailures[variant] > 0 || c.Neither > 0 || c.Mixed > 0 {
		return false
	}
	other := "baseline"
	if variant == "baseline" {
		other = "candidate"
	}
	for _, task := range c.ByTask {
		if task[other] > task[variant] {
			return false
		}
	}
	return true
}
func chooseHumanDirection(d, h HumanCounts) (string, string) {
	if d.Cases == 0 {
		return "collect-development", "This session contains only held-out cases. Use a development session to choose revisions; keep these judgments as validation evidence."
	}
	if d.Complete < d.Cases {
		return "continue-review", fmt.Sprintf("Review the remaining %d development cases before selecting a guide. Partial preferences are descriptive only.", d.Cases-d.Complete)
	}
	if strongHumanPreference(d, "candidate") {
		if h.Cases == 0 || h.Complete < h.Cases {
			return "validate-candidate", "Development preferences favor the candidate. Complete a separate held-out check before adopting it."
		}
		if h.HardFailures["candidate"] > 0 || h.Baseline > h.Candidate || h.Neither > 0 || h.Mixed > 0 {
			return "revise-candidate", "Development favors the candidate, but held-out judgments show unresolved failures or conflicting preferences. Revise using development feedback and validate on fresh held-out cases."
		}
		for _, task := range h.ByTask {
			if task["baseline"] > task["candidate"] {
				return "revise-candidate", "Development favors the candidate, but a held-out task category favors the baseline. Investigate using new development examples, then validate on fresh held-out cases."
			}
		}
		return "advance-candidate", "Development preferences favor the candidate, with no observed held-out regression under this policy. Advance it for the reviewed use cases; this is a human preference decision, not proof of general superiority."
	}
	if strongHumanPreference(d, "baseline") {
		return "retain-baseline", "Development preferences favor the baseline. Use it as the next experiment's starting point; retain candidate improvements only where individual development judgments support them."
	}
	if d.Candidate == 0 && d.Baseline == 0 && d.Neither == 0 && d.Mixed == 0 && d.HardFailures["candidate"] == 0 && d.HardFailures["baseline"] == 0 {
		return "no-preference", "All completed development cases tie. Keep the current guide; collect more discriminating tasks before changing it on quality grounds."
	}
	return "targeted-revision", "Development judgments do not support one guide across task categories. Use the tagged problems and preferred development passages to test one focused revision at a time."
}
func analyzeHuman(s HumanSuite, j HumanJournal) HumanAnalysis {
	a := HumanAnalysis{Policy: "human-direction-v1", Reviewer: s.Reviewer, Expected: len(s.Pairs), Cases: []HumanCaseResult{}, Findings: []HumanFinding{}}
	a.Reviewed, a.Skipped = humanProgress(s, j)
	votes := humanVotes(j)
	cases := map[string]*HumanCaseResult{}
	for _, p := range s.Pairs {
		c := cases[p.Case.ID]
		if c == nil {
			c = &HumanCaseResult{ID: p.Case.ID, Split: p.Case.Split, Task: p.Case.Task, Issues: map[string][]string{}}
			cases[p.Case.ID] = c
		}
		c.Expected++
		v, ok := votes[p.ID]
		if !ok || v.Choice == "skip" {
			continue
		}
		c.Reviewed++
		switch v.Choice {
		case "tie":
			c.Ties++
		case "neither":
			c.Neither++
		default:
			if mappedWinner(strings.ToUpper(v.Choice), p.AVariant) == "candidate" {
				c.Candidate++
			} else {
				c.Baseline++
			}
		}
		for side, tags := range map[string][]string{"A": v.IssuesA, "B": v.IssuesB} {
			variant := mappedWinner(side, p.AVariant)
			for _, tag := range tags {
				c.Issues[variant] = append(c.Issues[variant], tag)
				a.Findings = append(a.Findings, HumanFinding{CaseID: p.Case.ID, PairID: p.ID, Split: p.Case.Split, Variant: variant, Issue: tag})
			}
		}
	}
	for _, c := range cases {
		// Repetitions contribute to a case verdict, not extra independent case wins.
		c.Winner = "pending"
		if c.Reviewed == c.Expected {
			switch {
			case c.Neither == c.Reviewed:
				c.Winner = "neither"
			case c.Neither > 0:
				c.Winner = "mixed"
			case c.Candidate > c.Baseline:
				c.Winner = "candidate"
			case c.Baseline > c.Candidate:
				c.Winner = "baseline"
			case c.Candidate > 0:
				c.Winner = "mixed"
			default:
				c.Winner = "tie"
			}
		}
		for variant, tags := range c.Issues {
			c.Issues[variant] = uniqueStrings(tags)
		}
		a.Cases = append(a.Cases, *c)
	}
	sort.Slice(a.Cases, func(i, j int) bool { return a.Cases[i].ID < a.Cases[j].ID })
	sort.Slice(a.Findings, func(i, j int) bool { return digest(a.Findings[i]) < digest(a.Findings[j]) })
	a.Development = summarizeHumanCases(a.Cases, "development")
	a.HeldOut = summarizeHumanCases(a.Cases, "held-out")
	a.Direction, a.Reason = chooseHumanDirection(a.Development, a.HeldOut)
	return a
}

var humanExperiments = map[string]string{
	"meaning":     "Test an edit that explicitly preserves the original scope, conditions, and uncertainty; include an acceptable sentence that should remain unchanged.",
	"unsupported": "Test a narrower instruction against adding facts or context; pair the failure with a task that explicitly authorizes elaboration.",
	"completion":  "Clarify the requested output format and task mode; test both the failed format and a neighboring format.",
	"verbose":     "Test a short instruction to remove repeated claims and explanations already evident from the supplied facts; retain an example that genuinely requires explanation.",
	"agency":      "Test subject and predicate corrections alongside valid operations, dependencies, and established technical terminology.",
	"vague":       "Test replacing the flagged wording with a concrete, supported claim, alongside a useful technical abstraction.",
	"voice":       "Use the preferred passage and reviewer note to formulate one voice change; compare against the published samples.",
	"structure":   "Test one paragraph-order or connection change without adding arguments.",
	"overedit":    "Test an explicit instruction to leave effective wording unchanged, alongside a passage requiring a substantive correction.",
	"other":       "Use the reviewer note to state a general failure and a neighboring acceptable case before changing the guide.",
}

func writeHumanReport(dir string, s HumanSuite, j HumanJournal, out io.Writer) error {
	a := analyzeHuman(s, j)
	if err := WriteJSON(filepath.Join(dir, "human-report.json"), a); err != nil {
		return err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# Human writing review\n\nReviewer: %s\n\n**Direction: %s**\n\n%s\n\nReviewed %d/%d comparisons; %d skipped.\n\n", humanText(s.Reviewer), a.Direction, a.Reason, a.Reviewed, a.Expected, a.Skipped)
	fmt.Fprintf(&b, "Baseline: `%s` (SHA-256 `%s`).\n\nCandidate: `%s` (SHA-256 `%s`).\n\n", s.Sources["baseline"].Path, Hash([]byte(s.Sources["baseline"].Content)), s.Sources["candidate"].Path, Hash([]byte(s.Sources["candidate"].Content)))
	b.WriteString("## Case preferences\n\nRepetitions are collapsed within each case. Only fully reviewed cases enter the preference counts. A tied vote count with opposing preferences, or any mixture including neither, is mixed.\n\n| Set | Complete | Baseline | Candidate | Tie | Neither | Mixed |\n| --- | ---: | ---: | ---: | ---: | ---: | ---: |\n")
	for _, item := range []struct {
		name string
		c    HumanCounts
	}{{"Development", a.Development}, {"Held-out", a.HeldOut}} {
		c := item.c
		fmt.Fprintf(&b, "| %s | %d/%d | %d | %d | %d | %d | %d |\n", item.name, c.Complete, c.Cases, c.Baseline, c.Candidate, c.Ties, c.Neither, c.Mixed)
	}
	b.WriteString("\n## Task categories\n\n| Set | Task | Baseline | Candidate | Tie | Neither | Mixed |\n| --- | --- | ---: | ---: | ---: | ---: | ---: |\n")
	for _, item := range []struct {
		name string
		c    HumanCounts
	}{{"Development", a.Development}, {"Held-out", a.HeldOut}} {
		for _, task := range []string{"draft", "edit", "review"} {
			c, ok := item.c.ByTask[task]
			if ok {
				fmt.Fprintf(&b, "| %s | %s | %d | %d | %d | %d | %d |\n", item.name, task, c["baseline"], c["candidate"], c["tie"], c["neither"], c["mixed"])
			}
		}
	}
	b.WriteString("\n## Decision policy\n\nThis is a transparent heuristic, not a significance test. Complete all development cases first. A strong preference requires at least three case wins, at least two-thirds of decisive cases, no task category favoring the other guide, no mixed or neither case, and no meaning, unsupported-addition, or completion flag on the preferred guide. Candidate advancement also requires completed held-out coverage with no such candidate flags, no overall or task-category preference for the baseline, and no mixed or neither cases. Otherwise the program recommends more review or focused revision. Equal votes do not establish equivalence.\n\nIssue tags are explicit reviewer observations. Case rule labels are not treated as diagnoses. A vote for one response can coexist with problems in both. Notes and preferences come only from this reviewer; model judgments are not included.\n")
	b.WriteString("\n## Judgments\n\n")
	votes := humanVotes(j)
	for _, p := range s.Pairs {
		v, ok := votes[p.ID]
		if !ok {
			continue
		}
		choice := v.Choice
		if choice == "a" || choice == "b" {
			choice = mappedWinner(strings.ToUpper(choice), p.AVariant)
		}
		fmt.Fprintf(&b, "- %s (%s, %s): %s. A = %s. A issues: %v; B issues: %v.\n", p.ID, p.Case.Split, p.Case.Task, choice, p.AVariant, v.IssuesA, v.IssuesB)
		if v.Note != "" {
			fmt.Fprintf(&b, "  Reviewer note: %s\n", humanText(v.Note))
		}
	}
	b.WriteString("\nThe frozen outputs and guide snapshots are in `suite.json`. Keep this report closed during blind review. See `next-experiment.md` for development feedback and proposed tests.\n")
	if err := os.WriteFile(filepath.Join(dir, "human-report.md"), []byte(b.String()), 0600); err != nil {
		return err
	}
	var plan strings.Builder
	fmt.Fprintf(&plan, "# Next guide experiment\n\nDirection: **%s**\n\n%s\n\n", a.Direction, a.Reason)
	plan.WriteString("Use this as an experiment brief, not as permission to change or publish the guide automatically. Preserve the current guide and results. Make one proposed revision, use the current guide as baseline, generate fresh development outputs with the same writer settings, and collect another blind review. Do not reuse a prior session for changed outputs.\n\nHeld-out passage text and notes are excluded below. Once held-out findings influence a revision, retire those cases from validation and collect fresh held-out cases.\n\n")
	if a.Development.Complete < a.Development.Cases {
		plan.WriteString("Review coverage is incomplete. The observations below are provisional; finish review before selecting a direction.\n\n")
	}
	type target struct {
		variant, tag string
		cases        map[string]bool
	}
	targets := map[string]*target{}
	for _, f := range a.Findings {
		if f.Split != "development" {
			continue
		}
		key := f.Variant + "/" + f.Issue
		if targets[key] == nil {
			targets[key] = &target{variant: f.Variant, tag: f.Issue, cases: map[string]bool{}}
		}
		targets[key].cases[f.CaseID] = true
	}
	sorted := []*target{}
	for _, t := range targets {
		sorted = append(sorted, t)
	}
	sort.Slice(sorted, func(i, j int) bool {
		if len(sorted[i].cases) != len(sorted[j].cases) {
			return len(sorted[i].cases) > len(sorted[j].cases)
		}
		return sorted[i].variant+sorted[i].tag < sorted[j].variant+sorted[j].tag
	})
	plan.WriteString("## Revision targets\n\n")
	if len(sorted) == 0 {
		plan.WriteString("No development issue tags recorded. Use the preferences and notes below, or collect specific issue tags before attributing a failure to a rule.\n\n")
	}
	for _, t := range sorted {
		ids := []string{}
		for id := range t.cases {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		fmt.Fprintf(&plan, "- %s / %s: %d distinct cases (%s). %s\n", t.variant, t.tag, len(ids), strings.Join(ids, ", "), humanExperiments[t.tag])
	}
	plan.WriteString("\n## Development evidence\n\n")
	for _, p := range s.Pairs {
		v, ok := votes[p.ID]
		if !ok || p.Case.Split != "development" || v.Choice == "skip" {
			continue
		}
		if v.Choice == "tie" && len(v.IssuesA)+len(v.IssuesB) == 0 && v.Note == "" {
			continue
		}
		fmt.Fprintf(&plan, "### %s\n\nTask: %s\n\nPreference: %s. A = %s.\n\nA issues: %v. B issues: %v.\n\nReviewer note: %s\n\nTask input:\n\n", p.ID, p.Case.Task, v.Choice, p.AVariant, v.IssuesA, v.IssuesB, humanText(v.Note))
		quoteHuman(&plan, p.Case.Input)
		plan.WriteString("\nResponse A:\n\n")
		quoteHuman(&plan, p.A)
		plan.WriteString("\nResponse B:\n\n")
		quoteHuman(&plan, p.B)
		plan.WriteString("\n")
	}
	if err := os.WriteFile(filepath.Join(dir, "next-experiment.md"), []byte(plan.String()), 0600); err != nil {
		return err
	}
	fmt.Fprintf(out, "%s\n%s\nReviewed %d/%d comparisons.\nReport: %s\nExperiment brief: %s\n", a.Direction, a.Reason, a.Reviewed, a.Expected, filepath.Join(dir, "human-report.md"), filepath.Join(dir, "next-experiment.md"))
	return nil
}
func quoteHuman(b *strings.Builder, text string) {
	for _, line := range strings.Split(humanText(text), "\n") {
		fmt.Fprintf(b, "> %s\n", line)
	}
}
func HumanReport(dir string, out io.Writer) error {
	lock, err := humanLock(dir)
	if err != nil {
		return err
	}
	defer lock.Close()
	s, j, err := loadHuman(dir)
	if err != nil {
		return err
	}
	return writeHumanReport(dir, s, j, out)
}
