---------------------------- MODULE Flows ----------------------------
EXTENDS Naturals, Sequences
VARIABLES phase, mode, result, owner, closed, originalTimeout, requestTimeout,
          context, read, relation, codexPresent, bookLookup, codexLookup, ids, members
vars == <<phase, mode, result, owner, closed, originalTimeout, requestTimeout,
          context, read, relation, codexPresent, bookLookup, codexLookup, ids, members>>
Init == /\ phase = "start" /\ mode = "none" /\ result = "none"
        /\ owner = "none" /\ closed = FALSE /\ originalTimeout = "unset"
        /\ requestTimeout = "unset" /\ context = "caller" /\ read = "none"
        /\ relation = "none" /\ codexPresent = FALSE
        /\ bookLookup = "none" /\ codexLookup = "none" /\ ids = <<>> /\ members = <<>>
Choose == /\ phase = "start" /\ mode' \in {"raw", "json", "execute", "stream", "book", "list", "series"}
          /\ originalTimeout' \in {"positive", "nonpositive"} /\ requestTimeout' = originalTimeout'
          /\ phase' = IF mode' \in {"book", "list", "series"} THEN "primary" ELSE "marshal"
          /\ UNCHANGED <<result, owner, closed, context, read, relation, codexPresent, bookLookup, codexLookup, ids, members>>
Marshal == /\ phase = "marshal" /\ phase' \in {"build", "returned"}
           /\ result' = IF phase' = "returned" THEN "marshal-error" ELSE result
           /\ UNCHANGED <<mode, owner, closed, originalTimeout, requestTimeout, context, read, relation, codexPresent, bookLookup, codexLookup, ids, members>>
Build == /\ phase = "build" /\ phase' \in {"transport", "returned"}
         /\ result' = IF phase' = "returned" THEN "build-error" ELSE result
         /\ requestTimeout' = IF phase' = "transport" /\ mode = "stream" /\ originalTimeout = "positive" THEN "nonpositive" ELSE requestTimeout
         /\ UNCHANGED <<mode, owner, closed, originalTimeout, context, read, relation, codexPresent, bookLookup, codexLookup, ids, members>>
Transport == /\ phase = "transport" /\ phase' \in {"status", "returned"}
             /\ owner' = IF phase' = "status" THEN "helper" ELSE owner
             /\ result' \in (IF phase' = "returned" THEN {"transport-error", "context-error"} ELSE {result})
             /\ UNCHANGED <<mode, closed, originalTimeout, requestTimeout, context, read, relation, codexPresent, bookLookup, codexLookup, ids, members>>
Reject == /\ phase = "status" /\ phase' = "returned" /\ result' = "status-error"
          /\ closed' = TRUE /\ owner' = "none" /\ read' \in {"error-body", "error-body-failed"}
          /\ UNCHANGED <<mode, originalTimeout, requestTimeout, context, relation, codexPresent, bookLookup, codexLookup, ids, members>>
Accept == /\ phase = "status" /\ phase' = CASE mode = "stream" -> "caller" [] mode = "execute" -> "returned" [] OTHER -> "read"
          /\ owner' = IF mode = "stream" THEN "caller" ELSE IF mode = "execute" THEN "none" ELSE owner
          /\ closed' = (mode = "execute") /\ result' = IF mode = "execute" THEN "success" ELSE result
          /\ UNCHANGED <<mode, originalTimeout, requestTimeout, context, read, relation, codexPresent, bookLookup, codexLookup, ids, members>>
Read == /\ phase = "read" /\ read' \in {"bytes", "failed"}
        /\ phase' = IF read' = "bytes" /\ mode = "json" THEN "decode" ELSE "returned"
        /\ result' = IF read' = "failed" THEN "read-error" ELSE IF mode = "raw" THEN "success" ELSE result
        /\ closed' = TRUE /\ owner' = "none"
        /\ UNCHANGED <<mode, originalTimeout, requestTimeout, context, relation, codexPresent, bookLookup, codexLookup, ids, members>>
