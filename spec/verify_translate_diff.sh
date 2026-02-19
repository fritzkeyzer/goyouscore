#!/usr/bin/env bash
# Verify that only comment lines (and embedded spec data) changed in the git diff.
# Lists any changed lines that don't start with // or " (after trimming whitespace).
# Exits with error if any unexpected changes are found.

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

# Get added/removed lines from diff, strip the +/- prefix, trim leading whitespace.
# Exclude diff headers (+++/---), empty lines, comment lines (//), and quoted strings (embedded spec).
unexpected=$(
  git diff --unified=0 -- client.gen.go \
    | grep '^[+-]' \
    | grep -v '^[+-][+-][+-]' \
    | sed 's/^[+-]//' \
    | sed 's/^[[:space:]]*//' \
    | grep -v '^$' \
    | grep -v '^//' \
    | grep -v '^"' \
  || true
)

if [ -n "$unexpected" ]; then
  count=$(echo "$unexpected" | wc -l | tr -d ' ')
  echo "ERROR: Found $count unexpected non-comment changed line(s):"
  echo ""
  echo "$unexpected"
  exit 1
else
  echo "OK: All changed lines are comments or embedded spec data."
fi
