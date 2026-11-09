---------------------------- MODULE Wizard ----------------------------
EXTENDS Naturals, FiniteSets
CONSTANT CopiesWork
VARIABLES phase, saved, input, value, browser, secret, stage, run,
          defaults, answers, values, written, secrets, skipped, variable, failure,
          eof, key, tmp, tempExists
vars == <<phase, saved, input, value, browser, secret, stage, run,
          defaults, answers, values, written, secrets, skipped, variable, failure,
          eof, key, tmp, tempExists>>
Keys == {"public", "private"}
Default(x) == IF x = "absent" THEN "empty" ELSE x
Init == /\ phase = "start" /\ saved \in [Keys -> {"absent", "empty", "old"}]
        /\ input = "none" /\ value = "none" /\ browser = "untried"
        /\ secret = "untried" /\ stage = 0 /\ run = 1
        /\ defaults = [k \in Keys |-> "none"] /\ answers = [k \in Keys |-> "none"]
        /\ values = [k \in Keys |-> "none"] /\ written = {} /\ secrets = {} /\ skipped = {}
        /\ variable = "untried" /\ failure = "none" /\ eof = FALSE
        /\ key = "public" /\ tmp = [k \in Keys |-> "absent"] /\ tempExists = FALSE
Pause == /\ phase = "start" /\ eof' \in {TRUE, FALSE} /\ phase' = "paused"
         /\ UNCHANGED <<saved, input, value, browser, secret, stage, run, defaults, answers, values, written, secrets, skipped, variable, failure, key, tmp, tempExists>>
Stage == /\ phase = "paused" /\ stage' = stage + 1 /\ phase' = "stage"
         /\ UNCHANGED <<saved, input, value, browser, secret, run, defaults, answers, values, written, secrets, skipped, variable, failure, eof, key, tmp, tempExists>>
Open == /\ phase = "stage" /\ browser' \in {"opened", "failed"} /\ phase' = "prompt"
        /\ UNCHANGED <<saved, input, value, secret, stage, run, defaults, answers, values, written, secrets, skipped, variable, failure, eof, key, tmp, tempExists>>
Reads == IF eof THEN {"eof"} ELSE {"empty", "new", "eof", "partial"}
Ask == /\ phase = "prompt" /\ input' \in Reads /\ eof' = (eof \/ input' \in {"eof", "partial"})
       /\ defaults' = [defaults EXCEPT !["public"] = Default(saved["public"])] /\ phase' = "asked"
       /\ UNCHANGED <<saved, value, browser, secret, stage, run, answers, values, written, secrets, skipped, variable, failure, key, tmp, tempExists>>
Choose == /\ phase = "asked"
          /\ value' = IF input = "empty" THEN saved["public"] ELSE IF input = "eof" THEN saved["public"] ELSE "new"
          /\ values' = [values EXCEPT !["public"] = IF value' = "absent" THEN "empty" ELSE value']
          /\ answers' = [answers EXCEPT !["public"] = input] /\ phase' = "privatePrompt"
          /\ UNCHANGED <<saved, input, browser, secret, stage, run, defaults, written, secrets, skipped, variable, failure, eof, key, tmp, tempExists>>
PrivateAsk == /\ phase = "privatePrompt" /\ input' \in Reads /\ eof' = (eof \/ input' \in {"eof", "partial"})
              /\ defaults' = [defaults EXCEPT !["private"] = Default(saved["private"])] /\ phase' = "privateAsked"
              /\ UNCHANGED <<saved, value, browser, secret, stage, run, answers, values, written, secrets, skipped, variable, failure, key, tmp, tempExists>>
PrivateChoose == /\ phase = "privateAsked" /\ values' = [values EXCEPT !["private"] = IF input \in {"new", "partial"} THEN "new" ELSE Default(saved["private"])]
                 /\ answers' = [answers EXCEPT !["private"] = input] /\ phase' = "touch"
                 /\ UNCHANGED <<saved, input, value, browser, secret, stage, run, defaults, written, secrets, skipped, variable, failure, eof, key, tmp, tempExists>>
Touch == /\ phase = "touch" /\ phase' = "mktemp" /\ UNCHANGED <<saved, input, value, browser, secret, stage, run, defaults, answers, values, written, secrets, skipped, variable, failure, eof, key, tmp, tempExists>>
Mktemp == /\ phase = "mktemp" /\ tempExists' = TRUE /\ tmp' = [k \in Keys |-> "absent"] /\ phase' = "copy"
          /\ UNCHANGED <<saved, input, value, browser, secret, stage, run, defaults, answers, values, written, secrets, skipped, variable, failure, eof, key>>
Copy == /\ phase = "copy" /\ tmp' \in (IF CopiesWork THEN {[saved EXCEPT ![key] = "absent"]} ELSE {[saved EXCEPT ![key] = "absent"], [k \in Keys |-> "absent"]})
        /\ phase' = "append" /\ UNCHANGED <<saved, input, value, browser, secret, stage, run, defaults, answers, values, written, secrets, skipped, variable, failure, eof, key, tempExists>>
Append == /\ phase = "append" /\ tmp' = [tmp EXCEPT ![key] = values[key]] /\ phase' = "mv"
          /\ UNCHANGED <<saved, input, value, browser, secret, stage, run, defaults, answers, values, written, secrets, skipped, variable, failure, eof, key, tempExists>>
Move == /\ phase = "mv" /\ saved' = tmp /\ tempExists' = FALSE /\ phase' = "record"
        /\ UNCHANGED <<input, value, browser, secret, stage, run, defaults, answers, values, written, secrets, skipped, variable, failure, eof, key, tmp>>
Record == /\ phase = "record" /\ written' = written \cup {key}
          /\ phase' = (IF key = "public" THEN "touch" ELSE "secret") /\ key' = "private"
          /\ UNCHANGED <<saved, input, value, browser, secret, stage, run, defaults, answers, values, secrets, skipped, variable, failure, eof, tmp, tempExists>>
Secret == /\ phase = "secret" /\ secret' \in {"set", "skipped"}
          /\ secrets' = IF secret' = "set" THEN secrets \cup {"private"} ELSE secrets
          /\ skipped' = (IF secret' = "skipped" THEN skipped \cup {"secret"} ELSE skipped) /\ phase' = "summary"
          /\ UNCHANGED <<saved, input, value, browser, stage, run, defaults, answers, values, written, variable, failure, eof, key, tmp, tempExists>>
SetVariable == /\ phase = "summary" /\ variable' \in {"set", "skipped"}
               /\ skipped' = (IF variable' = "skipped" THEN skipped \cup {"variable"} ELSE skipped) /\ phase' = "variableSummary"
               /\ UNCHANGED <<saved, input, value, browser, secret, stage, run, defaults, answers, values, written, secrets, failure, eof, key, tmp, tempExists>>
