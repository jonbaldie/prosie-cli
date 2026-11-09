----------------------------- MODULE Output -----------------------------
EXTENDS Naturals
VARIABLES phase, latched, writes, afterFailure, handler, result, attempt
vars == <<phase, latched, writes, afterFailure, handler, result, attempt>>
Init == /\ phase = "writing" /\ latched = "none" /\ writes = 0
        /\ afterFailure = FALSE /\ handler \in {0, 1} /\ result = "pending" /\ attempt = 0
Write == /\ phase = "writing" /\ attempt < 2
         /\ attempt' = attempt + 1
         /\ IF latched # "none"
               THEN /\ UNCHANGED <<latched, writes, afterFailure>>
               ELSE /\ writes' = writes + 1
                    /\ latched' \in {"none", "short", "error"}
                    /\ afterFailure' = afterFailure
         /\ UNCHANGED <<phase, handler, result>>
Return == /\ phase = "writing" /\ attempt \in 0..2
          /\ phase' = "returned"
          /\ result' = IF handler = 0 /\ latched # "none" THEN 1 ELSE handler
          /\ UNCHANGED <<latched, writes, afterFailure, handler, attempt>>
Done == /\ phase = "returned" /\ UNCHANGED vars
Next == Write \/ Return \/ Done
NoWriteAfterFailure == ~afterFailure
=============================================================================
