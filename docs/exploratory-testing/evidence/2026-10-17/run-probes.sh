#!/bin/sh
set -u
ROOT="/Users/jonathanbaldie/Code-2/github.com/jonbaldie/prosie-cli"
EVIDENCE="$ROOT/docs/exploratory-testing/evidence/2026-10-17"
PROSIE="$ROOT/build/prosie"
WORK="$TMPDIR/prosie-explore"
URL=$(cat "$WORK/url")
export EVIDENCE TMPDIR PROSIE_API_URL="$URL" PROSIE_API_TOKEN="test-token"
export PROSIE_CONFIG_PATH="$WORK/config-probes.json"
rm -f "$PROSIE_CONFIG_PATH"
unset PROSIE_OPENROUTER_API_KEY PROSIE_OPENAI_API_KEY PROSIE_ANTHROPIC_API_KEY
: > "$EVIDENCE/probes.txt"
cp "$EVIDENCE/transcript.txt" "$EVIDENCE/j2-j3.txt"
: > "$EVIDENCE/transcript.txt"
rec() { sh "$EVIDENCE/record.sh" "$@"; }

curl -s -X POST "$URL/_fixture/reset" >/dev/null
curl -s -X POST "$URL/_fixture/mode" -H 'Content-Type: application/json' -d '{"stream":"normal","fail_message":false,"delay":0}' >/dev/null

rec "$PROSIE" book create --title "Negative Target" --target-words -5 --json
rec "$PROSIE" book create --title "Zero Target" --target-words 0 --json
rec "$PROSIE" book update 101 --target-words 0
rec "$PROSIE" book update 101 --target-words -1
rec "$PROSIE" book update 101 --premise ""
rec "$PROSIE" book show 101 --json
rec "$PROSIE" chapter update 102 --content '<p>The door was shut.</p><p>The hall was cold.</p>'
rec "$PROSIE" chapter show 102
rec "$PROSIE" chapter export 102
rec "$PROSIE" chapter show 102 --json
printf '  keep spaces  \n' | rec "$PROSIE" generate rewrite 102 --selection-file - --prompt "Keep spaces."
rec "$PROSIE" chat send 101 "Why is the door shut?" --title "Door"
rec "$PROSIE" chat show 103
rec "$PROSIE" chat export 103 --json
rec "$PROSIE" chat export 103 -o "$WORK/chat.json"
rec "$PROSIE" chat import 101 "$WORK/chat.json" --json
rec "$PROSIE" chat send 101 -- --why
curl -s -X POST "$URL/_fixture/mode" -H 'Content-Type: application/json' -d '{"fail_message":true}' >/dev/null
rec "$PROSIE" chat send 101 "This will fail" --title "Failed turn"
curl -s -X POST "$URL/_fixture/mode" -H 'Content-Type: application/json' -d '{"fail_message":false}' >/dev/null
rec "$PROSIE" chat send --conversation 105 "Try again"
rec "$PROSIE" llm update
rec "$PROSIE" book export 101 --format DOCX -o "$WORK/upper.docx"
rec "$PROSIE" book create --title $'Line\nTwo' --json
echo PROBES_DONE
