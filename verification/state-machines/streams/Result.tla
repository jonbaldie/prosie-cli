---------------- MODULE Result ----------------
EXTENDS Naturals
VARIABLES phase, seenError, retainedError, done, count, callbacks, readOutcome, cancelled, result
vars == <<phase, seenError, retainedError, done, count, callbacks, readOutcome, cancelled, result>>
Init == /\ phase = "reading" /\ seenError = FALSE /\ retainedError = FALSE
        /\ done = "none" /\ count = 0 /\ callbacks = 0
        /\ readOutcome = "none" /\ cancelled = FALSE /\ result = "none"
ErrorEvent == /\ phase = "reading" /\ count < 3
              /\ seenError' = TRUE /\ retainedError' = TRUE /\ count' = count + 1
              /\ UNCHANGED <<phase, done, callbacks, readOutcome, cancelled, result>>
DoneEvent == /\ phase = "reading" /\ count < 3
             /\ done' \in {"one", "two", "zero"} /\ count' = count + 1
             /\ UNCHANGED <<phase, seenError, retainedError, callbacks, readOutcome, cancelled, result>>
DeltaEvent == /\ phase = "reading" /\ count < 3
              /\ callbacks' = callbacks + 1 /\ count' = count + 1
              /\ UNCHANGED <<phase, seenError, retainedError, done, readOutcome, cancelled, result>>
\* Malformed or empty done is ignored; JSON null decodes the zero value.
InvalidDone == /\ phase = "reading" /\ count < 3
               /\ count' = count + 1
               /\ UNCHANGED <<phase, seenError, retainedError, done, callbacks, readOutcome, cancelled, result>>
OtherEvent == /\ phase = "reading" /\ count < 3
              /\ count' = count + 1
              /\ UNCHANGED <<phase, seenError, retainedError, done, callbacks, readOutcome, cancelled, result>>
End == /\ phase = "reading" /\ phase' = "decide"
       /\ readOutcome' \in {"eof", "readError"} /\ cancelled' \in BOOLEAN
       /\ UNCHANGED <<seenError, retainedError, done, count, callbacks, result>>
Return == /\ phase = "decide" /\ phase' = "returned"
          /\ result' = IF cancelled THEN "context" ELSE IF readOutcome = "readError" THEN "readError"
                        ELSE IF retainedError THEN "streamError" ELSE IF done = "none" THEN "missing" ELSE done
          /\ UNCHANGED <<seenError, retainedError, done, count, callbacks, readOutcome, cancelled>>
Terminal == /\ phase = "returned" /\ UNCHANGED vars
Next == ErrorEvent \/ DoneEvent \/ DeltaEvent \/ InvalidDone \/ OtherEvent \/ End \/ Return \/ Terminal
ErrorRetained == seenError => retainedError
ReturnPrecedence == phase = "returned" =>
  /\ (cancelled => result = "context")
  /\ (~cancelled /\ readOutcome = "readError" => result = "readError")
  /\ (~cancelled /\ readOutcome = "eof" /\ retainedError => result = "streamError")
  /\ (~cancelled /\ readOutcome = "eof" /\ ~retainedError /\ done = "none" => result = "missing")
  /\ (~cancelled /\ readOutcome = "eof" /\ ~retainedError /\ done # "none" => result = done)
================================================
