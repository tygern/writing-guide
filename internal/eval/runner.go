package eval

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type job struct {
	Record Record
	Case   Case
	A, B   string
}

func Initialize(configPath, dir string) (Manifest, error) {
	c, err := LoadConfig(configPath)
	if err != nil {
		return Manifest{}, err
	}
	all, err := LoadCases(c.Cases)
	if err != nil {
		return Manifest{}, err
	}
	selected := map[string]bool{}
	for _, id := range c.CaseIDs {
		if selected[id] {
			return Manifest{}, fmt.Errorf("duplicate selected case: %s", id)
		}
		selected[id] = true
	}
	m := Manifest{Protocol: Protocol, Created: time.Now().UTC(), Config: c, Sources: map[string]Snapshot{}}
	for _, v := range all {
		if v.Split == c.Split && (len(c.CaseIDs) == 0 || selected[v.ID]) {
			m.Cases = append(m.Cases, v)
			delete(selected, v.ID)
		}
	}
	if len(selected) > 0 || len(m.Cases) == 0 {
		return m, fmt.Errorf("no matching cases or unknown case_ids: %v", selected)
	}
	for name, path := range map[string]string{"baseline": c.Baseline, "candidate": c.Candidate, "cases": c.Cases, "rubric": c.Rubric, "samples": c.Samples} {
		b, e := os.ReadFile(path)
		if e != nil {
			return m, e
		}
		content := string(b)
		if name == "baseline" && strings.HasPrefix(content, "<!-- Historical source;") {
			if at := strings.Index(content, "-->\n"); at >= 0 {
				content = strings.TrimSpace(content[at+4:]) + "\n"
			}
		}
		if name == "samples" {
			var quotes []string
			for _, line := range strings.Split(content, "\n") {
				if strings.HasPrefix(line, ">") {
					quotes = append(quotes, line)
				}
			}
			content = strings.Join(quotes, "\n")
			if content == "" {
				return m, fmt.Errorf("no published quotations in samples")
			}
		}
		m.Sources[name] = Snapshot{Path: path, SHA256: Hash([]byte(content)), Content: content}
	}
	root := filepath.Dir(c.Candidate)
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = root
	if b, e := cmd.Output(); e == nil {
		m.Revision = strings.TrimSpace(string(b))
	}
	cmd = exec.Command("git", "status", "--porcelain")
	cmd.Dir = root
	if b, e := cmd.Output(); e == nil {
		m.Dirty = len(b) > 0
	}
	if err = os.MkdirAll(dir, 0700); err != nil {
		return m, err
	}
	path := filepath.Join(dir, "manifest.json")
	if _, err = os.Stat(path); err == nil {
		old, e := LoadManifest(dir)
		if e != nil {
			return m, e
		}
		if digest(old.Config) != digest(m.Config) || digest(old.Sources) != digest(m.Sources) || digest(old.Cases) != digest(m.Cases) {
			return m, fmt.Errorf("run inputs changed; choose a new run directory")
		}
		return old, nil
	} else if !os.IsNotExist(err) {
		return m, err
	}
	err = WriteJSON(path, m)
	return m, err
}
func LoadManifest(dir string) (Manifest, error) {
	var m Manifest
	b, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return m, err
	}
	err = json.Unmarshal(b, &m)
	if err == nil && m.Protocol != Protocol {
		err = fmt.Errorf("unsupported run protocol: %s", m.Protocol)
	}
	return m, err
}
func generationJobs(m Manifest) []job {
	var jobs []job
	for _, c := range m.Cases {
		for rep := 0; rep < m.Config.Repetitions; rep++ {
			for _, variant := range []string{"baseline", "candidate"} {
				r := Request{Model: m.Config.Writer, Instructions: "Apply the following writing guide to the user's task. Follow the task's requested scope and output format.\n\n" + m.Sources[variant].Content, Input: c.Input}
				jobs = append(jobs, job{Record: Record{ID: fmt.Sprintf("%s/%d/%s", c.ID, rep, variant), CaseID: c.ID, Repetition: rep, Variant: variant, Request: r, Hash: digest(r)}, Case: c})
			}
		}
	}
	rand.New(rand.NewSource(m.Config.Seed)).Shuffle(len(jobs), func(i, j int) { jobs[i], jobs[j] = jobs[j], jobs[i] })
	return jobs
}
func readRecords(path string) (map[string]Record, error) {
	rows := map[string]Record{}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return rows, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	for line := 1; ; line++ {
		b, e := reader.ReadBytes('\n')
		if e == io.EOF && len(b) == 0 {
			break
		}
		if e != nil {
			return nil, fmt.Errorf("%s: incomplete journal line %d; inspect before resuming", path, line)
		}
		var r Record
		if e = json.Unmarshal(b, &r); e != nil {
			return nil, fmt.Errorf("%s line %d: %w", path, line, e)
		}
		if prior, ok := rows[r.ID]; ok && prior.Hash != r.Hash {
			return nil, fmt.Errorf("conflicting request hashes for %s", r.ID)
		}
		rows[r.ID] = r
	}
	return rows, nil
}
func judgeJobs(m Manifest, gen map[string]Record) ([]job, int) {
	var jobs []job
	skipped := 0
	for _, c := range m.Cases {
		for rep := 0; rep < m.Config.Repetitions; rep++ {
			base, okB := gen[fmt.Sprintf("%s/%d/baseline", c.ID, rep)]
			candidate, okC := gen[fmt.Sprintf("%s/%d/candidate", c.ID, rep)]
			if !okB || !okC || base.Error != "" || candidate.Error != "" || base.Response.Status != "completed" || candidate.Response.Status != "completed" {
				skipped++
				continue
			}
			key := fmt.Sprintf("%d/%s/%d", m.Config.Seed, c.ID, rep)
			flip := Hash([]byte(key))[0]%2 == 0
			for order := 0; order < 2; order++ {
				a, b := base, candidate
				variant := "baseline"
				if flip != (order == 1) {
					a, b = candidate, base
					variant = "candidate"
				}
				r := Request{Model: m.Config.Judge, Instructions: judgeInstructions(m), Input: judgeInput(c, a.Response.Text, b.Response.Text), Schema: JudgmentSchema()}
				jobs = append(jobs, job{Record: Record{ID: fmt.Sprintf("%s/%d/order-%d", c.ID, rep, order), CaseID: c.ID, Repetition: rep, Order: order, AVariant: variant, Request: r, Hash: digest(r)}, Case: c, A: a.Response.Text, B: b.Response.Text})
			}
		}
	}
	rand.New(rand.NewSource(m.Config.Seed+1)).Shuffle(len(jobs), func(i, j int) { jobs[i], jobs[j] = jobs[j], jobs[i] })
	return jobs, skipped
}
func execute(ctx context.Context, p Provider, j job, c Config, grading bool) Record {
	r := j.Record
	for attempt := 0; attempt <= c.Retries; attempt++ {
		start := time.Now().UTC()
		requestCtx, cancel := context.WithTimeout(ctx, time.Duration(c.TimeoutSeconds)*time.Second)
		response, err := p.Complete(requestCtx, r.Request)
		cancel()
		r.Response = response
		a := Attempt{Started: start, Milliseconds: time.Since(start).Milliseconds()}
		if err == nil && grading {
			r.Judgment, err = ParseJudgment(response.Text, j.Case, j.A, j.B)
		}
		if err != nil {
			a.Error = safeError(err)
			r.Error = a.Error
		} else {
			r.Error = ""
		}
		r.Attempts = append(r.Attempts, a)
		if err == nil || !retryable(err) || attempt == c.Retries || ctx.Err() != nil {
			break
		}
		select {
		case <-ctx.Done():
			return r
		case <-time.After(time.Duration(1<<attempt) * time.Second):
		}
	}
	return r
}
func Run(ctx context.Context, dir, stage string, p Provider, retryErrors bool, out io.Writer) error {
	m, err := LoadManifest(dir)
	if err != nil {
		return err
	}
	lock := filepath.Join(dir, ".lock")
	if err = os.Mkdir(lock, 0700); err != nil {
		return fmt.Errorf("run is locked (remove .lock only after confirming no process is active): %w", err)
	}
	defer os.Remove(lock)
	var jobs []job
	skipped := 0
	grading := stage == "judge"
	if stage == "generate" {
		jobs = generationJobs(m)
	} else if grading {
		gen, e := readRecords(filepath.Join(dir, "generations.jsonl"))
		if e != nil {
			return e
		}
		jobs, skipped = judgeJobs(m, gen)
	} else {
		return fmt.Errorf("unknown stage %s", stage)
	}
	name := "generations.jsonl"
	if grading {
		name = "judgments.jsonl"
	}
	path := filepath.Join(dir, name)
	existing, err := readRecords(path)
	if err != nil {
		return err
	}
	var todo []job
	failed := 0
	for _, j := range jobs {
		if old, ok := existing[j.Record.ID]; ok {
			if old.Hash != j.Record.Hash {
				return fmt.Errorf("request changed for %s; use a new run", old.ID)
			}
			if old.Error == "" {
				continue
			}
			if !retryErrors {
				failed++
				continue
			}
		}
		todo = append(todo, j)
	}
	fmt.Fprintf(out, "%s: %d pending, %d failed retained, %d pairs unavailable\n", stage, len(todo), failed, skipped)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	work := make(chan job)
	done := make(chan Record)
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	var wg sync.WaitGroup
	for n := 0; n < m.Config.Concurrency; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range work {
				if runCtx.Err() != nil {
					return
				}
				r := execute(runCtx, p, j, m.Config, grading)
				done <- r
			}
		}()
	}
	go func() {
		defer close(work)
		for _, j := range todo {
			select {
			case work <- j:
			case <-runCtx.Done():
				return
			}
		}
	}()
	go func() { wg.Wait(); close(done) }()
	var writeErr error
	for r := range done {
		if writeErr == nil {
			b, e := json.Marshal(r)
			if e == nil {
				_, e = f.Write(append(b, '\n'))
			}
			if e == nil {
				e = f.Sync()
			}
			if e != nil {
				writeErr = e
				cancel()
			}
		}
		if r.Error != "" {
			failed++
			fmt.Fprintf(out, "%s: %s\n", r.ID, r.Error)
		} else {
			fmt.Fprintf(out, "%s: completed (%d input, %d output tokens)\n", r.ID, r.Response.Usage.Input, r.Response.Usage.Output)
		}
	}
	if writeErr != nil {
		return writeErr
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if failed > 0 || skipped > 0 {
		return fmt.Errorf("%d requests failed; %d comparison pairs unavailable", failed, skipped)
	}
	return nil
}
func ReadAllRecords(path string) ([]Record, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var rows []Record
	d := json.NewDecoder(f)
	for {
		var r Record
		err = d.Decode(&r)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		rows = append(rows, r)
	}
	return rows, nil
}
func SortedRecords(rows map[string]Record) []Record {
	out := make([]Record, 0, len(rows))
	for _, r := range rows {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
