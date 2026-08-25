#!/usr/bin/env bash
# Print the authoritative plugin counts.
#
# Purpose: the free/pro/total counts appear in the PPI, in SPORT, on the
# website and in marketing copy, and every one of those had drifted from the
# registries by the time anyone checked (the PPI said 32 free / 124 pro when
# the registries held 49 / 127). The rule has always been that these numbers
# are generated rather than typed; this is the generator.
#
# Inputs: plugins/registry.json and plugins-pro/registry.json, resolved as
# siblings of this repo.
#
# Outputs: free, pro and total counts, plus the pro disk-vs-registry gap, which
# is expected and explained rather than reconciled.
#
# Constraints: reports what the registries say. It does not edit any document,
# because the documents that carry these numbers live in three repos.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
siblings="$(dirname "$root")"

count() { # count <registry.json>
  python3 -c '
import json, sys
d = json.load(open(sys.argv[1]))
p = d.get("plugins", {})
print(len(p))
' "$1"
}

free_reg="$siblings/plugins/registry.json"
pro_reg="$siblings/plugins-pro/registry.json"

missing=0
for f in "$free_reg" "$pro_reg"; do
  if [ ! -f "$f" ]; then
    echo "missing: $f" >&2
    missing=1
  fi
done
if [ "$missing" -ne 0 ]; then
  echo >&2
  echo "This reads the plugins and plugins-pro repos as siblings of $root." >&2
  echo "Clone them next to this repo, or run it from a full nself checkout." >&2
  exit 1
fi

free_n="$(count "$free_reg")"
pro_n="$(count "$pro_reg")"

printf 'free   %s\n' "$free_n"
printf 'pro    %s\n' "$pro_n"
printf 'total  %s\n' "$((free_n + pro_n))"

# The pro directory count is deliberately higher than the registry count: some
# directories under paid/ are shared code, not plugins. Print the gap with its
# members so nobody "reconciles" it by inventing entries.
paid_dir="$siblings/plugins-pro/paid"
if [ -d "$paid_dir" ]; then
  python3 -c '
import json, os, sys
reg = set(json.load(open(sys.argv[1])).get("plugins", {}))
dirs = {d for d in os.listdir(sys.argv[2]) if os.path.isdir(os.path.join(sys.argv[2], d))}
extra = sorted(dirs - reg)
print()
print("pro dirs on disk  %d" % len(dirs))
if extra:
    print("not plugins       %s" % ", ".join(extra))
gone = sorted(reg - dirs)
if gone:
    print("MISSING FROM DISK %s" % ", ".join(gone))
    sys.exit(1)
' "$pro_reg" "$paid_dir"
fi
