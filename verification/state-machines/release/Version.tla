------------------------------ MODULE Version ------------------------------
EXTENDS Naturals, Sequences
VARIABLES phase, candidates, index, value, status
vars == <<phase, candidates, index, value, status>>
\* Candidate classes capture failed directory queries and duplicate directories.
Init == /\ candidates \in {<<>>, <<"exe">>, <<"cwd">>, <<"exe", "cwd">>}
        /\ phase = "start" /\ index = 1 /\ value \in {"linked", ""} /\ status = "unread"
Start == /\ phase = "start"
         /\ IF value = "linked" THEN /\ phase' = "done" /\ UNCHANGED value
            ELSE /\ value' = "fallback" /\ phase' = "status"
         /\ UNCHANGED <<candidates, index, status>>
Exhausted == /\ phase = "status" /\ index > Len(candidates)
             /\ phase' = "done" /\ UNCHANGED <<candidates, index, value, status>>
Status == /\ phase = "status" /\ index <= Len(candidates)
          /\ status' \in {"clean", "dirty", "failed"}
          /\ IF status' = "clean" THEN /\ phase' = "tag" /\ UNCHANGED index
             ELSE /\ phase' = "status" /\ index' = index + 1
          /\ UNCHANGED <<candidates, value>>
TagSuccess == /\ phase = "tag" /\ phase' = "done" /\ value' = "tag"
              /\ UNCHANGED <<candidates, status, index>>
TagFailure == /\ phase = "tag" /\ phase' = "status" /\ index' = index + 1
              /\ UNCHANGED <<candidates, status, value>>
Done == /\ phase = "done" /\ UNCHANGED vars
Next == Start \/ Exhausted \/ Status \/ TagSuccess \/ TagFailure \/ Done
CleanTag == value = "tag" => status = "clean"
=============================================================================
