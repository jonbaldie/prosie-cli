#!/bin/bash
# Run prosie with isolated config; log command, stdout, stderr, exit code.
# Usage: r.sh <transcript> [--stdin FILE] args...
T=$1; shift
IN=/dev/null
if [ "$1" = "--stdin" ]; then IN=$2; shift 2; fi
export PROSIE_API_URL=http://127.0.0.1:$(cat $TMPDIR/run/port)
export PROSIE_CONFIG_PATH=$TMPDIR/run/config.json
export PROSIE_API_TOKEN=${PROSIE_API_TOKEN-test-token}
OUT=$(mktemp); ERR=$(mktemp)
"${PROSIE_BIN:-./build/prosie}" "$@" <"$IN" >"$OUT" 2>"$ERR"; RC=$?
{ printf '$ prosie'; printf ' %q' "$@"; [ "$IN" != /dev/null ] && printf ' < %s' "$(basename "$IN")"; echo
  echo "--- stdout"; cat "$OUT"; echo "--- stderr"; cat "$ERR"; echo "--- exit $RC"; echo; } | tee -a "$T"
rm -f "$OUT" "$ERR"
