#!/usr/bin/env python3
"""Validate sources, evaluation structure, quotes, links, and generated artifacts."""
import re
import subprocess
import sys
from pathlib import Path
import yaml
from export_prompts import prompt_records

ROOT = Path(__file__).resolve().parents[1]


def require(condition, message):
    if not condition:
        raise ValueError(message)


def main():
    guide = (ROOT / 'guide.md').read_text()
    rules = set(re.findall(r'^### ([a-z]+(?:-[a-z]+)+)$', guide, re.M))
    require(bool(rules), 'No rule identifiers found.')
    cases = yaml.safe_load((ROOT / 'evaluation/cases.yaml').read_text())
    require(isinstance(cases, list) and cases, 'Cases must be a nonempty YAML list.')
    ids = set()
    for case in cases:
        require(isinstance(case, dict), 'Each case must be a mapping.')
        required = {'id', 'split', 'task', 'input', 'rules', 'criteria'}
        require(required <= case.keys(), f'Case missing fields: {case.get("id")}')
        require(not (case.keys() - required - {'acceptable_example'}), 'Unknown case field.')
        name = case['id']
        require(isinstance(name, str) and re.fullmatch(r'[a-z0-9]+(?:-[a-z0-9]+)*', name), 'Invalid case ID.')
        require(name not in ids, f'Duplicate case: {name}')
        ids.add(name)
        require(case['split'] in {'development', 'held-out'}, f'Invalid split: {name}')
        require(case['task'] in {'draft', 'edit', 'review'}, f'Invalid task: {name}')
        require(isinstance(case['input'], str) and case['input'].strip(), f'Empty input: {name}')
        for field in ('rules', 'criteria'):
            require(isinstance(case[field], list) and case[field]
                    and all(isinstance(x, str) and x.strip() for x in case[field]), f'Invalid {field}: {name}')
        require(set(case['rules']) <= rules, f'Unknown rule in {name}: {set(case["rules"]) - rules}')
        if 'acceptable_example' in case:
            require(isinstance(case['acceptable_example'], str) and case['acceptable_example'].strip(),
                    f'Empty reference: {name}')
    for split in ('development', 'held-out'):
        exported = prompt_records(cases, split)
        require(exported, f'No {split} cases.')
        require(all(set(row) == {'id', 'task', 'input'} for row in exported), 'Grading material leaked into export.')
        require({row['task'] for row in exported} == {'draft', 'edit', 'review'}, f'Missing task mode in {split}.')
    # Compare original quoted lines, including paragraph separators, in order.
    archive = (ROOT / 'examples/original-guide.md').read_text()
    source = archive.split('## Examples From My Published Writing\n', 1)[1].split('## Reference Principles', 1)[0]
    quoted = lambda text: [line for line in text.splitlines() if line.startswith('>')]
    require(quoted(source) == quoted((ROOT / 'examples/annotated-passages.md').read_text()),
            'A published quotation was changed or omitted.')
    skill = (ROOT / 'skill/SKILL.md').read_text()
    require(skill.startswith('---\n'), 'Missing skill frontmatter.')
    metadata = yaml.safe_load(skill.split('---', 2)[1])
    require(metadata.get('name') == 'tyson-writing', 'Skill name must match package directory.')
    require(isinstance(metadata.get('description'), str) and metadata['description'].strip(), 'Missing description.')
    # The template's reference links resolve in its generated package, checked below.
    for file in ROOT.rglob('*.md'):
        if any(part in {'.git', '.venv'} for part in file.relative_to(ROOT).parts):
            continue
        if file == ROOT / 'skill/SKILL.md':
            continue
        for link in re.findall(r'\[[^\]]*\]\(([^)]+)\)', file.read_text()):
            if '://' in link or link.startswith(('#', 'mailto:')):
                continue
            target = link.split('#', 1)[0]
            require((file.parent / target).exists(), f'Broken link in {file.relative_to(ROOT)}: {link}')
    subprocess.run([sys.executable, str(ROOT / 'scripts/build.py'), '--check'], check=True)
    print(f'Validated {len(rules)} rules, {len(cases)} cases, published quotations, links, and skill metadata.')
    print('Structural checks only; no model quality comparison was run.')


if __name__ == '__main__':
    try:
        main()
    except (ValueError, KeyError, TypeError, yaml.YAMLError) as error:
        sys.exit(str(error))
