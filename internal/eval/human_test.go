package eval

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func humanFixture(t *testing.T, all bool) (string, string, HumanSuite) {
	t.Helper()
	_, run := fixture(t)
	if err := Run(context.Background(), run, "generate", &fakeProvider{}, false, io.Discard); err != nil {
		t.Fatal(err)
	}
	session := filepath.Join(t.TempDir(), "human")
	if err := HumanReview(HumanOptions{Session: session, Runs: []string{run}, Reviewer: "Tyson", AllRepetitions: all, Init: true}, strings.NewReader(""), io.Discard); err != nil {
		t.Fatal(err)
	}
	s, _, err := loadHuman(session)
	if err != nil {
		t.Fatal(err)
	}
	return run, session, s
}
func TestHumanBlindReviewResumeAndMapping(t *testing.T) {
	_, session, s := humanFixture(t, false)
	if len(s.Pairs) != 1 {
		t.Fatal("default did not select one pair per case")
	}
	var out bytes.Buffer
	if err := HumanReview(HumanOptions{Session: session, Limit: 1}, strings.NewReader("a\na:verbose\nKeep this direction, but remove repetition.\n"), &out); err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"old guide", "new guide", "baseline", "candidate", "SECRET_CRITERION", "SECRET_REFERENCE", "secret-rule"} {
		if strings.Contains(out.String(), secret) {
			t.Fatalf("blind UI leaked %q", secret)
		}
	}
	_, j, err := loadHuman(session)
	if err != nil {
		t.Fatal(err)
	}
	a := analyzeHuman(s, j)
	if a.Reviewed != 1 || a.Cases[0].Winner != s.Pairs[0].AVariant {
		t.Fatalf("incorrect side mapping: %+v", a)
	}
	if len(a.Findings) != 1 || a.Findings[0].Variant != s.Pairs[0].AVariant {
		t.Fatal("incorrect issue mapping")
	}
	out.Reset()
	if err = HumanReview(HumanOptions{Session: session}, strings.NewReader("b\n\n\n"), &out); err != nil {
		t.Fatal(err)
	}
	_, after, err := loadHuman(session)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Events) != 1 || strings.Contains(out.String(), "RESPONSE A") {
		t.Fatal("repeated a completed comparison")
	}
	if err = HumanReview(HumanOptions{Session: session, Reviewer: "Someone else"}, strings.NewReader(""), io.Discard); err == nil {
		t.Fatal("reviewer silently changed")
	}
}
func TestHumanUndoSkipAndInterruptedEntry(t *testing.T) {
	_, session, s := humanFixture(t, true)
	var out bytes.Buffer
	// Save, undo from the next comparison, correct, then skip remaining pairs.
	if err := HumanReview(HumanOptions{Session: session}, strings.NewReader("a\n\nfirst\nu\nb\n\ncorrected\ns\ns\n"), &out); err != nil {
		t.Fatal(err)
	}
	_, j, err := loadHuman(session)
	if err != nil {
		t.Fatal(err)
	}
	votes := humanVotes(j)
	if votes[s.Pairs[0].ID].Choice != "b" || len(j.Events) != 5 {
		t.Fatalf("undo lost history: %+v", j)
	}
	done, skipped := humanProgress(s, j)
	if done != 1 || skipped != 2 {
		t.Fatal(done, skipped)
	}
	out.Reset()
	if err = HumanReview(HumanOptions{Session: session, Limit: 1}, strings.NewReader("tie\n\n\n"), &out); err != nil {
		t.Fatal(err)
	}
	_, j, err = loadHuman(session)
	if err != nil {
		t.Fatal(err)
	}
	done, skipped = humanProgress(s, j)
	if done != 2 || skipped != 1 {
		t.Fatal("skipped item was not revisited")
	}
	n := len(j.Events)
	if err = HumanReview(HumanOptions{Session: session}, strings.NewReader("a\n"), io.Discard); err != nil {
		t.Fatal(err)
	}
	_, j, err = loadHuman(session)
	if err != nil {
		t.Fatal(err)
	}
	if len(j.Events) != n {
		t.Fatal("saved incomplete judgment on EOF")
	}
}
func TestHumanRejectsIncompleteOverlappingAndMismatchedRuns(t *testing.T) {
	run, _, _ := humanFixture(t, false)
	if _, err := newHuman(HumanOptions{Runs: []string{run, run}, Reviewer: "x"}); err == nil {
		t.Fatal("duplicate cases counted twice")
	}
	_, missing := fixture(t)
	if _, err := newHuman(HumanOptions{Runs: []string{missing}, Reviewer: "x"}); err == nil {
		t.Fatal("missing outputs accepted")
	}
	m, err := LoadManifest(run)
	if err != nil {
		t.Fatal(err)
	}
	m.Sources["candidate"] = Snapshot{Content: "changed guide"}
	if err = WriteJSON(filepath.Join(run, "manifest.json"), m); err != nil {
		t.Fatal(err)
	}
	if _, err = newHuman(HumanOptions{Runs: []string{run}, Reviewer: "x"}); err == nil {
		t.Fatal("changed writing request accepted")
	}
}
func TestHumanFrozenSessionAndTamperDetection(t *testing.T) {
	run, session, s := humanFixture(t, false)
	if err := os.Remove(filepath.Join(run, "generations.jsonl")); err != nil {
		t.Fatal(err)
	}
	if err := HumanReview(HumanOptions{Session: session}, strings.NewReader("tie\n\n\n"), io.Discard); err != nil {
		t.Fatal("frozen copy did not survive source removal", err)
	}
	s.Pairs[0].A = "changed output"
	if err := WriteJSON(filepath.Join(session, "suite.json"), s); err != nil {
		t.Fatal(err)
	}
	if _, _, err := loadHuman(session); err == nil {
		t.Fatal("changed snapshot reinterpreted old vote")
	}
}
func TestHumanLockReleased(t *testing.T) {
	dir := t.TempDir()
	f, err := humanLock(dir)
	if err != nil {
		t.Fatal(err)
	}
	if other, err := humanLock(dir); err == nil {
		other.Close()
		t.Fatal("concurrent writer accepted")
	}
	f.Close()
	f, err = humanLock(dir)
	if err != nil {
		t.Fatal("stale lock after close", err)
	}
	f.Close()
}
func TestHumanInputAndTerminalEscapes(t *testing.T) {
	a, b, err := parseIssues("a:verbose, b:meaning a:verbose")
	if err != nil || len(a) != 1 || len(b) != 1 {
		t.Fatal(a, b, err)
	}
	if _, _, err = parseIssues("a:invented"); err == nil {
		t.Fatal("unknown issue accepted")
	}
	text := humanText("hello\x1b[2J\x07\r\u202eworld\nnext")
	if strings.ContainsAny(text, "\x1b\x07\r\u202e") {
		t.Fatal("terminal controls retained")
	}
}
func TestHumanDecisionPolicy(t *testing.T) {
	strong := HumanCounts{Cases: 5, Complete: 5, Candidate: 3, Ties: 2, HardFailures: map[string]int{}, ByTask: map[string]map[string]int{"edit": {"candidate": 3, "tie": 2}}}
	held := HumanCounts{Cases: 2, Complete: 2, Ties: 2, HardFailures: map[string]int{}, ByTask: map[string]map[string]int{"edit": {"tie": 2}}}
	check := func(d, h HumanCounts, want string) {
		t.Helper()
		got, _ := chooseHumanDirection(d, h)
		if got != want {
			t.Fatalf("got %s want %s", got, want)
		}
	}
	check(strong, held, "advance-candidate")
	incomplete := strong
	incomplete.Complete = 4
	check(incomplete, held, "continue-review")
	check(strong, HumanCounts{}, "validate-candidate")
	bad := strong
	bad.HardFailures = map[string]int{"candidate": 1}
	check(bad, held, "targeted-revision")
	badHeld := held
	badHeld.HardFailures = map[string]int{"candidate": 1}
	check(strong, badHeld, "revise-candidate")
	taskLoss := strong
	taskLoss.ByTask = map[string]map[string]int{"edit": {"candidate": 3}, "review": {"baseline": 1}}
	check(taskLoss, held, "targeted-revision")
	mixed := strong
	mixed.Mixed = 1
	check(mixed, held, "targeted-revision")
	weak := strong
	weak.Candidate = 2
	check(weak, held, "targeted-revision")
	baseline := strong
	baseline.Candidate = 0
	baseline.Baseline = 3
	baseline.ByTask = map[string]map[string]int{"edit": {"baseline": 3}}
	check(baseline, held, "retain-baseline")
	ties := held
	check(ties, held, "no-preference")
	check(HumanCounts{}, held, "collect-development")
}
func TestHumanRepetitionsDoNotInflateEvidence(t *testing.T) {
	_, _, s := humanFixture(t, true)
	j := HumanJournal{SuiteHash: digest(s)}
	for _, p := range s.Pairs {
		choice := "a"
		if p.AVariant != "candidate" {
			choice = "b"
		}
		j.Events = append(j.Events, HumanEvent{At: time.Now(), Action: "vote", Vote: HumanVote{PairID: p.ID, Choice: choice}})
	}
	a := analyzeHuman(s, j)
	if a.Reviewed != 3 || a.Development.Candidate != 1 || a.Direction != "targeted-revision" {
		t.Fatalf("repetitions counted as independent cases: %+v", a)
	}
}
func TestHumanBriefExcludesHeldOutEvidence(t *testing.T) {
	_, session, s := humanFixture(t, false)
	p := s.Pairs[0]
	p.ID = "private/0"
	p.Case.ID = "private"
	p.Case.Split = "held-out"
	p.Case.Input = "HELDOUT_TASK_SECRET"
	p.A = "HELDOUT_OUTPUT_SECRET"
	p.B = "HELDOUT_OTHER_SECRET"
	s.Pairs = append(s.Pairs, p)
	j := HumanJournal{SuiteHash: digest(s), Events: []HumanEvent{{Action: "vote", Vote: HumanVote{PairID: p.ID, Choice: "a", IssuesB: []string{"verbose"}, Note: "HELDOUT_NOTE_SECRET"}}, {Action: "vote", Vote: HumanVote{PairID: s.Pairs[0].ID, Choice: "b", IssuesA: []string{"agency"}, Note: "DEVELOPMENT_NOTE"}}}}
	if err := writeHumanReport(session, s, j, io.Discard); err != nil {
		t.Fatal(err)
	}
	brief, err := os.ReadFile(filepath.Join(session, "next-experiment.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(brief), "HELDOUT_") || !strings.Contains(string(brief), "DEVELOPMENT_NOTE") {
		t.Fatal("incorrect development evidence selection")
	}
	raw, err := os.ReadFile(filepath.Join(session, "human-report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var a HumanAnalysis
	if err = json.Unmarshal(raw, &a); err != nil {
		t.Fatal(err)
	}
	if a.Expected != 2 || a.Reviewed != 2 {
		t.Fatal("report missing reviewed pair")
	}
}

func TestHumanUndoAfterCompletion(t *testing.T) {
	_, session, s := humanFixture(t, false)
	if err := HumanReview(HumanOptions{Session: session}, strings.NewReader("a\n\n\n"), io.Discard); err != nil {
		t.Fatal(err)
	}
	if err := HumanReview(HumanOptions{Session: session, Undo: true}, strings.NewReader("b\n\ncorrected after completion\n"), io.Discard); err != nil {
		t.Fatal(err)
	}
	_, j, err := loadHuman(session)
	if err != nil {
		t.Fatal(err)
	}
	if len(j.Events) != 3 || humanVotes(j)[s.Pairs[0].ID].Choice != "b" {
		t.Fatal("could not correct a completed session")
	}
}
func TestHumanDisplayPreservesSpacing(t *testing.T) {
	var b bytes.Buffer
	printHumanText(&b, "Example:\n\n    fmt.Println(\"two  spaces\")")
	if b.String() != "Example:\n\n    fmt.Println(\"two  spaces\")\n" {
		t.Fatal("display changed source spacing")
	}
}

func TestHumanIdenticalResponsesStillRequireVote(t *testing.T) {
	_, session, _ := humanFixture(t, false)
	var out bytes.Buffer
	if err := HumanReview(HumanOptions{Session: session}, strings.NewReader("q\n"), &out); err != nil {
		t.Fatal(err)
	}
	s, j, err := loadHuman(session)
	if err != nil {
		t.Fatal(err)
	}
	done, _ := humanProgress(s, j)
	if done != 0 || !strings.Contains(out.String(), "RESPONSES A AND B (identical)") {
		t.Fatal("identical responses bypassed review")
	}
	a, b, err := parseIssues("both:verbose")
	if err != nil || len(a) != 1 || len(b) != 1 {
		t.Fatal("both tag failed")
	}
}
