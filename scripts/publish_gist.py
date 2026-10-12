#!/usr/bin/env python3
"""Publish the canonical guide to its original Gist using authenticated gh."""
import argparse
import difflib
import json
from pathlib import Path
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]
GIST = 'ac9b9ac0d2dcc6be040d3f2f72698ede'


def read_gist():
    result = subprocess.run(['gh', 'api', f'gists/{GIST}'], check=True, capture_output=True, text=True)
    return json.loads(result.stdout)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--yes', action='store_true', help='Publish without prompting when already authorized.')
    parser.add_argument('--dry-run', action='store_true', help='Display the diff without publishing.')
    args = parser.parse_args()
    source = (ROOT / 'guide.md').read_text()
    gist = read_gist()
    if gist['owner']['login'] != 'tygern' or 'guide.md' not in gist['files']:
        parser.exit(1, 'Unexpected Gist owner or missing guide.md; no change made.\n')
    before = gist['files']['guide.md']['content']
    if before == source:
        print('The Gist already matches guide.md.')
        return
    print(''.join(difflib.unified_diff(before.splitlines(True), source.splitlines(True),
                                     fromfile='Gist guide.md', tofile='repository guide.md')))
    if args.dry_run:
        return
    if not args.yes and input('Publish this change to the public Gist? [y/N] ').strip().lower() != 'y':
        print('No change made.')
        return
    latest = read_gist()
    if latest['updated_at'] != gist['updated_at'] or latest['files']['guide.md']['content'] != before:
        parser.exit(1, 'The Gist changed during review. Run again to inspect the new diff.\n')
    # GitHub does not provide an atomic compare-and-swap here. Avoid concurrent edits.
    with tempfile.TemporaryDirectory(prefix='writing-guide-publish-') as directory:
        payload = Path(directory) / 'patch.json'
        payload.write_text(json.dumps({'files': {'guide.md': {'content': source}}}))
        subprocess.run(['gh', 'api', '--method', 'PATCH', f'gists/{GIST}', '--input', str(payload),
                        '--jq', '.html_url'], check=True)
    if read_gist()['files']['guide.md']['content'] != source:
        parser.exit(1, 'Saved content differs; inspect the Gist before retrying.\n')
    print('Verified the published guide.')


if __name__ == '__main__':
    main()
