------------------------------- MODULE Auth -------------------------------
EXTENDS Naturals
CONSTANT Scenario
VARIABLES flow, phase, verified, saved, denied, polls, clock, interval,
          cancelled, envToken, fileToken, source, browser, entryCancelled, meta
vars == <<flow, phase, verified, saved, denied, polls, clock, interval,
          cancelled, envToken, fileToken, source, browser, entryCancelled, meta>>
Init == /\ flow \in (IF Scenario = "device" THEN {"device"} ELSE {"token", "device", "status", "logout"})
        /\ phase \in (IF flow = "token" THEN {"start", "verify"} ELSE {"start"}) /\ verified = FALSE /\ saved = FALSE
        /\ denied = FALSE /\ polls = 0 /\ clock = 0 /\ interval = 1
        /\ entryCancelled = FALSE /\ cancelled = FALSE /\ envToken \in (IF Scenario = "device" THEN {FALSE} ELSE BOOLEAN)
        /\ fileToken \in (IF Scenario = "device" THEN {"absent"} ELSE {"absent", "valid", "invalid", "malformed"})
        /\ source = "none" /\ browser = "notTried"
        /\ meta \in [envURL : {"absent", "environment"}, fileURL : {"absent", "config"},
                     effective : {"unset", "helper"}, scopes : {"empty", "existing"},
                     issued : {"empty"}, result : {"none"}]
        /\ (phase = "verify") = (meta.effective = "helper")
CommandLoad == /\ flow \in {"token", "device"} /\ phase = "start"
               /\ phase' \in (IF fileToken = "malformed" THEN {"failed"} ELSE {"loaded", "failed"})
               /\ UNCHANGED <<flow, verified, saved, denied, polls, clock, interval, cancelled, envToken, fileToken, source, browser, entryCancelled>>
ChooseLogin == /\ phase = "loaded" /\ phase' = IF flow = "token" THEN "verify" ELSE "deviceRequest"
               /\ UNCHANGED <<flow, verified, saved, denied, polls, clock, interval, cancelled, envToken, fileToken, source, browser, entryCancelled>>
TokenVerify == /\ flow = "token" /\ phase = "verify"
               /\ phase' \in {"verified", "rejected"}
               /\ verified' = (phase' = "verified")
               /\ UNCHANGED <<flow, saved, denied, polls, clock, interval, cancelled, envToken, fileToken, source, browser, entryCancelled>>
TokenReload == /\ flow = "token" /\ phase = "verified"
               /\ phase' \in (IF fileToken = "malformed" THEN {"failed"} ELSE {"tokenSave", "failed"})
               /\ UNCHANGED <<flow, verified, saved, denied, polls, clock, interval, cancelled, envToken, fileToken, source, browser, entryCancelled>>
TokenSave == /\ phase = "tokenSave" /\ flow = "token"
             /\ phase' \in {"success", "failed"}
             /\ saved' = (phase' = "success")
             /\ fileToken' = IF saved' THEN "valid" ELSE fileToken
             /\ UNCHANGED <<flow, verified, denied, polls, clock, interval, cancelled, envToken, source, browser, entryCancelled>>
DeviceRequest == /\ flow = "device" /\ phase = "deviceRequest"
                 /\ phase' \in {"display", "failed"}
                 /\ UNCHANGED <<flow, verified, saved, denied, polls, clock, interval, cancelled, envToken, fileToken, source, browser, entryCancelled>>
Display == /\ phase = "display"
           /\ browser' \in (IF Scenario = "device" THEN {"failed"} ELSE {"skipped", "started", "failed"})
           /\ phase' = "entry"
           /\ UNCHANGED <<flow, verified, saved, denied, polls, clock, interval, cancelled, envToken, fileToken, source, entryCancelled>>
Entry == /\ phase = "entry"
         /\ phase' = IF clock > 6 THEN "expired" ELSE IF cancelled THEN "cancelled" ELSE "request"
         /\ entryCancelled' = cancelled
         /\ UNCHANGED <<flow, verified, saved, denied, polls, clock, interval, cancelled, envToken, fileToken, source, browser>>
