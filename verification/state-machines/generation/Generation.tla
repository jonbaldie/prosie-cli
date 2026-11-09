---------------- MODULE Generation ----------------
EXTENDS Naturals
CONSTANT Flow
VARIABLES phase, kind, streamFlag, noStream, json, persistFlag, noPersist,
          words, lines, mode, persist, last, prose, newline, fresh, attempted,
          result, cancelCause, outputFailed
vars == <<phase, kind, streamFlag, noStream, json, persistFlag, noPersist,
          words, lines, mode, persist, last, prose, newline, fresh, attempted,
          result, cancelCause, outputFailed>>
Init == /\ phase = "input" /\ kind = "none" /\ streamFlag = FALSE
        /\ noStream = FALSE /\ json = FALSE /\ persistFlag = FALSE /\ noPersist = FALSE
        /\ words = "omitted" /\ lines = "omitted" /\ mode = "none" /\ persist = FALSE
        /\ last = "none" /\ prose = "none" /\ newline = FALSE
        /\ fresh = FALSE /\ attempted = FALSE /\ result = "none"
        /\ cancelCause = "none" /\ outputFailed = FALSE
Choose == /\ phase = "input"
          /\ kind' \in {"continue", "rewrite", "summarize"}
          /\ streamFlag' \in BOOLEAN /\ noStream' \in BOOLEAN /\ json' \in BOOLEAN
          /\ persistFlag' \in BOOLEAN /\ noPersist' \in BOOLEAN
          /\ words' \in {"omitted", "positive", "nonpositive"}
          /\ lines' \in {"omitted", "positive", "nonpositive"}
          /\ (Flow = "all" \/ (kind' = "continue" /\ streamFlag' /\ ~noStream' /\ ~json'
               /\ persistFlag' /\ ~noPersist' /\ words' = "omitted" /\ lines' = "omitted"))
          /\ (kind' = "continue" \/ (words' = "omitted" /\ lines' = "omitted" /\ ~noStream'))
          /\ (kind' # "rewrite" \/ ~noPersist')
          /\ phase' = "validate"
          /\ UNCHANGED <<mode, persist, last, prose, newline, fresh, attempted, result, cancelCause, outputFailed>>
Validate == /\ phase = "validate"
            /\ IF kind = "continue" /\ (words = "nonpositive" \/ lines = "nonpositive")
                  THEN /\ phase' = "invalid" /\ mode' = "none" /\ persist' = FALSE
                  ELSE /\ phase' = "client"
                       /\ mode' = IF kind = "summarize" THEN "ordinary"
                                   ELSE IF streamFlag /\ ~json /\ (kind # "continue" \/ ~noStream)
                                        THEN "stream" ELSE "ordinary"
                       /\ persist' = (persistFlag /\ (kind = "rewrite" \/ ~noPersist))
            /\ UNCHANGED <<kind, streamFlag, noStream, json, persistFlag, noPersist, words, lines, last, prose, newline, fresh, attempted, result, cancelCause, outputFailed>>
Client == /\ phase = "client" /\ phase' \in {"request", "failed"}
          /\ UNCHANGED <<kind, streamFlag, noStream, json, persistFlag, noPersist, words, lines, mode, persist, last, prose, newline, fresh, attempted, result, cancelCause, outputFailed>>
Token == /\ phase = "request" /\ mode = "stream"
         /\ last' \in {"empty", "text", "line"} /\ outputFailed' \in {outputFailed, TRUE}
         /\ UNCHANGED <<phase, kind, streamFlag, noStream, json, persistFlag, noPersist, words, lines, mode, persist, prose, newline, fresh, attempted, result, cancelCause>>
Success == /\ phase = "request"
           /\ prose' \in (IF mode = "stream" THEN {"nil", "text", "line"} ELSE {"text", "line"})
           /\ result' \in {"persisted", "not-persisted", "conflict"}
           /\ phase' = IF result' = "conflict" THEN "failed" ELSE "success"
           /\ UNCHANGED <<kind, streamFlag, noStream, json, persistFlag, noPersist, words, lines, mode, persist, last, newline, fresh, attempted, cancelCause, outputFailed>>
Finish == /\ phase = "success"
          /\ newline' = (mode = "stream" /\ prose = "text" /\ last = "text")
          /\ phase' = "done" /\ outputFailed' \in {outputFailed, TRUE}
          /\ UNCHANGED <<kind, streamFlag, noStream, json, persistFlag, noPersist, words, lines, mode, persist, last, prose, fresh, attempted, result, cancelCause>>
Failure == /\ phase = "request" /\ kind # "continue" /\ phase' = "failed"
           /\ UNCHANGED <<kind, streamFlag, noStream, json, persistFlag, noPersist, words, lines, mode, persist, last, prose, newline, fresh, attempted, result, cancelCause, outputFailed>>
ContinueError == /\ phase = "request" /\ kind = "continue"
                 /\ cancelCause' \in {"ordinary-error", "error-canceled", "context-done"}
                 /\ phase' = IF cancelCause' = "ordinary-error" THEN "failed" ELSE "cancelling"
                 /\ UNCHANGED <<kind, streamFlag, noStream, json, persistFlag, noPersist, words, lines, mode, persist, last, prose, newline, fresh, attempted, result, outputFailed>>
CancelRequest == /\ phase = "cancelling" /\ fresh' = TRUE /\ attempted' = TRUE /\ phase' = "cancelled"
                 /\ UNCHANGED <<kind, streamFlag, noStream, json, persistFlag, noPersist, words, lines, mode, persist, last, prose, newline, result, cancelCause, outputFailed>>
Done == /\ phase \in {"done", "failed", "cancelled", "invalid"} /\ UNCHANGED vars
Next == Choose \/ Validate \/ Client \/ Token \/ Success \/ Finish \/ Failure \/ ContinueError \/ CancelRequest \/ Done
FreshCancel == attempted => fresh
NoEmptyNewline == (phase = "done" /\ last \in {"none", "empty"}) => ~newline
JSONDisablesStreaming == (kind = "continue" /\ json /\ mode # "none") => mode # "stream"
====================================================
