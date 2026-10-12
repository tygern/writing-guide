# Evaluating guide revisions

Structural validation does not establish that a guide improves writing. This repository starts with evaluation material, not measured claims about effectiveness.

## Run a comparison

1. Select a baseline guide revision and a candidate revision. Use the same model, settings, task inputs, and relevant examples for both. Record which resources each condition includes.
2. Use the Go harness described in `README.md` to snapshot inputs and run generation separately from grading. It supplies only the chosen guide and task text to the writing model. The optional Python exporter remains available for other tools; its output excludes rule labels, criteria, and reference answers. Freeze the rubric before comparing guides.
3. Keep the grading criteria separate. If a model has already read the cases, start a fresh session for generation. Held-out cases are withheld from the writing prompt, not secret repository files. If repeatedly used to tune the guide, treat them as development cases and add fresh held-out cases.
4. Generate baseline and candidate outputs. Repeat selected cases to check whether an apparent improvement survives variation between runs.
5. Compare outputs without showing guide labels to the reviewer. Randomize order and permit ties. The reviewer receives the task, criteria, and both outputs. Human preference is the final authority on Tyson's voice. If using an LLM reviewer, check its judgments against human reviews.
6. Record the guide commits, model identifier, date, settings, complete prompts, outputs, individual judgments, and reasons in an untracked results directory. Do not publish task inputs or outputs containing private material without authorization.

## Assess separately

| Dimension | Question | Rating |
| --- | --- | --- |
| Meaning | Are facts, scope, uncertainty, and relevant conditions preserved? | Pass/fail; identify any change. |
| Clarity | Is the claim direct, specific, and understandable? | Prefer A, prefer B, or tie; explain. |
| Voice | Is the prose measured, concise, and consistent with the annotated samples? | Prefer A, prefer B, or tie; explain. |
| Editing restraint | Were effective passages preserved and edits limited to the task? | Pass/fail; identify unnecessary changes. |
| Rule compliance | Are applicable requirements satisfied? | Record rule-specific failures. |
| Task completion | Does the output provide the requested draft, edit, or review? | Pass/fail. |

Do not hide meaning failures inside an aggregate style score. Report counts and concrete examples alongside pairwise preferences. Accept different phrasings that meet the criteria; reference answers are illustrative, not exact-match targets. Keep acceptable originals in the set to detect unnecessary editing.

## Adding a case

Each case has a unique `id`, a `split` (`development` or `held-out`), a `task` (`draft`, `edit`, or `review`), a self-contained `input`, applicable `rules`, and observable `criteria`. An `acceptable_example` is optional. Include context necessary to interpret a claim. Avoid grading on facts that the writing model was never given.

## Harness interpretation

The initial comparison uses the archived Gist guide and the current standalone guide as complete instruction packages. The archive already includes the general subject-predicate rule added before the repository migration; this comparison measures the later consolidation and workflow changes, not that rule in isolation. Published examples embedded in the old guide remain part of that condition. The new guide contains its inline examples. Neither condition loads optional external references.

Review citations should identify an equivalent principle; exact identifiers from the new guide are not required. The same neutral rubric and unannotated published quotations are supplied for both conditions. Freeze these before a comparison. Reversing A/B order provides a second judgment; disagreement is reported as order sensitivity, not averaged into a win.

Flags are counted once per output if either judge order reports a failure, among pairs with both judgments complete. Rule breakdowns use case-level rule associations and may overlap. They do not prove that each associated rule was violated. Human reviewers should inspect the quoted evidence and the task context.

The supplemental `long-form.yaml` set contains three longer development tasks. Use it to check connected prose, but do not infer broad writing quality from a small sample. Repeated outputs from one case are correlated; report case counts and concrete failures rather than treating repetitions as independent tasks.

Run `go test -race ./...` and `go vet ./...` to check the harness. These offline checks validate isolation, resumability, response handling, blind reporting, and other execution behavior. Live results require model credentials and separate inspection.
