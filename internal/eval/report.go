package eval

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Counts struct {
	Baseline       int `json:"baseline"`
	Candidate      int `json:"candidate"`
	Tie            int `json:"tie"`
	OrderSensitive int `json:"order_sensitive"`
}
type Flags struct {
	Meaning    int `json:"meaning"`
	Restraint  int `json:"restraint"`
	Completion int `json:"completion"`
	Criteria   int `json:"criteria"`
}
type Summary struct {
	ExpectedGenerations  int                       `json:"expected_generations"`
	CompletedGenerations int                       `json:"completed_generations"`
	FailedGenerations    int                       `json:"failed_generations"`
	MissingGenerations   int                       `json:"missing_generations"`
	ExpectedJudgments    int                       `json:"expected_judgments"`
	CompletedJudgments   int                       `json:"completed_judgments"`
	FailedJudgments      int                       `json:"failed_judgments"`
	MissingJudgments     int                       `json:"missing_judgments"`
	PairedComparisons    int                       `json:"paired_comparisons"`
	Clarity              Counts                    `json:"clarity"`
	Voice                Counts                    `json:"voice"`
	Flags                map[string]Flags          `json:"flags"`
	ByTask               map[string]Counts         `json:"clarity_by_task"`
	ByRule               map[string]map[string]int `json:"criterion_flags_by_rule"`
	Usage                Usage                     `json:"usage"`
	Attempts             int                       `json:"attempts"`
	Failures             []string                  `json:"failures"`
}
type ReviewPair struct {
	ID, Task, Input, A, B string
	Criteria              []string
}

