--------------------------- MODULE Checker ---------------------------
EXTENDS Naturals
VARIABLES phase, cache, download, scratch, accepted, selftest, iteration,
          testExit, marker, mutantAccepted, goRestored, tlcRestored
vars == <<phase, cache, download, scratch, accepted, selftest, iteration,
          testExit, marker, mutantAccepted, goRestored, tlcRestored>>
Init == /\ phase = "platform" /\ cache = "missing" /\ download = "none"
        /\ scratch = FALSE /\ accepted = FALSE /\ selftest \in BOOLEAN
        /\ iteration = 1 /\ testExit = "none" /\ marker = FALSE
        /\ mutantAccepted = FALSE /\ goRestored = FALSE /\ tlcRestored = FALSE
Platform == /\ phase = "platform" /\ phase' \in {"version", "rejected"}
            /\ UNCHANGED <<cache, download, scratch, accepted, selftest, iteration, testExit, marker, mutantAccepted, goRestored, tlcRestored>>
Version == /\ phase = "version" /\ phase' \in {"start", "rejected"}
           /\ UNCHANGED <<cache, download, scratch, accepted, selftest, iteration, testExit, marker, mutantAccepted, goRestored, tlcRestored>>
ChooseCache == /\ phase = "start" /\ cache' \in {"missing", "good", "bad"}
               /\ phase' = "cacheCheck"
               /\ UNCHANGED <<download, scratch, accepted, selftest, iteration, testExit, marker, mutantAccepted, goRestored, tlcRestored>>
Download == /\ phase = "cacheCheck" /\ cache = "missing"
            /\ scratch' = TRUE /\ download' \in {"good", "bad", "error"}
            /\ phase' = IF download' = "error" THEN "cleanupReject" ELSE "verifyDownload"
            /\ UNCHANGED <<cache, accepted, selftest, iteration, testExit, marker, mutantAccepted, goRestored, tlcRestored>>
VerifyDownload == /\ phase = "verifyDownload"
                  /\ phase' = IF download = "good" THEN "replace" ELSE "cleanupReject"
                  /\ UNCHANGED <<cache, download, scratch, accepted, selftest, iteration, testExit, marker, mutantAccepted, goRestored, tlcRestored>>
Replace == /\ phase = "replace"
           /\ \/ /\ cache' = download /\ phase' = "cleanupCache"
              \/ /\ cache' = cache /\ phase' = "cleanupReject"
           /\ UNCHANGED <<download, scratch, accepted, selftest, iteration, testExit, marker, mutantAccepted, goRestored, tlcRestored>>
Cleanup == /\ phase \in {"cleanupCache", "cleanupReject"}
           /\ scratch' = FALSE /\ phase' = IF phase = "cleanupCache" THEN "cacheCheck" ELSE "rejected"
           /\ UNCHANGED <<cache, download, accepted, selftest, iteration, testExit, marker, mutantAccepted, goRestored, tlcRestored>>
VerifyCache == /\ phase = "cacheCheck" /\ cache # "missing"
               /\ IF cache = "good" THEN /\ accepted' = TRUE /\ phase' = "baselineGo"
                  ELSE /\ accepted' = FALSE /\ phase' = "rejected"
               /\ UNCHANGED <<cache, download, scratch, selftest, iteration, testExit, marker, mutantAccepted, goRestored, tlcRestored>>
BaselineGo == /\ phase = "baselineGo" /\ phase' \in {"baselineTlc", "rejected"}
              /\ UNCHANGED <<cache, download, scratch, accepted, selftest, iteration, testExit, marker, mutantAccepted, goRestored, tlcRestored>>
BaselineTlc == /\ phase = "baselineTlc" /\ phase' \in {IF selftest THEN "mutantSetup" ELSE "done", "rejected"}
               /\ UNCHANGED <<cache, download, scratch, accepted, selftest, iteration, testExit, marker, mutantAccepted, goRestored, tlcRestored>>
MutantSetup == /\ phase = "mutantSetup" /\ scratch' = TRUE
               /\ phase' \in {"test", "mutantCleanupReject"}
               /\ UNCHANGED <<cache, download, accepted, selftest, iteration, testExit, marker, mutantAccepted, goRestored, tlcRestored>>
RunMutant == /\ phase \in {"test", "modelTest"}
             /\ testExit' \in {"zero", "nonzero"} /\ marker' \in BOOLEAN
             /\ mutantAccepted' = (testExit' = "nonzero" /\ marker')
             /\ phase' = IF mutantAccepted' THEN IF phase = "test" THEN "modelTest" ELSE "mutantCleanup" ELSE "mutantCleanupReject"
             /\ UNCHANGED <<cache, download, scratch, accepted, selftest, iteration, goRestored, tlcRestored>>
MutantCleanup == /\ phase \in {"mutantCleanup", "mutantCleanupReject"}
                 /\ scratch' = FALSE
                 /\ iteration' = IF phase = "mutantCleanup" /\ iteration < 4 THEN iteration + 1 ELSE iteration
                 /\ phase' = IF phase = "mutantCleanupReject" THEN "rejected" ELSE IF iteration < 4 THEN "mutantSetup" ELSE "restoreGo"
                 /\ UNCHANGED <<cache, download, accepted, selftest, testExit, marker, mutantAccepted, goRestored, tlcRestored>>
RestoreGo == /\ phase = "restoreGo" /\ goRestored' \in BOOLEAN
             /\ phase' = IF goRestored' THEN "restoreTlc" ELSE "rejected"
             /\ UNCHANGED <<cache, download, scratch, accepted, selftest, iteration, testExit, marker, mutantAccepted, tlcRestored>>
RestoreTlc == /\ phase = "restoreTlc" /\ tlcRestored' \in BOOLEAN
              /\ phase' = IF tlcRestored' THEN "done" ELSE "rejected"
              /\ UNCHANGED <<cache, download, scratch, accepted, selftest, iteration, testExit, marker, mutantAccepted, goRestored>>
Terminal == /\ phase \in {"rejected", "done"} /\ UNCHANGED vars
Next == Platform \/ Version \/ ChooseCache \/ Download \/ VerifyDownload \/ Replace \/ Cleanup \/ VerifyCache \/ BaselineGo \/ BaselineTlc \/ MutantSetup \/ RunMutant \/ MutantCleanup \/ RestoreGo \/ RestoreTlc \/ Terminal
CacheValid == accepted => cache = "good"
MutantFailed == mutantAccepted => testExit = "nonzero"
MarkerRequired == mutantAccepted => marker
=============================================================================
