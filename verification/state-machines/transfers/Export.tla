---------------------------- MODULE Export ----------------------------
EXTENDS Naturals
VARIABLES phase, kind, destination, json, data, published, success, output
vars == <<phase, kind, destination, json, data, published, success, output>>
Init == /\ phase = "validate" /\ kind \in {"prose", "docx", "chat"}
        /\ destination \in {"file", "stdout"} /\ json \in BOOLEAN
        /\ data = "none" /\ published = FALSE /\ success = FALSE /\ output = "none"
Validate == /\ phase = "validate" /\ phase' \in {"auth", "failed"}
            /\ UNCHANGED <<kind, destination, json, data, published, success, output>>
Auth == /\ phase = "auth" /\ phase' \in {"fetch", "failed"}
        /\ UNCHANGED <<kind, destination, json, data, published, success, output>>
FetchOK == /\ phase = "fetch" /\ phase' = "route"
           /\ data' \in {"valid", "validNL", "malformed", "malformedNL"}
           /\ UNCHANGED <<kind, destination, json, published, success, output>>
FetchError == /\ phase = "fetch" /\ phase' = "failed"
              /\ UNCHANGED <<kind, destination, json, data, published, success, output>>
Route == /\ phase = "route" /\ phase' = IF destination = "file" THEN "write" ELSE "render"
         /\ UNCHANGED <<kind, destination, json, data, published, success, output>>
WriteOK == /\ phase = "write" /\ phase' = "report" /\ published' = TRUE
           /\ UNCHANGED <<kind, destination, json, data, success, output>>
WriteError == /\ phase = "write" /\ phase' = "failed" /\ UNCHANGED <<kind, destination, json, data, published, success, output>>
Render == /\ phase = "render"
          /\ IF kind = "chat" /\ json /\ data \in {"malformed", "malformedNL"}
                THEN /\ phase' = "decodeError" /\ output' = "none" /\ success' = FALSE
                ELSE /\ phase' = "done" /\ success' = TRUE
                     /\ output' \in {"outputError", IF kind = "chat" THEN (IF json THEN "chatJSON" ELSE IF data \in {"validNL", "malformedNL"} THEN "chatRaw" ELSE "chatAddedNewline")
                                  ELSE IF json THEN (IF kind = "docx" THEN "base64" ELSE "proseJSON") ELSE "raw"}
          /\ UNCHANGED <<kind, destination, json, data, published>>
Report == /\ phase = "report" /\ phase' = "done" /\ success' = TRUE
          /\ output' \in {"report", "outputError"}
          /\ UNCHANGED <<kind, destination, json, data, published>>
Done == /\ phase \in {"failed", "decodeError", "done"} /\ UNCHANGED vars
Next == Validate \/ Auth \/ FetchOK \/ FetchError \/ Route \/ WriteOK \/ WriteError \/ Render \/ Report \/ Done
SuccessNeedsPublication == success /\ destination = "file" => published
FileRaw == destination = "file" => phase # "decodeError"
=============================================================================