func mappedWinner(w, av string) string {
	if w == "tie" {
		return "tie"
	}
	if w == "A" {
		return av
	}
	if av == "candidate" {
		return "baseline"
	}
	return "candidate"
}
func addCount(c *Counts, a, b string) {
	if a != b {
		c.OrderSensitive++
		return
	}
	switch a {
	case "baseline":
		c.Baseline++
	case "candidate":
		c.Candidate++
	case "tie":
		c.Tie++
	}
}
func assessments(r Record, variant string) Assessment {
	if r.AVariant == variant {
		return r.Judgment.A
	}
	return r.Judgment.B
}
func makeSummary(m Manifest, gen, judges map[string]Record, all []Record) Summary {
	s := Summary{ExpectedGenerations: len(m.Cases) * m.Config.Repetitions * 2, ExpectedJudgments: len(m.Cases) * m.Config.Repetitions * 2, Flags: map[string]Flags{}, ByTask: map[string]Counts{}, ByRule: map[string]map[string]int{}, Failures: []string{}}
	for _, r := range gen {
		if r.Error == "" && r.Response.Status == "completed" {
			s.CompletedGenerations++
		} else {
			s.FailedGenerations++
		}
	}
	for _, r := range judges {
		if r.Error == "" && r.Judgment != nil {
			s.CompletedJudgments++
		} else {
			s.FailedJudgments++
		}
	}
	s.MissingGenerations = s.ExpectedGenerations - len(gen)
	s.MissingJudgments = s.ExpectedJudgments - len(judges)
	for _, r := range all {
		s.Usage.Input += r.Response.Usage.Input
		s.Usage.CachedInput += r.Response.Usage.CachedInput
		s.Usage.Output += r.Response.Usage.Output
		s.Attempts += len(r.Attempts)
	}
	for _, c := range m.Cases {
		for rep := 0; rep < m.Config.Repetitions; rep++ {
			r0, ok0 := judges[fmt.Sprintf("%s/%d/order-0", c.ID, rep)]
			r1, ok1 := judges[fmt.Sprintf("%s/%d/order-1", c.ID, rep)]
			if !ok0 || !ok1 || r0.Error != "" || r1.Error != "" || r0.Judgment == nil || r1.Judgment == nil {
				continue
			}
			s.PairedComparisons++
			ca, cb := mappedWinner(r0.Judgment.Clarity.Winner, r0.AVariant), mappedWinner(r1.Judgment.Clarity.Winner, r1.AVariant)
			addCount(&s.Clarity, ca, cb)
			count := s.ByTask[c.Task]
			addCount(&count, ca, cb)
			s.ByTask[c.Task] = count
			addCount(&s.Voice, mappedWinner(r0.Judgment.Voice.Winner, r0.AVariant), mappedWinner(r1.Judgment.Voice.Winner, r1.AVariant))
			for _, variant := range []string{"baseline", "candidate"} {
				a, b := assessments(r0, variant), assessments(r1, variant)
				flags := s.Flags[variant]
				if !a.Meaning || !b.Meaning {
					flags.Meaning++
				}
				if !a.Restraint || !b.Restraint {
					flags.Restraint++
				}
				if !a.Completion || !b.Completion {
					flags.Completion++
				}
				bad := map[int]bool{}
				for _, assessment := range []Assessment{a, b} {
					for _, criterion := range assessment.Criteria {
						if !criterion.Pass {
							bad[criterion.Index] = true
						}
					}
				}
				if len(bad) > 0 {
					flags.Criteria++
					for _, rule := range c.Rules {
						if s.ByRule[rule] == nil {
							s.ByRule[rule] = map[string]int{}
						}
						s.ByRule[rule][variant]++
					}
					indexes := make([]int, 0, len(bad))
					for index := range bad {
						indexes = append(indexes, index)
					}
					sort.Ints(indexes)
					for _, index := range indexes {
						s.Failures = append(s.Failures, fmt.Sprintf("%s / repetition %d / %s: %s", c.ID, rep+1, variant, c.Criteria[index]))
					}
				}
				s.Flags[variant] = flags
			}
		}
	}
	return s
}
func Report(dir string) error {
	m, err := LoadManifest(dir)
	if err != nil {
		return err
	}
	gen, err := readRecords(filepath.Join(dir, "generations.jsonl"))
	if err != nil {
		return err
	}
	judges, err := readRecords(filepath.Join(dir, "judgments.jsonl"))
	if err != nil {
		return err
	}
	all, err := ReadAllRecords(filepath.Join(dir, "generations.jsonl"))
	if err != nil {
		return err
	}
	more, err := ReadAllRecords(filepath.Join(dir, "judgments.jsonl"))
	if err != nil {
		return err
	}
	all = append(all, more...)
	s := makeSummary(m, gen, judges, all)
	if err = WriteJSON(filepath.Join(dir, "report.json"), s); err != nil {
		return err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# Writing evaluation\n\nSplit: %s. Writing model: %s. Judge: %s. Repetitions: %d.\n\n", m.Config.Split, m.Config.Writer.Name, m.Config.Judge.Name, m.Config.Repetitions)
	fmt.Fprintf(&b, "Generation: %d completed, %d failed, %d missing of %d. Judgment: %d completed, %d failed, %d missing of %d.\n\n", s.CompletedGenerations, s.FailedGenerations, s.MissingGenerations, s.ExpectedGenerations, s.CompletedJudgments, s.FailedJudgments, s.MissingJudgments, s.ExpectedJudgments)
	fmt.Fprintf(&b, "%d comparison pairs have both judge orders. Preference counts require agreement across orders; disagreements are reported separately. Repetitions of a case are not independent evidence of generalization.\n\n", s.PairedComparisons)
	b.WriteString("| Preference | Baseline | Candidate | Tie | Order-sensitive |\n| --- | ---: | ---: | ---: | ---: |\n")
	for _, row := range []struct {
		Name string
		C    Counts
	}{{"Clarity", s.Clarity}, {"Voice", s.Voice}} {
		fmt.Fprintf(&b, "| %s | %d | %d | %d | %d |\n", row.Name, row.C.Baseline, row.C.Candidate, row.C.Tie, row.C.OrderSensitive)
	}
	b.WriteString("\nFlags below count an output once if either judge order reports a failure. These are unverified model judgments, not human labels. Denominator for each variant is the number of fully judged pairs.\n\n| Variant | Meaning | Unnecessary edits/scope | Completion | Any criterion |\n| --- | ---: | ---: | ---: | ---: |\n")
	for _, v := range []string{"baseline", "candidate"} {
		f := s.Flags[v]
		fmt.Fprintf(&b, "| %s | %d | %d | %d | %d |\n", v, f.Meaning, f.Restraint, f.Completion, f.Criteria)
	}
	b.WriteString("\n## Clarity by task\n\n| Task | Baseline | Candidate | Tie | Order-sensitive |\n| --- | ---: | ---: | ---: | ---: |\n")
	for _, task := range []string{"draft", "edit", "review"} {
		c := s.ByTask[task]
		fmt.Fprintf(&b, "| %s | %d | %d | %d | %d |\n", task, c.Baseline, c.Candidate, c.Tie, c.OrderSensitive)
	}
	b.WriteString("\n## Criterion flags by associated rule\n\nCases can have several rules, so these counts overlap. A flag is associated with a case's rule labels; it is not proof that every associated rule was violated.\n\n| Rule | Baseline | Candidate |\n| --- | ---: | ---: |\n")
	rules := make([]string, 0, len(s.ByRule))
	for r := range s.ByRule {
		rules = append(rules, r)
	}
	sort.Strings(rules)
	for _, r := range rules {
		fmt.Fprintf(&b, "| %s | %d | %d |\n", r, s.ByRule[r]["baseline"], s.ByRule[r]["candidate"])
	}
	fmt.Fprintf(&b, "\n## Usage\n\nRecorded attempts: %d. Input tokens: %d (%d cached). Output tokens: %d. Usage unavailable from failed network attempts is not included; consult provider billing for actual cost.\n\n", s.Attempts, s.Usage.Input, s.Usage.CachedInput, s.Usage.Output)
	b.WriteString("## Criteria flagged for review\n\n")
	if len(s.Failures) == 0 {
		b.WriteString("No criterion failures recorded in fully judged pairs. Inspect coverage above before interpreting this statement.\n")
	}
	for _, line := range s.Failures {
		fmt.Fprintf(&b, "- %s\n", line)
	}
	b.WriteString("\n## Responses and judgments\n\n")
	for _, c := range m.Cases {
		for rep := 0; rep < m.Config.Repetitions; rep++ {
			fmt.Fprintf(&b, "### %s, repetition %d\n\nTask: %s\n\n", c.ID, rep+1, c.Input)
			for _, v := range []string{"baseline", "candidate"} {
				r, ok := gen[fmt.Sprintf("%s/%d/%s", c.ID, rep, v)]
				if !ok {
					continue
				}
				fmt.Fprintf(&b, "**%s**\n\n", v)
				for _, line := range strings.Split(r.Response.Text, "\n") {
					fmt.Fprintf(&b, "> %s\n", line)
				}
				if r.Error != "" {
					fmt.Fprintf(&b, "\nExecution error: %s\n", r.Error)
				}
				b.WriteString("\n")
			}
			for order := 0; order < 2; order++ {
				r, ok := judges[fmt.Sprintf("%s/%d/order-%d", c.ID, rep, order)]
				if !ok {
					continue
				}
				fmt.Fprintf(&b, "Judge order %d (A = %s):\n\n", order, r.AVariant)
				if r.Judgment == nil {
					fmt.Fprintf(&b, "Error: %s\n\n", r.Error)
					continue
				}
				fmt.Fprintf(&b, "- Clarity: %s. %s\n- Voice: %s. %s\n", mappedWinner(r.Judgment.Clarity.Winner, r.AVariant), r.Judgment.Clarity.Reason, mappedWinner(r.Judgment.Voice.Winner, r.AVariant), r.Judgment.Voice.Reason)
				for _, v := range []string{"baseline", "candidate"} {
					for _, a := range assessments(r, v).Criteria {
						if !a.Pass {
							fmt.Fprintf(&b, "- %s criterion %d: %s Evidence: %q\n", v, a.Index, a.Reason, a.Evidence)
						}
					}
				}
				b.WriteString("\n")
			}
		}
	}
	if err = os.WriteFile(filepath.Join(dir, "report.md"), []byte(b.String()), 0600); err != nil {
		return err
	}
	jobs, _ := judgeJobs(m, gen)
	var pairs []ReviewPair
	mapping := map[string]string{}
	for _, j := range jobs {
		if j.Record.Order != 0 {
			continue
		}
		id := fmt.Sprintf("%s/%d", j.Case.ID, j.Record.Repetition)
		pairs = append(pairs, ReviewPair{id, j.Case.Task, j.Case.Input, j.A, j.B, j.Case.Criteria})
		mapping[id] = j.Record.AVariant
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].ID < pairs[j].ID })
	if err = WriteJSON(filepath.Join(dir, "review-map.json"), mapping); err != nil {
		return err
	}
	t, err := template.New("review").Parse(reviewTemplate)
	if err != nil {
		return err
	}
	var html bytes.Buffer
	err = t.Execute(&html, struct {
		Key   string
		Pairs []ReviewPair
	}{digest(m), pairs})
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "review.html"), html.Bytes(), 0600)
}