DevicePoll == /\ phase = "request"
              /\ phase' \in (IF polls < (IF Scenario = "device" THEN 1 ELSE 2) THEN {"pending", "slow", "issued", "issuedEmpty", "denied", "failed", "expired", "cancelled"} ELSE {"issued", "issuedEmpty", "denied", "failed", "expired", "cancelled"})
              /\ denied' = (phase' = "denied")
              /\ polls' = IF polls < 2 THEN polls + 1 ELSE polls
              /\ clock' \in {clock, 7}
              /\ UNCHANGED <<flow, verified, saved, interval, cancelled, envToken, fileToken, source, browser, entryCancelled>>
Backoff == /\ phase \in {"pending", "slow"}
           /\ interval' = IF phase = "slow" THEN interval + 5 ELSE interval
           /\ phase' = "wait"
           /\ UNCHANGED <<flow, verified, saved, denied, polls, clock, cancelled, envToken, fileToken, source, browser, entryCancelled>>
Wait == /\ phase = "wait"
        /\ phase' \in {"entry", "cancelled"}
        /\ clock' = IF phase' = "entry" THEN IF clock + interval > 6 THEN 7 ELSE clock + interval ELSE clock
        /\ cancelled' = IF phase' = "cancelled" THEN TRUE ELSE cancelled
        /\ UNCHANGED <<flow, verified, saved, denied, polls, interval, envToken, fileToken, source, browser, entryCancelled>>
Cancel == /\ phase \in {"entry", "request", "wait"} /\ ~cancelled
          /\ cancelled' = TRUE
          /\ UNCHANGED <<flow, phase, verified, saved, denied, polls, clock, interval, envToken, fileToken, source, browser, entryCancelled>>
DeviceSave == /\ phase \in {"issued", "issuedEmpty"} /\ phase' \in {"saved", "failed"}
              /\ saved' = (phase' = "saved")
              /\ fileToken' = IF saved' THEN IF phase = "issuedEmpty" THEN "absent" ELSE "valid" ELSE fileToken
              /\ UNCHANGED <<flow, verified, denied, polls, clock, interval, cancelled, envToken, source, browser, entryCancelled>>
Profile == /\ flow = "device" /\ phase = "saved"
           /\ phase' \in {"success", "successWithoutProfile"}
           /\ UNCHANGED <<flow, verified, saved, denied, polls, clock, interval, cancelled, envToken, fileToken, source, browser, entryCancelled>>
StatusLoad == /\ phase = "start" /\ flow = "status"
              /\ phase' \in (IF fileToken = "malformed" THEN {"failed"} ELSE {"resolve", "failed"})
              /\ UNCHANGED <<flow, verified, saved, denied, polls, clock, interval, cancelled, envToken, fileToken, source, browser, entryCancelled>>
Resolve == /\ phase = "resolve"
           /\ source' = IF envToken THEN "environment" ELSE IF fileToken \in {"valid", "invalid"} THEN "config" ELSE "none"
           /\ phase' = IF source' = "none" THEN "unauthenticated" ELSE "lookup"
           /\ UNCHANGED <<flow, verified, saved, denied, polls, clock, interval, cancelled, envToken, fileToken, browser, entryCancelled>>
StatusLookup == /\ phase = "lookup" /\ phase' \in {"authenticated", "statusError"}
                /\ UNCHANGED <<flow, verified, saved, denied, polls, clock, interval, cancelled, envToken, fileToken, source, browser, entryCancelled>>
Logout == /\ flow = "logout" /\ phase = "start"
          /\ phase' \in {"loggedOut", "failed"}
          /\ fileToken' = IF phase' = "loggedOut" THEN "absent" ELSE fileToken
          /\ UNCHANGED <<flow, verified, saved, denied, polls, clock, interval, cancelled, envToken, source, browser, entryCancelled>>
AfterLogin == /\ Scenario # "device" /\ flow \in {"token", "device"} /\ phase \in {"success", "successWithoutProfile"}
              /\ flow' = "status" /\ phase' = "start"
              /\ UNCHANGED <<verified, saved, denied, polls, clock, interval, cancelled, envToken, fileToken, source, browser, entryCancelled>>
AfterLogout == /\ phase = "loggedOut" /\ flow' = "status" /\ phase' = "start"
               /\ UNCHANGED <<verified, saved, denied, polls, clock, interval, cancelled, envToken, fileToken, source, browser, entryCancelled>>
