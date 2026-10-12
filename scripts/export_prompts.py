#!/usr/bin/env python3
"""Export JSONL writing tasks without grading criteria or reference answers."""
import argparse
import json
from pathlib import Path
import yaml

ROOT = Path(__file__).resolve().parents[1]


def prompt_records(cases, split):
    return [{'id': case['id'], 'task': case['task'], 'input': case['input']}
            for case in cases if case['split'] == split]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--split', choices=['development', 'held-out'], required=True)
    args = parser.parse_args()
    cases = yaml.safe_load((ROOT / 'evaluation/cases.yaml').read_text())
    for record in prompt_records(cases, args.split):
        print(json.dumps(record, ensure_ascii=False))


if __name__ == '__main__':
    main()
