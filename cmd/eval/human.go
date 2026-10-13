package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/tygern/writing-guide/internal/eval"
)

type runPaths []string

func (p *runPaths) String() string     { return fmt.Sprint([]string(*p)) }
func (p *runPaths) Set(s string) error { *p = append(*p, s); return nil }
func runHuman(stage string, args []string) error {
	flags := flag.NewFlagSet(stage, flag.ContinueOnError)
	session := flags.String("session", "evaluation/results/human", "review session directory")
	if stage == "human-report" {
		if err := flags.Parse(args); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return fmt.Errorf("unexpected positional arguments")
		}
		return eval.HumanReport(*session, os.Stdout)
	}
	var runs runPaths
	flags.Var(&runs, "run", "existing generation run (repeat to combine nonoverlapping case sets; new sessions only)")
	reviewer := flags.String("reviewer", "", "reviewer label (new sessions only)")
	all := flags.Bool("all-repetitions", false, "review every repetition instead of one randomly selected pair per case (new sessions only)")
	undo := flags.Bool("undo", false, "undo the most recent saved judgment and review it again")
	init := flags.Bool("init", false, "prepare session without recording judgments")
	limit := flags.Int("limit", 10, "maximum judgments this sitting; 0 = unlimited")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	return eval.HumanReview(eval.HumanOptions{Session: *session, Runs: runs, Reviewer: *reviewer, AllRepetitions: *all, Init: *init, Undo: *undo, Limit: *limit}, os.Stdin, os.Stdout)
}
