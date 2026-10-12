# Tyson Gern's writing guide

Instructions, examples, and evaluation cases for drafting, editing, and reviewing prose in my voice.

## Use the guide

Read [guide.md](guide.md) for the complete instructions. It is self-contained and is the canonical source for the published copies and skill package.

For a web-capable agent, use this prompt with the task:

> Read the complete writing guide at https://raw.githubusercontent.com/tygern/writing-guide/main/guide.md before starting. Apply its instructions for the requested task. If the guide cannot be read in full, report the access problem and work with me to resolve it.

The [rendered GitHub page](https://github.com/tygern/writing-guide/blob/main/guide.md) is an alternative when raw URLs are inaccessible. An agent must fetch the content; a link alone does not supply the instructions. For a reproducible run, replace `main` in the raw URL with a commit SHA. An agent without network access can use an attached or local copy.

### Agent skill

The ready-to-install package is [dist/tyson-writing](dist/tyson-writing). Copy that entire directory into the skill location supported by the agent. It includes the guide and supporting examples; it has no runtime dependencies or network requirement.

Invoke it explicitly when consistent application matters:

> Use the tyson-writing skill to edit this passage. Preserve its meaning and return the revised prose.

The root `skill/SKILL.md` is the package source, not a standalone install. Its references are populated during the build. Root `AGENTS.md` contains instructions for maintaining this repository.

## Contents

| Path | Purpose |
| --- | --- |
| `guide.md` | Priorities, rules, examples, task modes, and final check. |
| `examples/sentence-edits.md` | Corrections and acceptable wording that should be preserved. |
| `examples/annotated-passages.md` | Original published excerpts with notes on voice and current exceptions. |
| `evaluation/cases.yaml` | Development and held-out tasks with grading criteria. |
| `evaluation/rubric.md` | Procedure for comparing guide revisions. |
| `skill/SKILL.md` | Agent skill metadata and entry instructions. |
| `dist/` | Generated standalone guide and self-contained skill package. |

## Develop and validate

Use Python 3.9 or later. Building uses only the standard library; checks and prompt export use PyYAML.

```sh
python3 -m venv .venv
.venv/bin/python -m pip install -r requirements-dev.txt
.venv/bin/python scripts/build.py
.venv/bin/python scripts/check.py
.venv/bin/python scripts/export_prompts.py --split development
```

Edit the source files, then rebuild and commit the generated files with them. Checks verify rule references, evaluation structure, local links, quote preservation, and generated artifacts. GitHub Actions runs those checks for pushes and pull requests.

These checks do not measure writing quality. Follow [the evaluation procedure](evaluation/rubric.md) to compare model outputs. The cases and automated judgments are evaluation tools. A passing structural check or a model preference alone does not establish improved writing.

## Run writing evaluations with Go

The Go harness generates writing responses, grades anonymous pairs in both orders, and creates a report plus a blind human-review page. Ordinary tests run offline. Live commands use the OpenAI Responses API and require `OPENAI_API_KEY` in the process environment. Do not put credentials in configuration files or commits.

Use Go 1.27.2 or later for the pinned SDK. The model names and reasoning settings in `evaluation/pilot.yaml` and `evaluation/run.yaml` are explicit defaults; choose models available to the account before starting a run. Both guide conditions use the same writer and settings. The default is `gpt-6-astra`, with low writer effort and medium judge effort.

```sh
go test -race ./...
go vet ./...

# Freeze inputs and inspect workload without calling a model.
go run ./cmd/eval plan --config evaluation/pilot.yaml --run evaluation/results/pilot

# Pilot: 10 writing responses and 10 judgments, before any retries.
go run ./cmd/eval generate --run evaluation/results/pilot
go run ./cmd/eval judge --run evaluation/results/pilot
go run ./cmd/eval report --run evaluation/results/pilot
```

Inspect `report.md` and use `review.html` for anonymous human preferences. Export reviews as JSON; the page also attempts to save selections in browser-local storage. `review-map.json` maps anonymous A/B outputs to guide conditions. Keep it hidden until preferences are recorded. The HTML page uses no network resources.

After checking the pilot prompts, errors, and judgments, start the development comparison:

```sh
go run ./cmd/eval generate --config evaluation/run.yaml --run evaluation/results/development
go run ./cmd/eval judge --run evaluation/results/development
go run ./cmd/eval report --run evaluation/results/development
```

This requests 90 writing responses and 90 judgments across 15 cases, two guides, and three repetitions. `evaluation/long-form-run.yaml` adds a separate comparison of three longer development tasks (18 writing responses and 18 judgments). For the 10 held-out cases, use `evaluation/held-out-run.yaml` and a new run directory after fixing the evaluation procedure (60 writing responses and 60 judgments).

Run directories are ignored by Git. Each manifest snapshots the guide text, selected cases, grading rubric, published voice samples, model settings, source hashes, and repository revision. The baseline archive's historical wrapper is removed. Only task text and the chosen guide reach the writing model. The judge sees the task, criteria, published quotations, and anonymous outputs; it does not see guide identities or reference answers.

Resume by repeating a command with `--run`. Completed responses are retained. Recorded execution errors remain visible unless `--retry-errors` is explicitly supplied. For saved complete judgments rejected by an older validator, `go run ./cmd/eval recheck --run <directory>` revalidates them locally without model calls. It appends a validation event and does not double-count usage. Evidence matching allows removal of paired Markdown bold markers but rejects changed words; normalized matches are labeled in the report. Retryable transport errors receive bounded backoff; bad writing and incomplete responses are not automatically resampled. Changed inputs require a new run directory. Each run has a process lock; after a crash, remove `.lock` only after confirming that no process is active. If a process dies after a request completes but before its result is recorded, resuming can repeat that request. The harness does not guarantee exactly-once billing.

The report separates meaning, restraint, completion, criterion flags, clarity, voice, and order-sensitive judgments. It records tokens rather than estimating prices; failed network attempts may have usage unavailable to the harness. Calibrate model judgments against human reviews before using them to claim improved writing. API execution failures and missing results are reported separately.

## Publish the Gist mirror

The project began as [a Gist](https://gist.github.com/tygern/ac9b9ac0d2dcc6be040d3f2f72698ede). The repository is now the editing source. To replace the Gist's `guide.md` with the current guide, explicitly run:

```sh
python3 scripts/publish_gist.py
```

This requires an authenticated GitHub CLI with Gist write access. The command displays the proposed diff and asks for confirmation before publishing. `--yes` skips the prompt for an already-authorized invocation. It preserves other Gist files and verifies the saved content. Publication is manual; pushing this repository does not update the Gist.

## Provenance

The initial source is archived in [examples/original-guide.md](examples/original-guide.md), with its Gist revision recorded at the top. It preserves the original published quotations and the reference principles from Vonnegut, Orwell, and Strunk and White. Current instructions live in `guide.md`; the archive is historical material, not additional rules.