const reviewTemplate = `<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Blind writing review</title>
<style>body{font:17px/1.5 system-ui;margin:2rem auto;max-width:1100px;padding:0 1rem;color:#17212b;background:#fafafa}section{margin:2rem 0;border-top:1px solid #aaa;padding-top:1rem}.outputs{display:grid;grid-template-columns:1fr 1fr;gap:1rem}pre{white-space:pre-wrap;overflow-wrap:anywhere;font:inherit;background:white;padding:1rem;border:1px solid #ddd}label{display:block;margin:.5rem 0}textarea{width:95%;min-height:5rem}button{padding:.6rem 1rem}@media(max-width:650px){.outputs{grid-template-columns:1fr}}</style>
<body data-run="{{.Key}}"><h1>Blind writing review</h1><p>Compare the outputs against the task and criteria. Guide identities and model judgments are hidden. Export reviews before closing if local browser storage is unavailable. Open report.md only after recording preferences.</p><button id="export">Export reviews as JSON</button><span id="status" role="status"></span>
{{range .Pairs}}<section data-id="{{.ID}}"><h2>{{.ID}}</h2><p>{{.Input}}</p><ul>{{range .Criteria}}<li>{{.}}</li>{{end}}</ul><div class="outputs"><div><h3>A</h3><pre>{{.A}}</pre></div><div><h3>B</h3><pre>{{.B}}</pre></div></div>
<label>Clarity <select name="clarity"><option value="">Unreviewed</option><option>A</option><option>B</option><option>tie</option></select></label>
<label>Voice <select name="voice"><option value="">Unreviewed</option><option>A</option><option>B</option><option>tie</option></select></label>
<label>A preserves meaning <select name="a_meaning"><option value="">Unreviewed</option><option>pass</option><option>fail</option></select></label>
<label>B preserves meaning <select name="b_meaning"><option value="">Unreviewed</option><option>pass</option><option>fail</option></select></label>
<label>Notes<textarea name="notes"></textarea></label></section>{{end}}
<script>
const key='writing-review-'+document.body.dataset.run;let saved={};try{saved=JSON.parse(localStorage.getItem(key)||'{}')}catch(e){}
const sections=[...document.querySelectorAll('section[data-id]')];for(const s of sections){for(const f of s.querySelectorAll('[name]')){f.value=(saved[s.dataset.id]||{})[f.name]||'';f.addEventListener('input',save)}}
function values(){return Object.fromEntries(sections.map(s=>[s.dataset.id,Object.fromEntries([...s.querySelectorAll('[name]')].map(f=>[f.name,f.value]))]))}
function save(){try{localStorage.setItem(key,JSON.stringify(values()));document.getElementById('status').textContent=' Saved locally.'}catch(e){document.getElementById('status').textContent=' Export to save your review.'}}
document.getElementById('export').onclick=()=>{const blob=new Blob([JSON.stringify({run:document.body.dataset.run,reviews:values()},null,2)],{type:'application/json'});const url=URL.createObjectURL(blob);const a=document.createElement('a');a.href=url;a.download='human-reviews.json';a.click();setTimeout(()=>URL.revokeObjectURL(url),1000)};
</script></body></html>`
