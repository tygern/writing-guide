package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

func fixture(t *testing.T) (Manifest, string) {
	t.Helper()
	dir := t.TempDir()
	m := Manifest{Protocol: Protocol, Config: Config{Repetitions: 3, Concurrency: 3, TimeoutSeconds: 10, Writer: Model{Name: "writer", MaxOutputTokens: 100}, Judge: Model{Name: "judge", MaxOutputTokens: 1000}, Seed: 17}, Cases: []Case{{ID: "case-one", Task: "edit", Split: "development", Input: "Edit this sentence.", Rules: []string{"secret-rule"}, Criteria: []string{"SECRET_CRITERION"}, Example: "SECRET_REFERENCE"}}, Sources: map[string]Snapshot{"baseline": {Content: "old guide"}, "candidate": {Content: "new guide"}, "rubric": {Content: "fixed rubric"}, "samples": {Content: "published sample"}}}
	if err := WriteJSON(filepath.Join(dir, "manifest.json"), m); err != nil {
		t.Fatal(err)
	}
	return m, dir
}

type fakeProvider struct {
	mu    sync.Mutex
	calls int
	fail  bool
}

func (p *fakeProvider) Complete(ctx context.Context, r Request) (Response, error) {
	p.mu.Lock()
	p.calls++
	fail := p.fail
	p.mu.Unlock()
	if fail {
		return Response{}, fmt.Errorf("test failure")
	}
	if r.Schema == nil {
		if strings.Contains(r.Input+r.Instructions, "SECRET_") {
			return Response{}, fmt.Errorf("grading material leaked")
		}
		return Response{Status: "completed", Text: "A valid response.", Usage: Usage{Input: 12, Output: 4}}, nil
	}
	a := Assessment{Meaning: true, Restraint: true, Completion: true, Criteria: []Criterion{{Index: 0, Pass: true, Reason: "Meets criterion.", Evidence: "A valid response."}}}
	j := Judgment{A: a, B: a, Clarity: Preference{Winner: "tie", Reason: "Equivalent."}, Voice: Preference{Winner: "tie", Reason: "Equivalent."}}
	b, _ := json.Marshal(j)
	return Response{Status: "completed", Text: string(b), Usage: Usage{Input: 20, Output: 10}}, nil
}
func TestRunResumeAndIsolation(t *testing.T) {
	m, dir := fixture(t)
	p := &fakeProvider{}
	for _, stage := range []string{"generate", "generate", "judge", "judge"} {
		if err := Run(context.Background(), dir, stage, p, false, io.Discard); err != nil {
			t.Fatal(err)
		}
	}
	if p.calls != 12 {
		t.Fatalf("got %d calls; repetitions or resume incorrect", p.calls)
	}
	gen, err := readRecords(filepath.Join(dir, "generations.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(gen) != 6 {
		t.Fatal(len(gen))
	}
	judges, err := readRecords(filepath.Join(dir, "judgments.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	for rep := 0; rep < 3; rep++ {
		a := judges[fmt.Sprintf("case-one/%d/order-0", rep)]
		b := judges[fmt.Sprintf("case-one/%d/order-1", rep)]
		if a.AVariant == b.AVariant {
			t.Fatal("order did not swap")
		}
	}
	summary := makeSummary(m, gen, judges, nil)
	if summary.PairedComparisons != 3 || summary.Clarity.Tie != 3 {
		t.Fatalf("bad summary: %+v", summary)
	}
	if err = Report(dir); err != nil {
		t.Fatal(err)
	}
	html, err := os.ReadFile(filepath.Join(dir, "review.html"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(html), "old guide") || strings.Contains(string(html), "candidate") {
		t.Fatal("blind review exposed condition")
	}
}
func TestErrorsRequireExplicitRetry(t *testing.T) {
	_, dir := fixture(t)
	p := &fakeProvider{fail: true}
	if Run(context.Background(), dir, "generate", p, false, io.Discard) == nil {
		t.Fatal("expected failure")
	}
	if p.calls != 6 {
		t.Fatal(p.calls)
	}
	p.fail = false
	if Run(context.Background(), dir, "generate", p, false, io.Discard) == nil {
		t.Fatal("retained failures should remain visible")
	}
	if p.calls != 6 {
		t.Fatal("silently retried errors")
	}
	if err := Run(context.Background(), dir, "generate", p, true, io.Discard); err != nil {
		t.Fatal(err)
	}
	if p.calls != 12 {
		t.Fatal(p.calls)
	}
}
func TestOrderSensitivity(t *testing.T) {
	var c Counts
	addCount(&c, mappedWinner("A", "baseline"), mappedWinner("A", "candidate"))
	if c.OrderSensitive != 1 || c.Candidate != 0 || c.Baseline != 0 {
		t.Fatalf("position preference counted as a win: %+v", c)
	}
	addCount(&c, mappedWinner("B", "baseline"), mappedWinner("A", "candidate"))
	if c.Candidate != 1 {
		t.Fatal(c)
	}
}
func TestRejectFabricatedEvidence(t *testing.T) {
	a := Assessment{Meaning: true, Restraint: true, Completion: true, Criteria: []Criterion{{Index: 0, Pass: false, Reason: "Problem", Evidence: "invented quotation"}}}
	b, _ := json.Marshal(Judgment{A: a, B: a, Clarity: Preference{"tie", "same"}, Voice: Preference{"tie", "same"}})
	if _, err := ParseJudgment(string(b), Case{Criteria: []string{"criterion"}}, "actual response", "actual response"); err == nil {
		t.Fatal("accepted fabricated evidence")
	}
}
func TestJournalRejectsPartialLine(t *testing.T) {
	p := filepath.Join(t.TempDir(), "records.jsonl")
	if err := os.WriteFile(p, []byte(`{"id":"x"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readRecords(p); err == nil {
		t.Fatal("accepted incomplete journal")
	}
}
func TestChangedInputRefusesResume(t *testing.T) {
	m, dir := fixture(t)
	p := &fakeProvider{}
	if err := Run(context.Background(), dir, "generate", p, false, io.Discard); err != nil {
		t.Fatal(err)
	}
	m.Sources["candidate"] = Snapshot{Content: "different guide"}
	if err := WriteJSON(filepath.Join(dir, "manifest.json"), m); err != nil {
		t.Fatal(err)
	}
	if Run(context.Background(), dir, "generate", p, false, io.Discard) == nil {
		t.Fatal("accepted changed prompt")
	}
	if p.calls != 6 {
		t.Fatal("made calls after mismatch")
	}
}
func TestOpenAIRequestAndIncompleteResponse(t *testing.T) {
	for _, status := range []string{"completed", "incomplete"} {
		t.Run(status, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/responses" {
					t.Errorf("wrong endpoint %s", r.URL.Path)
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body["store"] != false || body["instructions"] != "guide" || body["input"] != "task" {
					t.Errorf("unexpected request: %v", body)
				}
				if _, ok := body["previous_response_id"]; ok {
					t.Error("context leaked")
				}
				format := body["text"].(map[string]any)["format"].(map[string]any)
				if format["strict"] != true || format["type"] != "json_schema" {
					t.Error("schema not strict")
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{"id":"resp_test","object":"response","model":"test","status":%q,"output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"result","annotations":[]}]}],"usage":{"input_tokens":10,"input_tokens_details":{"cached_tokens":3},"output_tokens":2}}`, status)
			})
			transport := roundTripper(func(r *http.Request) (*http.Response, error) {
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, r)
				return w.Result(), nil
			})
			p := OpenAI{Client: openai.NewClient(option.WithAPIKey("test-key"), option.WithBaseURL("https://example.invalid/"), option.WithHTTPClient(&http.Client{Transport: transport}), option.WithMaxRetries(0))}
			out, err := p.Complete(context.Background(), Request{Model: Model{Name: "test", MaxOutputTokens: 100}, Instructions: "guide", Input: "task", Schema: JudgmentSchema()})
			if (err == nil) != (status == "completed") {
				t.Fatalf("status %s error %v", status, err)
			}
			if out.Text != "result" || out.Usage.CachedInput != 3 {
				t.Fatalf("lost output or usage: %+v", out)
			}
		})
	}
}
func TestHTMLIsEscaped(t *testing.T) {
	_, dir := fixture(t)
	p := &fakeProvider{}
	if err := Run(context.Background(), dir, "generate", p, false, io.Discard); err != nil {
		t.Fatal(err)
	}
	rows, err := readRecords(filepath.Join(dir, "generations.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	for key, r := range rows {
		r.Response.Text = "<script>alert('x')</script>"
		rows[key] = r
	}
	f, err := os.Create(filepath.Join(dir, "generations.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if err = json.NewEncoder(f).Encode(r); err != nil {
			t.Fatal(err)
		}
	}
	f.Close()
	if err = Report(dir); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "review.html"))
	if strings.Contains(string(b), "<script>alert('x')") {
		t.Fatal("unescaped model output")
	}
}

