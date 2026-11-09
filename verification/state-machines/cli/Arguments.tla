-------------------------- MODULE Arguments --------------------------
EXTENDS Naturals, Sequences, FiniteSets
Tokens == {"v", "-", "--", "--text", "--text=", "--bool", "--bool=false", "--unknown", "--help"}
Inputs == UNION { [1..n -> Tokens] : n \in 0..3 }
IsFlag(t) == t \in {"--text", "--text=", "--bool", "--bool=false", "--unknown", "--help"}
NeedsValue(t) == t = "--text"
RECURSIVE Scan(_)
Scan(s) ==
    IF Len(s) = 0 THEN [flags |-> <<>>, pos |-> <<>>, missing |-> FALSE]
    ELSE LET t == Head(s) IN
      IF t = "--" THEN [flags |-> <<>>, pos |-> Tail(s), missing |-> FALSE]
      ELSE IF ~IsFlag(t) THEN
        LET r == Scan(Tail(s)) IN [r EXCEPT !.pos = <<t>> \o @]
      ELSE IF NeedsValue(t) /\ Len(s) = 1 THEN
        [flags |-> <<t>>, pos |-> <<>>, missing |-> TRUE]
      ELSE IF NeedsValue(t) THEN
        LET r == Scan(SubSeq(s, 3, Len(s))) IN [r EXCEPT !.flags = <<t, s[2]>> \o @]
      ELSE LET r == Scan(Tail(s)) IN [r EXCEPT !.flags = <<t>> \o @]
VARIABLES input, i, flags, pos, missing, phase, reordered, p, visited, text, bool, result
vars == <<input, i, flags, pos, missing, phase, reordered, p, visited, text, bool, result>>
Init == /\ input \in Inputs /\ i = 1 /\ flags = <<>> /\ pos = <<>>
        /\ missing = FALSE /\ phase = "scan" /\ reordered = <<>> /\ p = 1
        /\ visited = {} /\ text = "default" /\ bool = FALSE /\ result = "pending"
ScanStep ==
  /\ phase = "scan" /\ i <= Len(input)
  /\ LET t == input[i] IN
     (IF t = "--" THEN
       /\ pos' = pos \o SubSeq(input, i+1, Len(input)) /\ i' = Len(input)+1
       /\ UNCHANGED <<flags, missing>>
     ELSE IF ~IsFlag(t) THEN
       /\ pos' = Append(pos, t) /\ i' = i+1 /\ UNCHANGED <<flags, missing>>
     ELSE IF NeedsValue(t) /\ i = Len(input) THEN
       /\ flags' = Append(flags, t) /\ missing' = TRUE /\ i' = i+1 /\ UNCHANGED pos
     ELSE IF NeedsValue(t) THEN
       /\ flags' = flags \o <<t, input[i+1]>> /\ i' = i+2
       /\ UNCHANGED <<pos, missing>>
     ELSE /\ flags' = Append(flags, t) /\ i' = i+1 /\ UNCHANGED <<pos, missing>>)
  /\ UNCHANGED <<input, phase, reordered, p, visited, text, bool, result>>
FinishScan == /\ phase = "scan" /\ i > Len(input)
  /\ reordered' = (IF Len(pos) > 0 /\ ~missing THEN flags \o <<"--">> \o pos ELSE flags)
  /\ phase' = "parse" /\ UNCHANGED <<input, i, flags, pos, missing, p, visited, text, bool, result>>
ParseStep == /\ phase = "parse"
  /\ (IF p > Len(reordered) \/ reordered[p] = "--" THEN
       /\ phase' = "done" /\ result' = (IF Len(pos) > 1 THEN "count-error" ELSE "ok")
       /\ UNCHANGED <<p, visited, text, bool>>
     ELSE LET t == reordered[p] IN
       IF t = "--unknown" THEN
         /\ phase' = "done" /\ result' = "parser-error" /\ UNCHANGED <<p, visited, text, bool>>
       ELSE IF t = "--help" THEN
         /\ phase' = "done" /\ result' = "help" /\ UNCHANGED <<p, visited, text, bool>>
       ELSE IF t = "--text" /\ p = Len(reordered) THEN
         /\ phase' = "done" /\ result' = "missing-value" /\ UNCHANGED <<p, visited, text, bool>>
       ELSE IF t = "--text" THEN
         /\ p' = p+2 /\ visited' = visited \cup {"text"} /\ text' = reordered[p+1]
         /\ UNCHANGED <<phase, result, bool>>
       ELSE IF t = "--text=" THEN
         /\ p' = p+1 /\ visited' = visited \cup {"text"} /\ text' = ""
         /\ UNCHANGED <<phase, result, bool>>
       ELSE
         /\ p' = p+1 /\ visited' = visited \cup {"bool"} /\ bool' = (t = "--bool")
         /\ UNCHANGED <<phase, result, text>>)
  /\ UNCHANGED <<input, i, flags, pos, missing, reordered>>
Terminal == /\ phase = "done" /\ UNCHANGED vars
Next == ScanStep \/ FinishScan \/ ParseStep \/ Terminal
ScannerRetention == phase # "scan" => /\ flags = Scan(input).flags
                                      /\ pos = Scan(input).pos
                                      /\ missing = Scan(input).missing
=============================================================================