End == /\ phase \in {"rejected", "failed", "denied", "expired", "cancelled", "success", "successWithoutProfile", "unauthenticated", "authenticated", "statusError"}
       /\ UNCHANGED vars
ResolvedURL(m) == IF m.envURL # "absent" THEN m.envURL ELSE IF m.fileURL # "absent" THEN m.fileURL ELSE "default"
UpdateMeta == IF phase = "start" /\ phase' = "loaded"
              THEN meta' = [meta EXCEPT !.effective = ResolvedURL(meta)]
           ELSE IF phase = "request" /\ phase' \in {"issued", "issuedEmpty"}
              THEN meta' \in {[meta EXCEPT !.issued = s] : s \in {"empty", "device"}}
           ELSE IF phase = "tokenSave" /\ phase' = "success"
              THEN meta' = [meta EXCEPT !.fileURL = meta.effective,
                                         !.scopes = IF meta.scopes = "empty" THEN "default" ELSE meta.scopes,
                                         !.result = IF meta.scopes = "empty" THEN "default" ELSE meta.scopes]
           ELSE IF phase \in {"issued", "issuedEmpty"} /\ phase' = "saved"
              THEN meta' = [meta EXCEPT !.fileURL = meta.effective, !.scopes = meta.issued]
           ELSE IF phase = "saved" /\ phase' \in {"success", "successWithoutProfile"}
              THEN meta' = [meta EXCEPT !.result = meta.scopes]
           ELSE IF phase = "lookup" /\ phase' = "authenticated"
              THEN meta' = [meta EXCEPT !.result = IF meta.scopes = "empty" THEN "default" ELSE meta.scopes]
           ELSE IF phase = "start" /\ flow = "status" /\ phase' = "resolve"
              THEN meta' = [meta EXCEPT !.effective = ResolvedURL(meta), !.result = "none"]
           ELSE IF phase = "start" /\ flow = "logout" /\ phase' = "loggedOut"
              THEN meta' = [meta EXCEPT !.fileURL = "absent", !.scopes = "empty", !.result = "none"]
           ELSE meta' = meta
LoadCommand == CommandLoad /\ UpdateMeta
ChooseFlow == ChooseLogin /\ UpdateMeta
VerifyToken == TokenVerify /\ UpdateMeta
ReloadTokenConfig == TokenReload /\ UpdateMeta
SaveTokenConfig == TokenSave /\ UpdateMeta
GetDeviceCode == DeviceRequest /\ UpdateMeta
ShowDeviceInstructions == Display /\ UpdateMeta
BeginPoll == Entry /\ UpdateMeta
ReadPollResponse == DevicePoll /\ UpdateMeta
UpdatePollWait == Backoff /\ UpdateMeta
WaitForPoll == Wait /\ UpdateMeta
CancelContext == Cancel /\ UpdateMeta
SaveDeviceToken == DeviceSave /\ UpdateMeta
GetProfile == Profile /\ UpdateMeta
LoadStatus == StatusLoad /\ UpdateMeta
ResolveCredentials == Resolve /\ UpdateMeta
GetStatusProfile == StatusLookup /\ UpdateMeta
ClearStoredCredentials == Logout /\ UpdateMeta
InspectAfterLogin == AfterLogin /\ UpdateMeta
InspectAfterLogout == AfterLogout /\ UpdateMeta
Next == LoadCommand \/ ChooseFlow \/ VerifyToken \/ ReloadTokenConfig \/ SaveTokenConfig
        \/ GetDeviceCode \/ ShowDeviceInstructions \/ BeginPoll \/ ReadPollResponse
        \/ UpdatePollWait \/ WaitForPoll \/ CancelContext \/ SaveDeviceToken \/ GetProfile
        \/ LoadStatus \/ ResolveCredentials \/ GetStatusProfile \/ ClearStoredCredentials
        \/ InspectAfterLogin \/ InspectAfterLogout \/ End
ChartInit == Init /\ meta.envURL = "absent" /\ meta.fileURL = "absent" /\ meta.scopes = "empty"
VerifiedBeforeTokenSave == flow = "token" /\ saved => verified
DeniedNotIssued == denied => phase = "denied"
NoRequestAfterEntryCancel == phase = "request" => ~entryCancelled
EnvironmentSelected == phase = "lookup" /\ envToken => source = "environment"
=============================================================================
