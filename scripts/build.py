#!/usr/bin/env python3
"""Build or verify published guide copies using only the Python standard library."""
import argparse
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def artifacts():
    sources = {
        'dist/guide.md': 'guide.md',
        'dist/tyson-writing/SKILL.md': 'skill/SKILL.md',
        'dist/tyson-writing/references/guide.md': 'guide.md',
        'dist/tyson-writing/references/sentence-edits.md': 'examples/sentence-edits.md',
        'dist/tyson-writing/references/annotated-passages.md': 'examples/annotated-passages.md',
    }
    return {target: (ROOT / source).read_bytes() for target, source in sources.items()}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--check', action='store_true', help='Fail if committed artifacts differ from sources.')
    args = parser.parse_args()
    expected = artifacts()
    if args.check:
        stale = [name for name, data in expected.items()
                 if not (ROOT / name).is_file() or (ROOT / name).read_bytes() != data]
        extra = sorted(str(path.relative_to(ROOT)) for path in (ROOT / 'dist').rglob('*')
                       if path.is_file() and str(path.relative_to(ROOT)) not in expected)
        if stale or extra:
            parser.exit(1, 'Generated files differ. Run python3 scripts/build.py.\n'
                        + '\n'.join(stale + extra) + '\n')
        print('Generated guide and skill match their sources.')
        return
    for name, data in expected.items():
        target = ROOT / name
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_bytes(data)
    print(f'Built {len(expected)} files. Run scripts/check.py to validate.')


if __name__ == '__main__':
    main()