Finish == /\ phase \in {"summary", "variableSummary"} /\ phase' = "done" /\ UNCHANGED <<saved, input, value, browser, secret, stage, run, defaults, answers, values, written, secrets, skipped, variable, failure, eof, key, tmp, tempExists>>
Rerun == /\ phase \in {"done", "aborted"} /\ run = 1 /\ phase' = "start" /\ run' = 2 /\ stage' = 0
         /\ input' = "none" /\ value' = "none" /\ browser' = "untried" /\ secret' = "untried"
         /\ defaults' = [k \in Keys |-> "none"] /\ answers' = [k \in Keys |-> "none"] /\ values' = [k \in Keys |-> "none"]
         /\ written' = {} /\ secrets' = {} /\ skipped' = {} /\ variable' = "untried" /\ failure' = "none"
         /\ eof' = FALSE /\ key' = "public" /\ tmp' = [k \in Keys |-> "absent"] /\ tempExists' = FALSE /\ UNCHANGED saved
Abort == /\ phase \notin {"done", "aborted"} /\ phase' = "aborted" /\ failure' = "unhandled"
         \* External mv failure may leave original, empty, or partially published destination.
         /\ saved' \in IF phase = "mv" THEN {saved, [k \in Keys |-> "absent"], tmp} ELSE {saved}
         /\ tmp' \in IF phase = "append" THEN {tmp, [tmp EXCEPT ![key] = "partial"]} ELSE {tmp}
         /\ UNCHANGED <<input, value, browser, secret, stage, run, defaults, answers, values, written, secrets, skipped, variable, eof, key, tempExists>>
Done == /\ phase \in {"done", "aborted"} /\ UNCHANGED vars
Next == Pause \/ Stage \/ Open \/ Ask \/ Choose \/ PrivateAsk \/ PrivateChoose \/ Touch \/ Mktemp \/ Copy \/ Append \/ Move \/ Record \/ Secret \/ SetVariable \/ Finish \/ Rerun \/ Abort \/ Done
DefaultPreserved == (phase = "done" /\ answers["public"] = "empty") => saved["public"] = defaults["public"]
ChartNext == (Pause /\ ~eof') \/ Stage \/ (Open /\ browser' = "failed") \/ (Ask /\ input' = "empty") \/ Choose \/ (PrivateAsk /\ input' = "empty") \/ PrivateChoose \/ Touch \/ Mktemp \/ Copy \/ Append \/ Move \/ Record \/ (Secret /\ secret' = "skipped") \/ Finish \/ Done
=============================================================================
