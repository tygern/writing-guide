# Maintaining the writing guide

This repository maintains Tyson Gern's writing instructions, examples, and evaluation material.

- Read `guide.md` before changing authored prose. It is the canonical guide.
- Preserve the exact wording of published quotations. Explain conflicts with current rules in annotations.
- Use stable rule identifiers. Update references when changing an identifier.
- For a reported writing failure, add a case that captures the general issue and an acceptable neighboring case when useful. Avoid accumulating special prohibitions for individual sentences.
- Keep development examples separate from held-out evaluation inputs. Do not load held-out cases or grading criteria into the prompt being evaluated.
- Do not edit `dist/` directly. Run `python3 scripts/build.py` after source changes.
- Install development requirements, then run `python3 scripts/check.py`. This checks structure and packaging, not writing quality.
- For substantive guide revisions, compare outputs using `evaluation/rubric.md` when model access is available and authorized. Report when behavioral evaluation has not been run. Do not claim an improvement from structural checks alone.
- The skill source in `skill/SKILL.md` is a packaging template. Its `references/` links resolve in the generated `dist/tyson-writing/` package.
- Publish only the requested destinations. `scripts/publish_gist.py` updates the existing public Gist when explicitly invoked for that purpose.

- For Go harness changes, run `go test -race ./...` and `go vet ./...`. Tests must not make live model calls.
- Keep credentials in the process environment. Do not commit `evaluation/results/` or publish raw evaluation material without authorization.
- For live comparisons, run the pilot first and inspect prompts and results before scaling up. Preserve failed responses; do not resample merely because the writing is poor.
