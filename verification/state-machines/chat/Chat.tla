------------------------------ MODULE Chat ------------------------------
EXTENDS Naturals, Sequences
CONSTANT Chart
VARIABLES phase, made, created, id, returnedID, mode, callback, tokens, output, writeFailed, deltaPrinted, response, outcome
vars == <<phase, made, created, id, returnedID, mode, callback, tokens, output, writeFailed, deltaPrinted, response, outcome>>
Init == /\ phase = "select" /\ made = FALSE /\ created = FALSE
        /\ id = "" /\ returnedID = "" /\ mode = "none" /\ callback = "none"
        /\ tokens = <<>> /\ output = <<>> /\ writeFailed = FALSE /\ deltaPrinted = FALSE
        /\ response = "none" /\ outcome = "none"
Existing == /\ phase = "select" /\ id' \in (IF Chart THEN {"existing"} ELSE {"existing", "space"}) /\ phase' = "choose"
            /\ UNCHANGED <<made, created, returnedID, mode, callback, tokens, output, writeFailed, deltaPrinted, response, outcome>>
Create == /\ phase = "select" /\ returnedID' \in (IF Chart THEN {"77"} ELSE {"77", "0"}) /\ phase' = "choose"
          /\ made' = TRUE /\ created' = TRUE /\ id' = returnedID'
          /\ UNCHANGED <<mode, callback, tokens, output, writeFailed, deltaPrinted, response, outcome>>
CreateFail == /\ phase = "select" /\ phase' = "failed" /\ outcome' = "create-error"
              /\ UNCHANGED <<made, created, id, returnedID, mode, callback, tokens, output, writeFailed, deltaPrinted, response>>
Choose == /\ phase = "choose" /\ mode' \in (IF Chart THEN {"stream-text", "stream-json"} ELSE {"ordinary-text", "ordinary-json", "stream-text", "stream-json"})
          /\ callback' = IF mode' \in {"ordinary-text", "ordinary-json"} THEN "nil" ELSE "non-nil"
          /\ phase' = IF id = "space" THEN "failed" ELSE "request"
          /\ outcome' = IF id = "space" THEN "invalid-id" ELSE outcome
          /\ UNCHANGED <<made, created, id, returnedID, tokens, output, writeFailed, deltaPrinted, response>>
Delta == /\ phase = "request" /\ mode \in {"stream-text", "stream-json"} /\ Len(tokens) < (IF Chart THEN 1 ELSE 2)
         /\ \E t \in {"text", "newline"}, ok \in BOOLEAN:
              /\ tokens' = Append(tokens, t)
              /\ deltaPrinted' = IF mode = "stream-text" /\ ~writeFailed /\ ok THEN TRUE ELSE deltaPrinted
              /\ output' = IF mode = "stream-text" /\ ~writeFailed THEN Append(output, IF ok THEN t ELSE "partial") ELSE output
              /\ writeFailed' = (writeFailed \/ (mode = "stream-text" /\ ~ok))
         /\ UNCHANGED <<phase, made, created, id, returnedID, mode, callback, response, outcome>>
Success == /\ phase = "request" /\ phase' = "report" /\ outcome' = "success"
           /\ response' \in {"nil-message", "text-message"}
           /\ UNCHANGED <<made, created, id, returnedID, mode, callback, tokens, output, writeFailed, deltaPrinted>>
Failure == /\ phase = "request" /\ phase' = "report" /\ outcome' = "send-error"
           /\ UNCHANGED <<made, created, id, returnedID, mode, callback, tokens, output, writeFailed, deltaPrinted, response>>
Report == /\ phase = "report"
          /\ \E ok \in BOOLEAN:
               /\ LET item == IF outcome = "send-error" THEN IF mode = "stream-text" /\ Len(tokens) > 0 THEN "finish" ELSE "none"
                              ELSE IF mode \in {"stream-json", "ordinary-json"} THEN "json"
                              ELSE IF mode = "stream-text" THEN "finish"
                              ELSE IF response = "text-message" THEN "message" ELSE "none"
                  IN /\ phase' = IF outcome = "success" /\ ~writeFailed /\ (item = "none" \/ ok) THEN "done" ELSE "failed"
                     /\ output' = IF item # "none" /\ ~writeFailed THEN Append(output, IF ok THEN item ELSE "partial") ELSE output
                     /\ writeFailed' = (writeFailed \/ (item # "none" /\ ~ok))
          /\ UNCHANGED <<made, created, id, returnedID, mode, callback, tokens, deltaPrinted, response, outcome>>
Finished == /\ phase \in {"done", "failed"} /\ UNCHANGED vars
Next == Existing \/ Create \/ CreateFail \/ Choose \/ Delta \/ Success \/ Failure \/ Report \/ Finished
RetainCreated == made => created /\ id = returnedID
JSONSuppressesDeltas == mode = "stream-json" => /\ ~deltaPrinted /\ \A i \in 1..Len(output): output[i] \notin {"text", "newline"}
=============================================================================
