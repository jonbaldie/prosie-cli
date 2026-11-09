----------------------------- MODULE Content -----------------------------
EXTENDS Naturals
VARIABLES flow, inline, path, present, filePresent, input, readFails,
          prompt, instruction, action, phase, consumed, value, result, patch, selectedPrompt
vars == <<flow, inline, path, present, filePresent, input, readFails,
          prompt, instruction, action, phase, consumed, value, result, patch, selectedPrompt>>
Text == {"empty", "space", "text"}
Init ==
  /\ flow \in {"chapter", "rewrite"}
  /\ inline \in Text /\ path \in {"empty", "stdin", "file"}
  /\ present \in BOOLEAN /\ filePresent \in BOOLEAN
  /\ (present \/ inline = "empty")
  /\ (filePresent \/ path = "empty")
  /\ input \in Text /\ readFails \in BOOLEAN
  /\ prompt \in Text /\ instruction \in Text /\ action \in Text
  /\ phase = "select" /\ consumed = FALSE /\ value = "empty"
  /\ result = "none" /\ patch = "absent" /\ selectedPrompt = "empty"
Blank(t) == t \in {"empty", "space"}
Select ==
  /\ phase = "select"
  /\ IF flow = "chapter" /\ (~filePresent \/ path = "empty")
        THEN /\ value' = inline /\ phase' = "build" /\ UNCHANGED <<consumed, result>>
        ELSE IF path = "empty"
          THEN /\ value' = inline /\ phase' = "validate" /\ UNCHANGED <<consumed, result>>
          ELSE IF inline # "empty"
            THEN /\ result' = "conflict" /\ phase' = "done" /\ UNCHANGED <<value, consumed>>
            ELSE /\ phase' = "read" /\ UNCHANGED <<value, consumed, result>>
  /\ UNCHANGED <<flow, inline, path, present, filePresent, input, readFails, prompt, instruction, action, patch, selectedPrompt>>
Read ==
  /\ phase = "read"
  /\ consumed' = (path = "stdin")
  /\ IF readFails
        THEN /\ result' = "readError" /\ phase' = "done" /\ value' = "empty"
        ELSE /\ value' = input /\ phase' = (IF flow = "chapter" THEN "build" ELSE "validate") /\ UNCHANGED result
  /\ UNCHANGED <<flow, inline, path, present, filePresent, input, readFails, prompt, instruction, action, patch, selectedPrompt>>
Build ==
  /\ phase = "build"
  /\ patch' = IF filePresent /\ path # "empty" THEN value ELSE IF present THEN inline ELSE "absent"
  /\ result' = "ready" /\ phase' = "done"
  /\ UNCHANGED <<flow, inline, path, present, filePresent, input, readFails, prompt, instruction, action, consumed, value, selectedPrompt>>
Validate ==
  /\ phase = "validate"
  /\ IF Blank(value)
       THEN /\ result' = "selectionError" /\ selectedPrompt' = "empty"
       ELSE /\ selectedPrompt' = IF prompt = "empty" THEN instruction ELSE prompt
            /\ result' = IF Blank(IF prompt = "empty" THEN instruction ELSE prompt) /\ Blank(action) THEN "promptError" ELSE "ready"
  /\ phase' = "done"
  /\ UNCHANGED <<flow, inline, path, present, filePresent, input, readFails, prompt, instruction, action, consumed, value, patch>>
Done == /\ phase = "done" /\ UNCHANGED vars
Next == Select \/ Read \/ Build \/ Validate \/ Done
NoConflictConsumption == (path # "empty" /\ inline # "empty" /\ (flow = "rewrite" \/ filePresent)) => ~consumed
=============================================================================
