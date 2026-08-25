#!/usr/bin/env bash
# Purpose: catch workflow YAML that GitHub will reject at dispatch time.
#          A duplicate mapping key (two `if:` on one step) is valid enough for
#          most YAML parsers — they silently keep the last one — but GitHub
#          refuses the file with only "this run likely failed because of a
#          workflow file issue", which costs a push cycle to diagnose.
# Inputs:  none. Lints every file in .github/workflows/.
# Outputs: exit 0 when all files parse strictly; exit 1 naming the offenders.
# Constraints: python3 with PyYAML, which CI already has.
set -euo pipefail

cd "$(dirname "$0")/../.."

python3 - <<'PYEOF'
import glob
import sys

import yaml


class StrictLoader(yaml.SafeLoader):
    """SafeLoader that refuses duplicate mapping keys."""


def no_duplicates(loader, node, deep=False):
    mapping = {}
    for key_node, value_node in node.value:
        key = loader.construct_object(key_node, deep=deep)
        if key in mapping:
            raise yaml.constructor.ConstructorError(
                "while constructing a mapping",
                node.start_mark,
                f"duplicate key {key!r}",
                key_node.start_mark,
            )
        mapping[key] = loader.construct_object(value_node, deep=deep)
    return mapping


StrictLoader.add_constructor(
    yaml.resolver.BaseResolver.DEFAULT_MAPPING_TAG, no_duplicates
)

failures = []
files = sorted(glob.glob(".github/workflows/*.yml") + glob.glob(".github/workflows/*.yaml"))
if not files:
    print("workflow-lint: no workflow files found", file=sys.stderr)
    sys.exit(1)

for path in files:
    try:
        with open(path, encoding="utf-8") as fh:
            yaml.load(fh, Loader=StrictLoader)
    except yaml.YAMLError as exc:
        failures.append(f"{path}: {exc}")

if failures:
    print("::error::Workflow YAML is invalid:", file=sys.stderr)
    for f in failures:
        print(f"  {f}", file=sys.stderr)
    sys.exit(1)

print(f"workflow-lint passed: {len(files)} workflow files parse with no duplicate keys.")
PYEOF
