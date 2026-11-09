----------------------------- MODULE HTTP -----------------------------
EXTENDS Naturals
VARIABLES phase, response, closed, result, secondary
vars == <<phase, response, closed, result, secondary>>
Init == /\ phase = "start" /\ response = FALSE /\ closed = FALSE
        /\ result = "none" /\ secondary = "none"
BuildFailure == /\ phase = "start" /\ phase' = "returned"
                /\ result' = "error" /\ UNCHANGED <<response, closed, secondary>>
Receive == /\ phase = "start" /\ phase' = "owned" /\ response' = TRUE
           /\ UNCHANGED <<closed, result, secondary>>
OrdinaryReturn == /\ phase = "owned" /\ phase' = "returned" /\ closed' = TRUE
                  /\ result' \in {"success", "error"}
                  /\ UNCHANGED <<response, secondary>>
StreamReceive == /\ phase = "start" /\ phase' = "stream-status"
                 /\ UNCHANGED <<response, closed, result, secondary>>
StreamReject == /\ phase = "stream-status" /\ phase' = "returned" /\ closed' = TRUE
                /\ result' = "error" /\ UNCHANGED <<response, secondary>>
StreamAccept == /\ phase = "stream-status" /\ phase' = "stream-owned"
                /\ UNCHANGED <<response, closed, result, secondary>>
StreamClose == /\ phase = "stream-owned" /\ phase' = "returned" /\ closed' = TRUE
               /\ result' \in {"success", "error"} /\ UNCHANGED <<response, secondary>>
PrimarySeries == /\ phase = "start" /\ phase' = "series-books"
                 /\ UNCHANGED <<response, closed, result, secondary>>
SeriesBooks == /\ phase = "series-books" /\ phase' = "series-codex"
               /\ UNCHANGED <<response, closed, result, secondary>>
SeriesCodex == /\ phase = "series-codex" /\ phase' = "returned" /\ result' = "series"
               /\ UNCHANGED <<response, closed, secondary>>
PrimaryBook == /\ phase = "start" /\ phase' = "book"
               /\ UNCHANGED <<response, closed, result, secondary>>
RelationPresent == /\ phase = "book" /\ phase' = "returned" /\ result' = "book"
                   /\ UNCHANGED <<response, closed, secondary>>
Lookup == /\ phase = "book" /\ phase' = "lookup"
          /\ UNCHANGED <<response, closed, result, secondary>>
LookupReturn == /\ phase = "lookup" /\ phase' = "returned"
                /\ secondary' \in {"populated", "unavailable"}
                /\ result' = "book"
                /\ UNCHANGED <<response, closed>>
Done == /\ phase = "returned" /\ UNCHANGED vars
Next == BuildFailure \/ Receive \/ OrdinaryReturn \/ StreamReceive \/ StreamReject \/ StreamAccept \/ StreamClose \/ PrimarySeries \/ SeriesBooks \/ SeriesCodex \/ PrimaryBook \/ RelationPresent \/ Lookup \/ LookupReturn \/ Done
BodyClosed == (phase = "returned" /\ response) => closed
BookSurvives == (phase = "returned" /\ secondary = "unavailable") => result = "book"
=============================================================================
