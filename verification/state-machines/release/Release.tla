------------------------------ MODULE Release ------------------------------
EXTENDS Naturals, Sequences, FiniteSets
CONSTANTS Modes, SourceInputs, DistClasses
VARIABLES mode, phase, oldArchive, builds, failed, scratch, archives,
          dist, version, jobs, ciStep
vars == <<mode, phase, oldArchive, builds, failed, scratch, archives, dist, version, jobs, ciStep>>
Init == /\ mode \in Modes
        /\ phase = "start" /\ oldArchive = TRUE /\ builds = 0 /\ failed = FALSE
        /\ scratch = "none" /\ archives = {} /\ dist \in DistClasses
        /\ version = "unset" /\ jobs = [i \in 1..5 |-> "pending"] /\ ciStep = 0
\* Classes retain the consulted inputs; later inputs are irrelevant after a match.
LocalVersion == /\ mode = "local" /\ phase = "start"
 /\ \E input \in SourceInputs:
      version' = [input |-> input,
                  source |-> CASE input = "argument" -> "argument"
                               [] input = "environment" -> "environment"
                               [] input \in {"status-ok-empty-tag", "status-failed-empty-tag"} -> "tag"
                               [] OTHER -> "fallback"]
 /\ phase' = "mkdir" /\ UNCHANGED <<oldArchive, builds, failed, scratch, archives, dist, jobs, ciStep, mode>>
LocalMkdir == /\ mode = "local" /\ phase = "mkdir" /\ phase' = "clean"
 /\ UNCHANGED <<oldArchive, builds, failed, scratch, archives, dist, version, jobs, ciStep, mode>>
LocalClean == /\ mode = "local" /\ phase = "clean" /\ phase' = "scratch" /\ oldArchive' = FALSE
 /\ UNCHANGED <<builds, failed, scratch, archives, dist, version, jobs, ciStep, mode>>
LocalScratch == /\ mode = "local" /\ phase = "scratch" /\ scratch' = "directory" /\ phase' = "build"
 /\ UNCHANGED <<oldArchive, builds, failed, archives, dist, version, jobs, ciStep, mode>>
LocalBuild == /\ mode = "local" /\ phase = "build" /\ builds' = builds + 1 /\ phase' = "license" /\ scratch' = "binary"
 /\ UNCHANGED <<oldArchive, failed, archives, dist, version, jobs, ciStep, mode>>
LocalCopy == /\ mode = "local" /\ phase \in {"license", "readme"}
 /\ phase' = (IF phase = "license" THEN "readme" ELSE "package")
 /\ scratch' = (IF phase = "license" THEN "license" ELSE "readme")
 /\ UNCHANGED <<oldArchive, builds, failed, archives, dist, version, jobs, ciStep, mode>>
LocalPackage == /\ mode = "local" /\ phase = "package" /\ phase' = "remove"
 /\ ~(builds = 5 /\ dist = "relative-missing")
 /\ archives' = archives \cup {IF builds = 5 /\ dist \in {"relative-elsewhere", "relative-outside"} THEN 6 ELSE builds}
 /\ UNCHANGED <<oldArchive, builds, failed, scratch, dist, version, jobs, ciStep, mode>>
LocalRemove == /\ mode = "local" /\ phase = "remove" /\ scratch' = "none"
 /\ phase' = (IF builds = 5 THEN "checksum" ELSE "scratch")
 /\ archives' = (IF dist = "relative-elsewhere" THEN archives \ {6} ELSE archives)
 /\ UNCHANGED <<oldArchive, builds, failed, dist, version, jobs, ciStep, mode>>
LocalChecksum == /\ mode = "local" /\ phase = "checksum" /\ phase' = "done" /\ scratch' = "checksum-complete"
 /\ UNCHANGED <<oldArchive, builds, failed, archives, dist, version, jobs, ciStep, mode>>
LocalFail == /\ mode = "local" /\ phase \in {"mkdir", "clean", "scratch", "build", "license", "readme", "package", "remove", "checksum"}
 /\ phase' = "done" /\ failed' = TRUE
 /\ oldArchive' \in (IF phase = "clean" THEN {TRUE, FALSE} ELSE {oldArchive})
 /\ scratch' \in (CASE phase = "scratch" -> {scratch, "directory"}
                       [] phase = "build" -> {scratch, "binary-partial"}
                       [] phase = "license" -> {scratch, "license-partial"}
                       [] phase = "readme" -> {scratch, "readme-partial"}
                       [] phase = "remove" -> {scratch, "remove-partial", "none"}
                       [] phase = "checksum" -> {scratch, "checksum-partial"}
                       [] OTHER -> {scratch})
 /\ archives' \in (CASE phase = "package" -> {archives, archives \cup {7}}
                       [] phase = "remove" /\ dist = "relative-elsewhere" /\ scratch' = "none" -> {archives \ {6}}
                       [] phase = "remove" /\ dist = "relative-elsewhere" -> {archives, archives \ {6}}
                       [] OTHER -> {archives})
 /\ builds' = (IF phase = "build" THEN builds + 1 ELSE builds)
 /\ UNCHANGED <<dist, version, jobs, ciStep, mode>>
