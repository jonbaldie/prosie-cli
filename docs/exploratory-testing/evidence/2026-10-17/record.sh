#!/bin/sh
# Append one command record. Truncate the scratch record first.
set -u
out="$EVIDENCE/current.txt"
: > "$out"
{
  printf 'COMMAND:'
  printf ' %s' "$@"
  printf '\n'
} >> "$out"
tmp_out="${TMPDIR}/prosie-out"
tmp_err="${TMPDIR}/prosie-err"
"$@" > "$tmp_out" 2> "$tmp_err"
code=$?
printf 'EXIT:%s\nSTDOUT:\n' "$code" >> "$out"
cat "$tmp_out" >> "$out"
printf 'STDERR:\n' >> "$out"
cat "$tmp_err" >> "$out"
printf '\n---\n' >> "$out"
cat "$out" >> "$EVIDENCE/transcript.txt"
exit "$code"
