---------------- MODULE Frame ----------------
EXTENDS Naturals, Sequences
CONSTANT MaxLines
VARIABLES phase, event, data, delivered, count, finalFrame, lineMode, expected
vars == <<phase, event, data, delivered, count, finalFrame, lineMode, expected>>
Init == /\ phase = "read" /\ event = "" /\ data = <<>>
        /\ delivered = <<>> /\ count = 0 /\ finalFrame = "none"
        /\ lineMode = "terminated" /\ expected = 0
EventLine == /\ phase = "read" /\ lineMode = "terminated" /\ count < MaxLines
             /\ event' \in {"", "x", "y"} /\ lineMode' \in {"terminated", "partial"} /\ count' = count + 1
             /\ UNCHANGED <<phase, data, delivered, finalFrame, expected>>
DataLine == /\ phase = "read" /\ lineMode = "terminated" /\ count < MaxLines
            /\ \E d \in {"", "a", "b"}: data' = Append(data, d)
            /\ lineMode' \in {"terminated", "partial"} /\ count' = count + 1
            /\ UNCHANGED <<phase, event, delivered, finalFrame, expected>>
OtherLine == /\ phase = "read" /\ lineMode = "terminated" /\ count < MaxLines
             /\ lineMode' \in {"terminated", "partial"} /\ count' = count + 1
             /\ UNCHANGED <<phase, event, data, delivered, finalFrame, expected>>
Separator == /\ phase = "read" /\ lineMode = "terminated" /\ count < MaxLines
             /\ delivered' = Append(delivered, <<event, data>>)
             /\ event' = "" /\ data' = <<>> /\ count' = count + 1
             /\ UNCHANGED <<phase, finalFrame, lineMode, expected>>
End == /\ phase = "read" /\ lineMode = "partial"
       /\ phase' = "eof" /\ finalFrame' = <<event, data>>
       /\ expected' = Len(delivered) + 1
       /\ delivered' = Append(delivered, <<event, data>>)
       /\ event' = "" /\ data' = <<>> /\ UNCHANGED <<count, lineMode>>
EmptyEnd == /\ phase = "read" /\ lineMode = "terminated"
            /\ phase' = "eof" /\ finalFrame' = <<"", <<>>>>
            /\ expected' = Len(delivered) + 2
            /\ delivered' = Append(Append(delivered, <<event, data>>), <<"", <<>>>>)
            /\ event' = "" /\ data' = <<>> /\ UNCHANGED <<count, lineMode>>
BlankEnd == /\ phase = "read" /\ lineMode = "terminated" /\ count < MaxLines
            /\ phase' = "eof" /\ finalFrame' = <<"", <<>>>>
            /\ expected' = Len(delivered) + 2
            /\ delivered' = Append(Append(delivered, <<event, data>>), <<"", <<>>>>)
            /\ event' = "" /\ data' = <<>> /\ count' = count + 1
            /\ lineMode' = "partialBlank"
ReadError == /\ phase = "read" /\ lineMode = "terminated" /\ phase' = "failed"
             /\ UNCHANGED <<event, data, delivered, count, finalFrame, lineMode, expected>>
Terminal == /\ phase # "read" /\ UNCHANGED vars
Next == EventLine \/ DataLine \/ OtherLine \/ Separator \/ End \/ EmptyEnd \/ BlankEnd \/ ReadError \/ Terminal
FlushAtEOF == phase = "eof" => /\ Len(delivered) = expected /\ Len(delivered) > 0
                               /\ delivered[Len(delivered)] = finalFrame
================================================