ReleaseStart == /\ mode = "release" /\ phase = "start" /\ phase' = "jobs"
 /\ UNCHANGED <<oldArchive, builds, failed, scratch, archives, dist, version, jobs, ciStep, mode>>
ReleaseJob == /\ mode = "release" /\ phase = "jobs" /\ \E i \in 1..5: (jobs[i] \in {"pending", "checkout", "setup", "build", "upload"} /\ jobs' = [jobs EXCEPT ![i] = IF @ = "pending" THEN "checkout" ELSE IF @ = "checkout" THEN "setup" ELSE IF @ = "setup" THEN "build" ELSE IF @ = "build" THEN "upload" ELSE "success"])
 /\ UNCHANGED <<oldArchive, builds, failed, scratch, archives, dist, version, ciStep, mode, phase>>
ReleaseJobFail == /\ mode = "release" /\ phase = "jobs"
 /\ \E i \in 1..5: (jobs[i] \in {"pending", "checkout", "setup", "build", "upload"} /\ \E outcome \in (CASE jobs[i] = "build" -> {"failed", "failed-build-partial"} [] jobs[i] = "upload" -> {"failed-upload", "failed-upload-partial"} [] OTHER -> {"failed"}): jobs' = [jobs EXCEPT ![i] = outcome])
 /\ UNCHANGED <<oldArchive, builds, failed, scratch, archives, dist, version, ciStep, mode, phase>>
ReleaseCancel == /\ mode = "release" /\ phase = "jobs"
 /\ \E j \in 1..5: jobs[j] \in {"failed", "failed-build-partial", "failed-upload", "failed-upload-partial"}
 /\ \E i \in 1..5: (jobs[i] \in {"pending", "checkout", "setup", "build", "upload"} /\ \E outcome \in (CASE jobs[i] = "build" -> {"cancelled", "cancelled-build-partial"} [] jobs[i] = "upload" -> {"cancelled-upload", "cancelled-upload-partial"} [] OTHER -> {"cancelled"}): jobs' = [jobs EXCEPT ![i] = outcome])
 /\ UNCHANGED <<oldArchive, builds, failed, scratch, archives, dist, version, ciStep, mode, phase>>
ReleaseJoin == /\ mode = "release" /\ phase = "jobs" /\ \A i \in 1..5: jobs[i] \in {"success", "failed", "failed-build-partial", "failed-upload", "failed-upload-partial", "cancelled", "cancelled-build-partial", "cancelled-upload", "cancelled-upload-partial"}
 /\ phase' = (IF (\A i \in 1..5: jobs[i] = "success") THEN "checkout" ELSE "done")
 /\ failed' = (phase' = "done")
 /\ UNCHANGED <<oldArchive, builds, scratch, archives, dist, version, jobs, ciStep, mode>>
ReleaseStep == /\ mode = "release" /\ phase \in {"checkout", "download", "checksum", "publish"}
 /\ phase' = (CASE phase = "checkout" -> "download" [] phase = "download" -> "checksum" [] phase = "checksum" -> "publish" [] OTHER -> "done-published")
 /\ UNCHANGED <<oldArchive, builds, failed, scratch, archives, dist, version, jobs, ciStep, mode>>
ReleaseFail == /\ mode = "release" /\ phase \in {"checkout", "download", "checksum", "publish"}
 /\ phase' \in (CASE phase = "download" -> {"done-download-failed", "done-download-partial"}
                    [] phase = "checksum" -> {"done-checksum-failed", "done-checksum-partial"}
                    [] phase = "publish" -> {"done-publish-failed", "done-publish-partial"}
                    [] OTHER -> {"done-checkout-failed"}) /\ failed' = TRUE
 /\ UNCHANGED <<oldArchive, builds, scratch, archives, dist, version, jobs, ciStep, mode>>
CIStep == /\ mode = "ci" /\ phase = "start" /\ ciStep < 5 /\ ciStep' = ciStep + 1
 /\ phase' = (IF ciStep' = 5 THEN "done" ELSE "start")
 /\ UNCHANGED <<oldArchive, builds, failed, scratch, archives, dist, version, jobs, mode>>
CIFail == /\ mode = "ci" /\ phase = "start" /\ phase' = "done" /\ failed' = TRUE
 /\ UNCHANGED <<oldArchive, builds, scratch, archives, dist, version, jobs, ciStep, mode>>
Done == /\ phase \in {"done", "done-published", "done-checkout-failed", "done-download-failed", "done-download-partial", "done-checksum-failed", "done-checksum-partial", "done-publish-failed", "done-publish-partial"} /\ UNCHANGED vars
Next == LocalVersion \/ LocalMkdir \/ LocalClean \/ LocalScratch \/ LocalBuild \/ LocalCopy \/ LocalPackage \/ LocalRemove \/ LocalChecksum \/ LocalFail \/ ReleaseStart \/ ReleaseJob \/ ReleaseJobFail \/ ReleaseCancel \/ ReleaseJoin \/ ReleaseStep \/ ReleaseFail \/ CIStep \/ CIFail \/ Done
CleanBeforeBuild == mode = "local" /\ builds > 0 => ~oldArchive
JoinBeforePublish == mode = "release" /\ phase \in {"checkout", "download", "checksum", "publish"} => \A i \in 1..5: jobs[i] = "success"
=============================================================================
