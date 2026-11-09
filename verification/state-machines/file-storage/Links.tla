---------------- MODULE Links ----------------
EXTENDS Naturals
CONSTANT Lengths
VARIABLES phase, path, hops, kind, length
vars == <<phase, path, hops, kind, length>>
Init == /\ phase = "inspect" /\ path = "dir/link" /\ hops = 0
        /\ kind \in {"relative", "absolute", "mixed", "cycle", "missing", "nonlink", "ioerror", "readerror"}
        /\ length \in Lengths
Inspect == /\ phase = "inspect"
           /\ IF kind = "ioerror" THEN phase' = "error"
              ELSE IF kind \in {"missing", "nonlink"} \/ (kind \in {"relative", "absolute", "mixed"} /\ hops = length)
                   THEN phase' \in {"resolved", "error"} ELSE phase' \in {"readlink", "error"}
           /\ UNCHANGED <<path, hops, kind, length>>
ReadLink == /\ phase = "readlink" /\ kind # "readerror"
            /\ IF kind = "mixed" THEN path' \in {"dir/target", "/target"}
               ELSE path' = CASE kind = "relative" -> "dir/target"
                         [] kind = "absolute" -> "/target"
                         [] OTHER -> path
            /\ hops' = hops + 1
            /\ phase' = IF hops' = 40 THEN "exhausted" ELSE "inspect"
            /\ UNCHANGED <<kind, length>>
ReadError == /\ phase = "readlink" /\ phase' = "error"
             /\ UNCHANGED <<path, hops, kind, length>>
Done == /\ phase \in {"resolved", "error", "exhausted"} /\ UNCHANGED vars
Next == Inspect \/ ReadLink \/ ReadError \/ Done
RelativeTarget == (kind = "relative" /\ phase = "resolved") => path = "dir/target"
================================================