Decode == /\ phase = "decode" /\ phase' = "returned" /\ result' \in {"success", "decode-error"}
          /\ UNCHANGED <<mode, owner, closed, originalTimeout, requestTimeout, context, read, relation, codexPresent, bookLookup, codexLookup, ids, members>>
CallerClose == /\ phase = "caller" /\ phase' = "returned" /\ owner' = "none" /\ closed' = TRUE
               /\ result' \in {"success", "stream-error", "context-error"}
               /\ UNCHANGED <<mode, originalTimeout, requestTimeout, context, read, relation, codexPresent, bookLookup, codexLookup, ids, members>>
\* Primary is the atomic S16 request plus endpoint decode outcome; no composition proof.
Primary == /\ phase = "primary" /\ phase' \in {"returned", "relations"}
           /\ result' = IF phase' = "returned" THEN "primary-error" ELSE result
           /\ relation' = IF phase' = "relations" THEN "decoded" ELSE relation
           /\ UNCHANGED <<mode, owner, closed, originalTimeout, requestTimeout, context, read, codexPresent, bookLookup, codexLookup, ids, members>>
Relations == /\ phase = "relations" /\ relation' \in {"present", "empty"}
             /\ codexPresent' \in (IF mode = "series" THEN BOOLEAN ELSE {FALSE})
             /\ phase' = IF mode = "book" THEN "chapters" ELSE "books"
             /\ ids' \in (IF mode \in {"list", "series"} THEN {<<>>, <<2, 1>>} ELSE {<<>>})
             /\ UNCHANGED <<mode, result, owner, closed, originalTimeout, requestTimeout, context, read, bookLookup, codexLookup, members>>
Chapters == /\ phase = "chapters" /\ phase' = "returned" /\ result' = "book"
            /\ bookLookup' \in (IF relation = "present" THEN {"skipped"} ELSE {"populated", "empty", "unavailable", "partial-decodes"})
            /\ UNCHANGED <<mode, owner, closed, originalTimeout, requestTimeout, context, read, relation, codexPresent, codexLookup, ids, members>>
Books == /\ phase = "books" /\ phase' = IF mode = "series" THEN "codex" ELSE "returned"
         /\ bookLookup' \in (IF relation = "present" THEN {"skipped"} ELSE {"populated", "empty", "unavailable"})
         /\ members' \in (IF bookLookup' = "populated" THEN IF ids = <<>> THEN {<<1, 2>>, <<1>>, <<2>>, <<>>} ELSE {<<2, 1>>, <<2>>, <<1>>, <<>>} ELSE {<<>>})
         /\ result' = IF mode = "list" THEN "series" ELSE result
         /\ UNCHANGED <<mode, owner, closed, originalTimeout, requestTimeout, context, read, relation, codexPresent, codexLookup, ids>>
Codex == /\ phase = "codex" /\ phase' = "returned" /\ result' = "series"
         /\ codexLookup' \in (IF codexPresent THEN {"skipped"} ELSE {"populated", "empty", "unavailable"})
         /\ UNCHANGED <<mode, owner, closed, originalTimeout, requestTimeout, context, read, relation, codexPresent, bookLookup, ids, members>>
\* Observe the original client's timeout on a later ordinary request, without composing another request.
LaterRequest == /\ phase = "returned" /\ mode = "stream" /\ phase' = "later"
                /\ requestTimeout' = originalTimeout
                /\ UNCHANGED <<mode, result, owner, closed, originalTimeout, context, read, relation, codexPresent, bookLookup, codexLookup, ids, members>>
Done == /\ phase \in {"returned", "later"} /\ UNCHANGED vars
Next == Choose \/ Marshal \/ Build \/ Transport \/ Reject \/ Accept \/ Read \/ Decode \/ CallerClose \/ Primary \/ Relations \/ Chapters \/ Books \/ Codex \/ LaterRequest \/ Done
StreamRejectClosed == (mode = "stream" /\ result = "status-error") => closed
=============================================================================
