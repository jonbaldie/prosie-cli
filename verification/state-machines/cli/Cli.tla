----------------------------- MODULE Cli -----------------------------
EXTENDS Naturals
VARIABLES flow, phase, approved, deleted, invoked, prompt
vars == <<flow, phase, approved, deleted, invoked, prompt>>
Init == /\ flow = "delete" /\ phase = "start"
        /\ approved = FALSE /\ deleted = FALSE /\ invoked = FALSE /\ prompt = "unread"
ValidateDelete == /\ phase = "start" /\ phase' \in {"validated", "failed"}
                  /\ UNCHANGED <<flow, approved, deleted, invoked, prompt>>
Confirm == /\ phase = "validated"
           /\ prompt' \in {"yesFlag", "yes", "no", "eof", "scanError"}
           /\ approved' = (prompt' \in {"yesFlag", "yes"})
           /\ phase' = IF approved' THEN "confirmed" ELSE "cancelled"
           /\ UNCHANGED <<flow, deleted, invoked>>
Authenticate == /\ phase = "confirmed" /\ phase' \in {"client", "failed"}
                /\ UNCHANGED <<flow, approved, deleted, invoked, prompt>>
Delete == /\ flow = "delete" /\ phase = "client" /\ approved
          /\ deleted' \in BOOLEAN /\ invoked' = TRUE
          /\ phase' = IF deleted' THEN "report" ELSE "failed"
          /\ UNCHANGED <<flow, approved, prompt>>
ReportDelete == /\ phase = "report" /\ phase' \in {"done", "failed"}
                /\ UNCHANGED <<flow, approved, deleted, invoked, prompt>>
Terminal == /\ phase \in {"done", "failed", "cancelled"} /\ UNCHANGED vars
Next == ValidateDelete \/ Confirm \/ Authenticate \/ Delete \/ ReportDelete \/ Terminal
DeletionRequiresApproval == invoked => approved
=============================================================================
