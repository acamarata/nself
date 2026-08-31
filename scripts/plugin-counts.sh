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

# Six slugs are registered in BOTH registries, and they are not all the same
# case, so a plain free+pro sum is wrong.
#
#   search, webhooks            genuinely different free and paid
#                               implementations that happen to share a name.
#                               Two plugins. Count both.
#   cron, notify,               the same plugin listed twice while a rename
#   subtitle-manager, vpn       decision is pending. One plugin. Count once.
#
# This script is what the PPI, SPORT, the website and marketing copy all cite,
# so the naive sum published a number that was too high by the size of the
# second group. The web build had the identical bug independently.
overlap_dupes="$(python3 -c '
import json, sys
free = set(json.load(open(sys.argv[1]))["plugins"])
pro  = set(json.load(open(sys.argv[2]))["plugins"])
# Slugs shared by both registries that are ONE plugin, not two.
distinct_by_design = {"search", "webhooks"}
print(len((free & pro) - distinct_by_design))
' "$free_reg" "$pro_reg")"

printf 'free   %s\n' "$free_n"
printf 'pro    %s\n' "$pro_n"
printf 'total  %s\n' "$((free_n + pro_n - overlap_dupes))"
if [ "$overlap_dupes" -gt 0 ]; then
  printf '       (%s dual-registry duplicate(s) deducted; search/webhooks counted twice by design)\n' "$overlap_dupes"
fi

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
