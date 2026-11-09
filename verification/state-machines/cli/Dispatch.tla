----------------------------- MODULE Dispatch -----------------------------
EXTENDS Naturals, Sequences, FiniteSets
Prefixes == {<<>>, <<"--json">>, <<"--json", "--json">>}
Tails == {<<>>, <<"help">>, <<"--help">>, <<"-h">>, <<"unknown">>,
          <<"leaf">>, <<"leaf", "x">>, <<"leaf", "--json">>,
          <<"leaf", "--json=false">>, <<"leaf", "x", "--json=false">>,
          <<"leaf", "--", "--json">>, <<"leaf", "--json", "--", "x">>,
          <<"leaf", "--bad">>}
Inputs == {p \o t : p \in Prefixes, t \in Tails} \cup
          {p \o <<"group">> \o q \o t : p \in Prefixes, q \in Prefixes, t \in Tails}
VARIABLES phase, invocation, args, level, parent, snapshot, mode, inherited,
          positionals, boundary, outcome, latch, sibling
vars == <<phase, invocation, args, level, parent, snapshot, mode, inherited,
          positionals, boundary, outcome, latch, sibling>>
First == IF Len(args) = 0 THEN "" ELSE Head(args)
Init == /\ phase = "start" /\ invocation = 1 /\ args = <<>> /\ level = 0
        /\ parent = FALSE /\ snapshot = FALSE /\ mode = FALSE /\ inherited = FALSE
        /\ positionals = <<>> /\ boundary = FALSE /\ outcome = "pending"
        /\ latch = FALSE /\ sibling = FALSE
Start == /\ phase = "start" /\ args' \in Inputs
         /\ phase' = "dispatch" /\ level' = 0
         /\ parent' = FALSE /\ snapshot' = FALSE /\ mode' = FALSE /\ inherited' = FALSE
         /\ positionals' = <<>> /\ boundary' = FALSE /\ outcome' = "pending"
         /\ latch' = FALSE /\ sibling' = FALSE /\ UNCHANGED invocation
ConsumeJSON == /\ phase = "dispatch" /\ Len(args) > 0 /\ Head(args) = "--json"
               /\ mode' = TRUE
               /\ args' = Tail(args)
               /\ UNCHANGED <<phase, invocation, level, parent, snapshot, inherited,
                               positionals, boundary, outcome, latch, sibling>>
Choose == /\ phase = "dispatch" /\ (First # "--json")
          /\ IF Len(args) = 0 \/ First \in {"help", "--help", "-h"}
                THEN /\ phase' = "output" /\ outcome' = "help" /\ UNCHANGED <<args, level, parent, snapshot, mode, inherited>>
                ELSE IF Head(args) = "group" /\ level = 0
                  THEN /\ phase' = "dispatch" /\ args' = Tail(args) /\ level' = 1
                       /\ parent' = mode /\ snapshot' = mode /\ inherited' = mode /\ mode' = mode
                  ELSE IF Head(args) = "leaf"
                    THEN /\ phase' = "leaf" /\ args' = Tail(args)
                         /\ UNCHANGED <<level, parent, snapshot, mode, inherited>>
                    ELSE /\ phase' = "returned" /\ outcome' = "unknown"
                         /\ sibling' = parent
                         /\ UNCHANGED <<args, level, parent, snapshot, mode, inherited>>
          /\ IF phase' # "returned" THEN UNCHANGED sibling ELSE TRUE
          /\ IF phase' # "output" /\ phase' # "returned" THEN UNCHANGED outcome ELSE TRUE
          /\ UNCHANGED <<invocation, positionals, boundary, latch>>
ScanLeaf == /\ phase = "leaf" /\ Len(args) > 0
            /\ args' = Tail(args)
            /\ IF boundary THEN /\ positionals' = Append(positionals, Head(args)) /\ UNCHANGED <<mode, boundary, phase, outcome>>
               ELSE IF Head(args) = "--" THEN /\ boundary' = TRUE /\ UNCHANGED <<positionals, mode, phase, outcome>>
               ELSE IF Head(args) = "--json" THEN /\ mode' = TRUE /\ UNCHANGED <<positionals, boundary, phase, outcome>>
               ELSE IF Head(args) = "--json=false" THEN /\ mode' = FALSE /\ UNCHANGED <<positionals, boundary, phase, outcome>>
               ELSE IF Head(args) = "--bad" THEN /\ phase' = "returned" /\ outcome' = "parse-error" /\ UNCHANGED <<positionals, boundary, mode>>
               ELSE /\ positionals' = Append(positionals, Head(args)) /\ UNCHANGED <<mode, boundary, phase, outcome>>
            /\ sibling' = IF phase' = "returned" THEN parent ELSE sibling
            /\ UNCHANGED <<invocation, level, parent, snapshot, inherited, latch>>
LeafDone == /\ phase = "leaf" /\ Len(args) = 0
            /\ phase' = IF Len(positionals) > 1 THEN "returned" ELSE "output"
            /\ outcome' = IF Len(positionals) > 1 THEN "count-error" ELSE IF mode THEN "json" ELSE "text"
            /\ sibling' = IF phase' = "returned" THEN parent ELSE sibling
            /\ UNCHANGED <<invocation, args, level, parent, snapshot, mode, inherited, positionals, boundary, latch>>
Output == /\ phase = "output" /\ phase' = "returned" /\ latch' \in BOOLEAN
          /\ sibling' = parent
          /\ UNCHANGED <<invocation, args, level, parent, snapshot, mode, inherited, positionals, boundary, outcome>>
Reset == /\ phase = "returned" /\ invocation = 1 /\ phase' = "start" /\ invocation' = 2
         /\ UNCHANGED <<args, level, parent, snapshot, mode, inherited, positionals, boundary, outcome, latch, sibling>>
Done == /\ phase = "returned" /\ invocation = 2 /\ UNCHANGED vars
Next == Start \/ ConsumeJSON \/ Choose \/ ScanLeaf \/ LeafDone \/ Output \/ Reset \/ Done
ParentCopyIsolation == parent = snapshot /\ (phase = "returned" => sibling = snapshot)
=============================================================================
