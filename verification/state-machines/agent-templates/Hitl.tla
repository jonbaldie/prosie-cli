----------------------------- MODULE Hitl -----------------------------
EXTENDS Naturals
VARIABLES phase, input, printed, signInput, firstInput
vars == <<phase, input, printed, signInput, firstInput>>
Init == /\ phase = "signin" /\ input = "none" /\ printed = FALSE
        /\ signInput = "none" /\ firstInput = "none"
SignIn == /\ phase = "signin" /\ signInput' \in {"line", "eof"}
          /\ phase' = IF signInput' = "line" THEN "capture" ELSE "aborted"
          /\ UNCHANGED <<input, printed, firstInput>>
FirstCapture == /\ phase = "capture" /\ firstInput' \in {"line", "eof"}
                /\ phase' = IF firstInput' = "line" THEN "message" ELSE "aborted"
                /\ UNCHANGED <<input, printed, signInput>>
LastCapture == /\ phase = "message" /\ input' \in {"line", "eof"}
               /\ phase' = IF input' = "line" THEN "summary" ELSE "aborted"
               /\ UNCHANGED <<printed, signInput, firstInput>>
Print == /\ phase = "summary" /\ phase' = "done" /\ printed' = TRUE
         /\ UNCHANGED <<input, signInput, firstInput>>
Done == /\ phase \in {"done", "aborted"} /\ UNCHANGED vars
Next == SignIn \/ FirstCapture \/ LastCapture \/ Print \/ Done
NoSummaryOnEOF == input = "eof" => ~printed
=============================================================================
