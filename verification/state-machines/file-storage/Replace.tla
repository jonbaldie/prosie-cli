---------------- MODULE Replace ----------------
EXTENDS Naturals
VARIABLES phase, kind, mode, requestedMode, tempMode, publishedMode, contents, failed, temp, closed, tempBytes, result, cleanupFailed, writeOK, syncOK, closeOK, published, target
vars == <<phase, kind, mode, requestedMode, tempMode, publishedMode, contents, failed, temp, closed, tempBytes, result, cleanupFailed, writeOK, syncOK, closeOK, published, target>>
Init == /\ phase = "stat" /\ kind \in {"regular", "missing", "special", "staterror"}
        /\ mode = 420 /\ requestedMode \in {384, 420} /\ tempMode = 0 /\ publishedMode = mode
        /\ contents = "old" /\ failed = FALSE /\ temp = FALSE /\ closed = FALSE
        /\ tempBytes = "empty" /\ result = "pending" /\ cleanupFailed = FALSE
        /\ writeOK = FALSE /\ syncOK = FALSE /\ closeOK = FALSE /\ published = FALSE /\ target = "unresolved"
Stat == /\ phase = "stat"
        /\ phase' = CASE kind = "staterror" -> "error" [] kind = "special" -> "direct" [] OTHER -> "resolve"
        /\ result' = IF kind = "staterror" THEN "failure" ELSE result
        /\ UNCHANGED <<kind, mode, requestedMode, tempMode, publishedMode, contents, failed, temp, closed, tempBytes, cleanupFailed, writeOK, syncOK, closeOK, published, target>>
Direct == /\ phase = "direct" /\ contents' \in {"old", "new", "partial"}
          /\ result' \in (IF contents' = "new" THEN {"success", "failure"} ELSE {"failure"})
          /\ phase' = IF result' = "success" THEN "done" ELSE "error"
          /\ UNCHANGED <<kind, mode, requestedMode, tempMode, publishedMode, failed, temp, closed, tempBytes, cleanupFailed, writeOK, syncOK, closeOK, published, target>>
Resolve == /\ phase = "resolve" /\ phase' \in {"create", "error"}
           /\ result' = IF phase' = "error" THEN "failure" ELSE result
           /\ target' = IF phase' = "create" THEN "resolved" ELSE target
           /\ UNCHANGED <<kind, mode, requestedMode, tempMode, publishedMode, contents, failed, temp, closed, tempBytes, cleanupFailed, writeOK, syncOK, closeOK, published>>
Create == /\ phase = "create" /\ phase' \in {"chmod", "error"}
          /\ temp' = (phase' = "chmod") /\ result' = IF temp' THEN result ELSE "failure"
          /\ tempMode' = IF temp' THEN 384 ELSE tempMode
          /\ UNCHANGED <<kind, mode, requestedMode, publishedMode, contents, failed, closed, tempBytes, cleanupFailed, writeOK, syncOK, closeOK, published, target>>
SelectedMode == IF kind = "regular" THEN mode ELSE requestedMode
Chmod == /\ phase = "chmod" /\ failed' \in BOOLEAN
         /\ tempMode' \in (IF failed' THEN {tempMode, SelectedMode} ELSE {SelectedMode})
         /\ phase' = IF failed' THEN "close" ELSE "write"
         /\ UNCHANGED <<kind, mode, requestedMode, publishedMode, contents, temp, closed, tempBytes, result, cleanupFailed, writeOK, syncOK, closeOK, published, target>>
Write == /\ phase = "write" /\ tempBytes' \in {"new", "partial", "empty"}
         /\ writeOK' \in (IF tempBytes' = "new" THEN BOOLEAN ELSE {FALSE})
         /\ failed' = ~writeOK' /\ phase' = IF failed' THEN "close" ELSE "sync"
         /\ UNCHANGED <<kind, mode, requestedMode, tempMode, publishedMode, contents, temp, closed, result, cleanupFailed, syncOK, closeOK, published, target>>
Sync == /\ phase = "sync" /\ syncOK' \in BOOLEAN /\ failed' = ~syncOK' /\ phase' = "close"
        /\ UNCHANGED <<kind, mode, requestedMode, tempMode, publishedMode, contents, temp, closed, tempBytes, result, cleanupFailed, writeOK, closeOK, published, target>>
Close == /\ phase = "close" /\ closed' = TRUE /\ closeOK' \in BOOLEAN
         /\ failed' = (failed \/ ~closeOK')
         /\ phase' = IF failed' THEN "cleanup" ELSE "rename"
         /\ result' = IF failed' THEN "failure" ELSE result
         /\ UNCHANGED <<kind, mode, requestedMode, tempMode, publishedMode, contents, temp, tempBytes, cleanupFailed, writeOK, syncOK, published, target>>
Rename == /\ phase = "rename" /\ result' \in {"success", "failure"}
          /\ contents' = IF result' = "success" THEN tempBytes ELSE contents
          /\ published' = (result' = "success") /\ publishedMode' = IF published' THEN tempMode ELSE publishedMode
          /\ temp' = ~published' /\ phase' = "cleanup"
          /\ UNCHANGED <<kind, mode, requestedMode, tempMode, failed, closed, tempBytes, cleanupFailed, writeOK, syncOK, closeOK, target>>
Cleanup == /\ phase = "cleanup" /\ cleanupFailed' \in BOOLEAN
           /\ temp' = (temp /\ cleanupFailed')
           /\ phase' = IF result = "success" THEN "done" ELSE "error"
           /\ UNCHANGED <<kind, mode, requestedMode, tempMode, publishedMode, contents, failed, closed, tempBytes, result, writeOK, syncOK, closeOK, published, target>>
Done == /\ phase \in {"done", "error"} /\ UNCHANGED vars
Next == Stat \/ Direct \/ Resolve \/ Create \/ Chmod \/ Write \/ Sync \/ Close \/ Rename \/ Cleanup \/ Done
PreserveMode == (kind = "regular" /\ phase = "done") => tempMode = mode
PublishedAfterSuccess == /\ (published => (writeOK /\ syncOK /\ closeOK /\ closed /\ ~failed /\ tempBytes = "new"))
                         /\ (kind \in {"regular", "missing"} /\ ~published => contents = "old")
================================================
