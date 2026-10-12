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

These checks do not measure writing quality. Follow [the evaluation procedure](evaluation/rubric.md) to compare model outputs. The initial version has structural validation and illustrative cases; no model comparison has established that it improves writing.

## Publish the Gist mirror

The project began as [a Gist](https://gist.github.com/tygern/ac9b9ac0d2dcc6be040d3f2f72698ede). The repository is now the editing source. To replace the Gist's `guide.md` with the current guide, explicitly run:

```sh
python3 scripts/publish_gist.py
```

This requires an authenticated GitHub CLI with Gist write access. The command displays the proposed diff and asks for confirmation before publishing. `--yes` skips the prompt for an already-authorized invocation. It preserves other Gist files and verifies the saved content. Publication is manual; pushing this repository does not update the Gist.

## Provenance

The initial source is archived in [examples/original-guide.md](examples/original-guide.md), with its Gist revision recorded at the top. It preserves the original published quotations and the reference principles from Vonnegut, Orwell, and Strunk and White. Current instructions live in `guide.md`; the archive is historical material, not additional rules.
