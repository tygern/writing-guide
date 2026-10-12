package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/tygern/writing-guide/internal/eval"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: eval plan|generate|judge|recheck|report [options]")
	}
	stage := os.Args[1]
	flags := flag.NewFlagSet(stage, flag.ContinueOnError)
	config := flags.String("config", "", "YAML configuration (required for plan or a new generation run)")
	dir := flags.String("run", "", "run directory")
	retry := flags.Bool("retry-errors", false, "retry recorded execution errors; never resample completed outputs")
	if err := flags.Parse(os.Args[2:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	if *dir == "" {
		return fmt.Errorf("--run is required")
	}
	switch stage {
	case "plan", "generate":
		var m eval.Manifest
		var err error
		if *config != "" {
			m, err = eval.Initialize(*config, *dir)
		} else {
			m, err = eval.LoadManifest(*dir)
		}
		if err != nil {
			return err
		}
		fmt.Printf("%d cases x %d repetitions x 2 guides = %d writing requests; %d judge requests (both orders). Up to %d attempts per request.\n", len(m.Cases), m.Config.Repetitions, len(m.Cases)*m.Config.Repetitions*2, len(m.Cases)*m.Config.Repetitions*2, m.Config.Retries+1)
		if stage == "plan" {
			return nil
		}
	case "judge", "report", "recheck":
		if *config != "" {
			return fmt.Errorf("%s uses the frozen run manifest; omit --config", stage)
		}
	default:
		return fmt.Errorf("unknown command: %s", stage)
	}
	if stage == "recheck" {
		return eval.Recheck(*dir, os.Stdout)
	}
	if stage == "report" {
		return eval.Report(*dir)
	}
	p, err := eval.NewOpenAI()
	if err != nil {
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	return eval.Run(ctx, *dir, stage, p, *retry, os.Stdout)
}
