---------------------------- MODULE Upload ----------------------------
EXTENDS Naturals
VARIABLES phase, source, owner, inputOK, request, consumed, result
vars == <<phase, source, owner, inputOK, request, consumed, result>>
Init == /\ phase = "validate" /\ source \in {"file", "stdin"}
        /\ owner = "none" /\ inputOK = FALSE /\ request = FALSE
        /\ consumed = FALSE /\ result = "none"
Validate == /\ phase = "validate" /\ phase' \in {"open", "failed"}
            /\ UNCHANGED <<source, owner, inputOK, request, consumed, result>>
Open == /\ phase = "open" /\ phase' = "auth"
        /\ owner' = IF source = "file" THEN "open" ELSE "stdin"
        /\ UNCHANGED <<source, inputOK, request, consumed, result>>
OpenError == /\ phase = "open" /\ source = "file" /\ phase' = "failed"
             /\ UNCHANGED <<source, owner, inputOK, request, consumed, result>>
Auth == /\ phase = "auth" /\ phase' \in {"fields", "cleanup"}
        /\ UNCHANGED <<source, owner, inputOK, request, consumed, result>>
Fields == /\ phase = "fields" /\ phase' \in {"part", "cleanup"}
          /\ UNCHANGED <<source, owner, inputOK, request, consumed, result>>
Part == /\ phase = "part" /\ phase' \in {"copy", "cleanup"}
        /\ UNCHANGED <<source, owner, inputOK, request, consumed, result>>
CopyOK == /\ phase = "copy" /\ phase' = "multipartClose" /\ consumed' = TRUE
          /\ UNCHANGED <<source, owner, inputOK, request, result>>
CopyError == /\ phase = "copy" /\ phase' = "cleanup" /\ consumed' = TRUE
             /\ UNCHANGED <<source, owner, inputOK, request, result>>
CloseMultipart == /\ phase = "multipartClose" /\ phase' = "build" /\ inputOK' = TRUE
                  /\ UNCHANGED <<source, owner, request, consumed, result>>
Build == /\ phase = "build" /\ phase' \in {"send", "cleanup"}
         /\ UNCHANGED <<source, owner, inputOK, request, consumed, result>>
Send == /\ phase = "send" /\ phase' = "response" /\ request' = TRUE
        /\ UNCHANGED <<source, owner, inputOK, consumed, result>>
Response == /\ phase = "response" /\ phase' \in {"read", "cleanup"}
            /\ UNCHANGED <<source, owner, inputOK, request, consumed, result>>
Read == /\ phase = "read" /\ phase' \in {"decode", "cleanup"}
        /\ UNCHANGED <<source, owner, inputOK, request, consumed, result>>
Decode == /\ phase = "decode" /\ phase' \in {"report", "cleanup"}
          /\ UNCHANGED <<source, owner, inputOK, request, consumed, result>>
Report == /\ phase = "report" /\ phase' = "cleanup" /\ result' \in {"success", "outputError"}
          /\ UNCHANGED <<source, owner, inputOK, request, consumed>>
Cleanup == /\ phase = "cleanup" /\ phase' = "done"
           /\ owner' = IF owner = "open" THEN "closed" ELSE owner
           /\ UNCHANGED <<source, inputOK, request, consumed, result>>
Done == /\ phase \in {"failed", "done"} /\ UNCHANGED vars
Next == Validate \/ Open \/ OpenError \/ Auth \/ Fields \/ Part \/ CopyOK \/ CopyError \/ CloseMultipart \/ Build \/ Send \/ Response \/ Read \/ Decode \/ Report \/ Cleanup \/ Done
NoBadRequest == request => inputOK
FileClosed == phase = "done" /\ source = "file" => owner = "closed"
=============================================================================
