package eval

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const Protocol = "writing-eval-v1"

type Case struct {
	ID       string   `yaml:"id" json:"id"`
	Split    string   `yaml:"split" json:"split"`
	Task     string   `yaml:"task" json:"task"`
	Input    string   `yaml:"input" json:"input"`
	Rules    []string `yaml:"rules" json:"rules"`
	Criteria []string `yaml:"criteria" json:"criteria"`
	Example  string   `yaml:"acceptable_example,omitempty" json:"acceptable_example,omitempty"`
}
type Model struct {
	Name            string `yaml:"name" json:"name"`
	Effort          string `yaml:"effort,omitempty" json:"effort,omitempty"`
	MaxOutputTokens int64  `yaml:"max_output_tokens" json:"max_output_tokens"`
}
type Config struct {
	Cases          string   `yaml:"cases" json:"cases"`
	Baseline       string   `yaml:"baseline" json:"baseline"`
	Candidate      string   `yaml:"candidate" json:"candidate"`
	Rubric         string   `yaml:"rubric" json:"rubric"`
	Samples        string   `yaml:"samples" json:"samples"`
	Split          string   `yaml:"split" json:"split"`
	CaseIDs        []string `yaml:"case_ids,omitempty" json:"case_ids,omitempty"`
	Repetitions    int      `yaml:"repetitions" json:"repetitions"`
	Concurrency    int      `yaml:"concurrency" json:"concurrency"`
	TimeoutSeconds int      `yaml:"timeout_seconds" json:"timeout_seconds"`
	Retries        int      `yaml:"retries" json:"retries"`
	Writer         Model    `yaml:"writer" json:"writer"`
	Judge          Model    `yaml:"judge" json:"judge"`
	Seed           int64    `yaml:"seed" json:"seed"`
}
type Snapshot struct {
	Path    string `json:"path"`
	SHA256  string `json:"sha256"`
	Content string `json:"content"`
}
type Manifest struct {
	Protocol string              `json:"protocol"`
	Created  time.Time           `json:"created"`
	Revision string              `json:"revision"`
	Dirty    bool                `json:"dirty"`
	Config   Config              `json:"config"`
	Cases    []Case              `json:"cases"`
	Sources  map[string]Snapshot `json:"sources"`
}
type Request struct {
	Model        Model          `json:"model"`
	Instructions string         `json:"instructions"`
	Input        string         `json:"input"`
	Schema       map[string]any `json:"schema,omitempty"`
}
type Usage struct {
	Input       int64 `json:"input"`
	CachedInput int64 `json:"cached_input"`
	Output      int64 `json:"output"`
}
type Response struct {
	ID     string `json:"id"`
	Model  string `json:"model"`
	Status string `json:"status"`
	Text   string `json:"text"`
	Usage  Usage  `json:"usage"`
}
type Provider interface {
	Complete(context.Context, Request) (Response, error)
}
type Attempt struct {
	Started      time.Time `json:"started"`
	Milliseconds int64     `json:"milliseconds"`
	Error        string    `json:"error,omitempty"`
}
type Record struct {
	ValidationVersion string    `json:"validation_version,omitempty"`
	ValidationNote    string    `json:"validation_note,omitempty"`
	Revalidated       bool      `json:"revalidated,omitempty"`
	ID                string    `json:"id"`
	CaseID            string    `json:"case_id"`
	Repetition        int       `json:"repetition"`
	Variant           string    `json:"variant,omitempty"`
	Order             int       `json:"order,omitempty"`
	AVariant          string    `json:"a_variant,omitempty"`
	Hash              string    `json:"request_hash"`
	Request           Request   `json:"request"`
	Response          Response  `json:"response"`
	Attempts          []Attempt `json:"attempts"`
	Error             string    `json:"error,omitempty"`
	Judgment          *Judgment `json:"judgment,omitempty"`
}

func Hash(data []byte) string { h := sha256.Sum256(data); return hex.EncodeToString(h[:]) }
func digest(v any) string     { b, _ := json.Marshal(v); return Hash(b) }
func decodeYAML(path string, value any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	d := yaml.NewDecoder(f)
	d.KnownFields(true)
	if err = d.Decode(value); err != nil {
		return err
	}
	var extra any
	if err = d.Decode(&extra); err != io.EOF {
		return fmt.Errorf("%s: expected one YAML document", path)
	}
	return nil
}
func LoadConfig(path string) (Config, error) {
	var c Config
	if err := decodeYAML(path, &c); err != nil {
		return c, err
	}
	if c.Split != "development" && c.Split != "held-out" {
		return c, fmt.Errorf("invalid split")
	}
	if c.Repetitions < 1 || c.Repetitions > 20 || c.Concurrency < 1 || c.Concurrency > 16 || c.TimeoutSeconds < 1 || c.Retries < 0 || c.Retries > 5 {
		return c, fmt.Errorf("invalid limits")
	}
	for _, m := range []Model{c.Writer, c.Judge} {
		if m.Name == "" || m.MaxOutputTokens < 1 {
			return c, fmt.Errorf("model name and positive max_output_tokens required")
		}
	}
	base := filepath.Dir(path)
	for _, s := range []*string{&c.Cases, &c.Baseline, &c.Candidate, &c.Rubric, &c.Samples} {
		if *s == "" {
			return c, fmt.Errorf("source paths required")
		}
		if !filepath.IsAbs(*s) {
			*s = filepath.Clean(filepath.Join(base, *s))
		}
	}
	return c, nil
}
func LoadCases(path string) ([]Case, error) {
	var cases []Case
	if err := decodeYAML(path, &cases); err != nil {
		return nil, err
	}
	ids := map[string]bool{}
	for _, c := range cases {
		if !regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`).MatchString(c.ID) || ids[c.ID] {
			return nil, fmt.Errorf("invalid or duplicate case ID: %s", c.ID)
		}
		ids[c.ID] = true
		if c.Split != "development" && c.Split != "held-out" {
			return nil, fmt.Errorf("invalid split in %s", c.ID)
		}
		if c.Task != "draft" && c.Task != "edit" && c.Task != "review" {
			return nil, fmt.Errorf("invalid task in %s", c.ID)
		}
		if strings.TrimSpace(c.Input) == "" || len(c.Criteria) == 0 || len(c.Rules) == 0 {
			return nil, fmt.Errorf("missing input, criteria, or rules in %s", c.ID)
		}
	}
	return cases, nil
}
func WriteJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".eval-*")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer os.Remove(name)
	if _, err = temp.Write(append(b, '\n')); err == nil {
		err = temp.Sync()
	}
	closeErr := temp.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(name, path)
}
