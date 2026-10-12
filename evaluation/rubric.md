# Evaluating guide revisions

Structural validation does not establish that a guide improves writing. This repository starts with evaluation material, not measured claims about effectiveness.

## Run a comparison

1. Select a baseline guide revision and a candidate revision. Use the same model, settings, task inputs, and relevant examples for both. Record which resources each condition includes.
2. Use `python3 scripts/export_prompts.py --split development` to prepare development inputs. For a final comparison, use `--split held-out`. The export excludes rule labels, criteria, and reference answers. Give the writing model only the guide, the exported task, and the resources allowed for both conditions.
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

Start with manual comparison. Add model execution automation when the chosen provider, budget, and evaluation routine are established.
