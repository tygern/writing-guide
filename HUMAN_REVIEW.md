# Human review

Use the terminal to compare saved writing responses, record which wording you would keep, and choose the next guide experiment. Review and reporting run locally without an API key or model calls. The terminal workflow supports macOS and Linux.

## Start a session

Run these commands from the repository root. Supply one or more completed generation runs. Model grading is optional.

```sh
go run ./cmd/eval review --init \
  --session evaluation/results/human \
  --reviewer Tyson \
  --run evaluation/results/development \
  --run evaluation/results/long-form \
  --run evaluation/results/held-out

go run ./cmd/eval review
```

Replace the run paths with existing directories. For the first evaluation, those directories end in `-20261012`. The default session is `evaluation/results/human`. After initialization, omit `--run` when resuming. Use `--session` to keep another comparison or reviewer separate.

Initialization freezes the task inputs, guide snapshots, responses, reviewer label, selected repetitions, order, and A/B assignments. The program selects one random repetition per case and shuffles the comparisons and response sides. Those choices remain fixed on resume. It rejects overlapping case sets, different guide conditions, different writer settings, and incomplete generation runs. Omit the pilot when combining the development and held-out runs.

To examine every repetition, add `--all-repetitions` when creating a new session. Repetitions contribute to a verdict within each case; they do not count as independent case wins. Run separate sessions to compare different guide revisions. Multiple reviewers' sessions are not pooled automatically.

## Review the writing

The terminal presents the original task and two anonymous responses. It hides guide identities, case identifiers, reference answers, rule labels, and model verdicts. Identical responses are displayed once and still require your judgment. It preserves source spacing and lets the terminal wrap at its current width.

Choose the response you would keep, considering accuracy first, then clarity and your voice:

| Input | Action |
| --- | --- |
| `a` or `b` | Prefer that response overall. |
| `tie` or `t` | Both responses are equally acceptable. |
| `neither` or `n` | Neither response is acceptable. |
| `c` | Show the task's grading criteria. |
| `v` | Show published voice samples. |
| `s` | Skip; return to this comparison next sitting. |
| `u` | Undo the most recent saved judgment and review it again. |
| `q` | Save progress and quit. |

After choosing a response, optionally tag problems in either response and add a note or suggested correction. For example, enter `a:verbose b:meaning`, or `both:verbose` to tag both. Press Enter at both prompts to save just the preference. Enter `?` at the issue prompt to see tag definitions.

Available tags are `meaning`, `unsupported`, `completion`, `verbose`, `agency`, `vague`, `voice`, `structure`, `overedit`, and `other`. Tag the preferred response too if it has problems. A preference alone does not imply that the response is flawless. Use the note to capture a better sentence or explain a distinction the tags miss.

Each completed judgment is saved atomically with its history. The default sitting ends after ten judgments; use `--limit 5` for a shorter sitting or `--limit 0` for no limit. Quit and resume as often as needed:

```sh
go run ./cmd/eval review --limit 5
```

Ctrl-C preserves completed judgments. An unfinished preference, issue entry, or note is discarded. The OS releases the session lock when the process exits. If another review command has the session open, close it before resuming elsewhere.

To correct the last judgment after the sitting or entire session ends:

```sh
go run ./cmd/eval review --undo --limit 1
```

The undo is recorded immediately. If interrupted before replacing the judgment, that comparison remains pending.

## Interpret the results

```sh
go run ./cmd/eval human-report
```

The program produces three files in the session directory:

- `human-report.md`: preferences by case and task type, the recommendation, the decision policy, notes, and revealed A/B identities.
- `human-report.json`: the same counts, explicit issue findings, and machine-readable direction for an agent or another tool.
- `next-experiment.md`: development feedback, preferred passages, reviewer corrections, and proposed tests for the tagged problems.

These files are also refreshed when a review sitting ends normally. Keep them closed during blind review. `suite.json` contains the frozen inputs and identities; `reviews.json` contains the reviewer event history. Both are local records, not files to edit by hand. The results directory is ignored by Git.

The recommendation follows a fixed, versioned policy. Complete all development cases first. A strong preference requires at least three distinct case wins, at least two-thirds of decisive cases, no task category favoring the other guide, no mixed or neither case, and no meaning, unsupported-addition, or completion flag on the preferred guide. Advancing the candidate also requires completed held-out review with no observed regression under the policy. The report shows the exact conditions.

Possible directions include completing review, validating the candidate, advancing it, retaining the baseline, making a targeted revision, or recording no preference. This is a practical decision rule, not a significance test. Three wins among many ties can satisfy the rule, so inspect task coverage and the underlying passages. A tie does not establish equivalence. Missing issue tags do not establish the absence of defects.

For all-repetition sessions, a case's winner is the guide with more preference votes, ignoring acceptable ties. Equal opposing votes are mixed. Any combination containing neither is mixed unless every vote is neither. Partially reviewed cases stay out of preference counts. Explicit issue flags still apply, even when another repetition or response was preferred.

## Improve the guide

Read `next-experiment.md` after review. It ranks explicitly tagged development problems by distinct case count, includes the corresponding passages and notes, and suggests tests. It does not infer a broken rule merely because a case carries that rule's label.

Use the brief to make one focused candidate revision. Keep the current guide as baseline, use fresh output runs with the same writer settings, then create a new blind review session. Supply the brief to an agent along with the instruction to preserve the guide's priorities and effective wording. Human review does not rewrite, promote, publish, or call a model automatically.

Held-out passages and notes are excluded from the experiment brief. If held-out findings influence a revision, retire those cases from validation and collect fresh held-out cases. The first comparison's archived baseline already includes the improved subject–predicate rule; those results compare the complete guide versions, not the addition of that rule alone.