type roundTripper func(*http.Request) (*http.Response, error)

func (f roundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestMissingJudgmentBooleanRejected(t *testing.T) {
	j := `{"a":{"restraint":true,"completion":true,"criteria":[{"index":0,"pass":true,"reason":"ok","evidence":""}]},"b":{"meaning":true,"restraint":true,"completion":true,"criteria":[{"index":0,"pass":true,"reason":"ok","evidence":""}]},"clarity":{"winner":"tie","reason":"same"},"voice":{"winner":"tie","reason":"same"}}`
	if _, err := ParseJudgment(j, Case{Criteria: []string{"criterion"}}, "a", "b"); err == nil {
		t.Fatal("accepted missing boolean")
	}
}
func TestConfigSelectionAndFreeze(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"old.md", "new.md", "rubric.md"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "samples.md"), []byte("> exact quote\nannotation\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cases := `- id: chosen
  split: development
  task: edit
  input: hello
  rules: [one]
  criteria: [one]
- id: excluded
  split: development
  task: edit
  input: second
  rules: [one]
  criteria: [one]
`
	if err := os.WriteFile(filepath.Join(dir, "cases.yaml"), []byte(cases), 0600); err != nil {
		t.Fatal(err)
	}
	config := `cases: cases.yaml
baseline: old.md
candidate: new.md
rubric: rubric.md
samples: samples.md
split: development
case_ids: [chosen]
repetitions: 1
concurrency: 1
timeout_seconds: 10
retries: 0
writer: {name: test, max_output_tokens: 100}
judge: {name: test, max_output_tokens: 100}
`
	path := filepath.Join(dir, "run.yaml")
	if err := os.WriteFile(path, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	m, err := Initialize(path, filepath.Join(dir, "results"))
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Cases) != 1 || m.Cases[0].ID != "chosen" {
		t.Fatal("case selection leaked")
	}
	if m.Sources["samples"].Content != "> exact quote" {
		t.Fatal("annotations leaked to judge")
	}
	if _, err = Initialize(path, filepath.Join(dir, "results")); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "new.md"), []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = Initialize(path, filepath.Join(dir, "results")); err == nil {
		t.Fatal("changed guide accepted into frozen run")
	}
}

func TestEvidenceAllowsBoldButNotChangedWords(t *testing.T) {
	if !evidenceMatches("**Principle:** preserve the claim.", "Principle: preserve the claim.") {
		t.Fatal("format-only quotation rejected")
	}
	if evidenceMatches("**Principle:** preserve the claim.", "Principle: rewrite the claim.") {
		t.Fatal("changed words accepted")
	}
	if evidenceMatches("Preserve all facts.", "Preserve facts.") {
		t.Fatal("omitted word accepted")
	}
}
func TestRecheckDoesNotDoubleCountUsage(t *testing.T) {
	m, dir := fixture(t)
	r := Record{ID: "case-one/0/order-0", CaseID: "case-one", AVariant: "baseline", Error: "judge evidence is not an exact output quotation", Request: Request{Input: judgeInput(m.Cases[0], "**A** valid response.", "**A** valid response.")}, Response: Response{Status: "completed", Usage: Usage{Input: 10, Output: 5}}, Attempts: []Attempt{{Error: "validation error"}}}
	a := Assessment{Meaning: true, Restraint: true, Completion: true, Criteria: []Criterion{{Index: 0, Pass: true, Reason: "Meets criterion.", Evidence: "A valid response."}}}
	j := Judgment{A: a, B: a, Clarity: Preference{"tie", "same"}, Voice: Preference{"tie", "same"}}
	b, _ := json.Marshal(j)
	r.Response.Text = string(b)
	r.Hash = digest(r.Request)
	f, err := os.Create(filepath.Join(dir, "judgments.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.NewEncoder(f).Encode(r); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if err = Recheck(dir, io.Discard); err != nil {
		t.Fatal(err)
	}
	rows, err := readRecords(filepath.Join(dir, "judgments.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	revised := rows[r.ID]
	if revised.Error != "" || !revised.Revalidated || revised.ValidationNote == "" {
		t.Fatal("missing recheck metadata")
	}
	all, err := ReadAllRecords(filepath.Join(dir, "judgments.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	s := makeSummary(m, nil, rows, all)
	if s.Usage.Input != 10 || s.Usage.Output != 5 || s.Attempts != 1 {
		t.Fatalf("double-counted recheck: %+v", s)
	}
}
