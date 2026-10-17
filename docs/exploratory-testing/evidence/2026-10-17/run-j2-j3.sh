#!/bin/sh
set -u
ROOT="/Users/jonathanbaldie/Code-2/github.com/jonbaldie/prosie-cli"
EVIDENCE="$ROOT/docs/exploratory-testing/evidence/2026-10-17"
PROSIE="$ROOT/build/prosie"
WORK="$TMPDIR/prosie-explore"
URL=$(cat "$WORK/url")
export EVIDENCE TMPDIR PROSIE_API_URL="$URL" PROSIE_API_TOKEN="test-token"
export PROSIE_CONFIG_PATH="$WORK/config.json"
: > "$EVIDENCE/transcript.txt"
rec() { sh "$EVIDENCE/record.sh" "$@"; }

curl -s -X POST "$URL/_fixture/reset" >/dev/null
curl -s -X POST "$URL/_fixture/mode" -H 'Content-Type: application/json' -d '{"stream":"normal","fail_message":false,"delay":0}' >/dev/null

# Journey 2: generate prose and recover from a conflict.
rec "$PROSIE" book create --title "The Cold Hall" --json
rec "$PROSIE" chapter update 102 --content "The door was shut and the hall was cold."
rec "$PROSIE" generate continue 102 --no-stream --instruction "Write the closing scene." --words 40 --lines 2
rec "$PROSIE" chapter show 102 --json
rec "$PROSIE" generate continue 102 --no-persist --no-stream
rec "$PROSIE" chapter show 102 --json
rec "$PROSIE" generate continue 102 --json --words 0
rec "$PROSIE" generate continue 102 --lines -1
printf 'The door was shut and the hall was cold.' | rec "$PROSIE" generate rewrite 102 --selection-file - --prompt "Make it tense." --no-stream
rec "$PROSIE" generate rewrite 102 --selection "The door was shut and the hall was cold." --action tighten --persist
rec "$PROSIE" chapter show 102 --json
rec "$PROSIE" generate undo 102
rec "$PROSIE" generate undo 102
rec "$PROSIE" generate summarize 102
rec "$PROSIE" chapter show 102 --json
rec "$PROSIE" generate summarize 102 --no-persist --json
curl -s -X POST "$URL/_fixture/mode" -H 'Content-Type: application/json' -d '{"stream":"conflict"}' >/dev/null
rec "$PROSIE" generate continue 102 --no-stream
rec "$PROSIE" generate continue 102 --json
rec "$PROSIE" generate continue 102
rec "$PROSIE" generate rewrite 102 --selection "cold." --prompt "Sharpen." --persist --stream
rec "$PROSIE" generate summarize 102
rec "$PROSIE" chapter show 102 --json
curl -s -X POST "$URL/_fixture/mode" -H 'Content-Type: application/json' -d '{"stream":"truncate"}' >/dev/null
rec "$PROSIE" generate continue 102
rec "$PROSIE" generate rewrite 102 --selection "cold." --prompt "Sharpen." --stream
rec "$PROSIE" chat stream 101 "Will she leave?"
curl -s -X POST "$URL/_fixture/mode" -H 'Content-Type: application/json' -d '{"stream":"error"}' >/dev/null
rec "$PROSIE" generate continue 102
curl -s -X POST "$URL/_fixture/mode" -H 'Content-Type: application/json' -d '{"stream":"normal"}' >/dev/null
rec "$PROSIE" generate continue 102
rec "$PROSIE" chapter show 102
rec "$PROSIE" generate reject 102
rec "$PROSIE" chapter show 102 --json

# Journey 3: sign in, ask the book, change the model.
curl -s -X POST "$URL/_fixture/reset" >/dev/null
unset PROSIE_API_TOKEN
rm -f "$PROSIE_CONFIG_PATH"
rec "$PROSIE" auth status
rec "$PROSIE" auth status --json
rec "$PROSIE" book list
rec "$PROSIE" auth login --help
rec "$PROSIE" auth login --token test-token
rec "$PROSIE" auth status
export PROSIE_API_TOKEN="test-token"
rec "$PROSIE" auth logout
rec "$PROSIE" auth status
unset PROSIE_API_TOKEN
rec "$PROSIE" auth status
rec "$PROSIE" auth login --token test-token --json
printf '{not json' > "$PROSIE_CONFIG_PATH"
export PROSIE_API_TOKEN="test-token"
rec "$PROSIE" book list
rm -f "$PROSIE_CONFIG_PATH"
rec "$PROSIE" auth login --no-browser --json
export PROSIE_API_TOKEN="test-token"
rec "$PROSIE" book create --title "The Cold Hall" --json
rec "$PROSIE" chat send 101 "Why is the door shut?" --title "Door"
rec "$PROSIE" chat show 102
rec "$PROSIE" chat list 101 --json
rec "$PROSIE" chat stream --conversation 102 "What happens next?"
rec "$PROSIE" chat stream 101 "A second thread" --json --title "Other"
rec "$PROSIE" chat export 102 -o "$WORK/chat.json"
rec "$PROSIE" chat export 102 --json
rec "$PROSIE" chat import 101 "$WORK/chat.json"
printf '%s\n' '{"title":"Piped chat","messages":[{"role":"user","content":"From stdin"}]}' | rec "$PROSIE" chat import 101 -
printf 'not-json' | rec "$PROSIE" chat import 101 -
rec "$PROSIE" chat send 101
rec "$PROSIE" chat send 101 --why
printf 'n\n' | rec "$PROSIE" chat delete 102 --json
rec "$PROSIE" chat show 102 --json
curl -s -X POST "$URL/_fixture/mode" -H 'Content-Type: application/json' -d '{"fail_message":true}' >/dev/null
rec "$PROSIE" chat send 101 "This will fail" --title "Failed turn"
curl -s -X POST "$URL/_fixture/mode" -H 'Content-Type: application/json' -d '{"fail_message":false}' >/dev/null
rec "$PROSIE" llm show
rec "$PROSIE" llm models --json
PROSIE_OPENROUTER_API_KEY="secret-router-key" rec "$PROSIE" llm update --provider openrouter --model openai/gpt-test --json
rec "$PROSIE" llm show
rec "$PROSIE" llm update
rec "$PROSIE" chapter update 103 --content '<p>The door was shut.</p><p>The hall was cold.</p>'
rec "$PROSIE" chapter show 103
rec "$PROSIE" chapter export 103
echo DONE
