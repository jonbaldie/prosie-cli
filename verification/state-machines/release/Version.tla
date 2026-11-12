------------------------------ MODULE Version ------------------------------
VARIABLES phase, main, value
vars == <<phase, main, value>>
\* main is the class of the main module version that go build embeds.
MainClasses == {"release", "pseudo", "dirty", "devel", "absent"}
Init == /\ phase = "start" /\ value \in {"linked", ""} /\ main = "unread"
Start == /\ phase = "start"
         /\ IF value = "linked" THEN /\ phase' = "done" /\ UNCHANGED value
            ELSE /\ value' = "fallback" /\ phase' = "buildinfo"
         /\ UNCHANGED main
BuildInfo == /\ phase = "buildinfo" /\ main' \in MainClasses
             /\ IF main' = "release" THEN value' = "tag" ELSE UNCHANGED value
             /\ phase' = "done"
Done == /\ phase = "done" /\ UNCHANGED vars
Next == Start \/ BuildInfo \/ Done
CleanTag == value = "tag" => main = "release"
=============================================================================
