----------------------------- MODULE ConfigSave -----------------------------
EXTENDS Integers, TLC

CONSTANT Writers
VARIABLES pc, content, private, synced, closed, failed, publishedBy, published
vars == <<pc, content, private, synced, closed, failed, publishedBy, published>>

Init ==
    /\ pc = [i \in Writers |-> "createTemp"]
    /\ content = [i \in Writers |-> "absent"]
    /\ private = [i \in Writers |-> FALSE]
    /\ synced = [i \in Writers |-> FALSE]
    /\ closed = [i \in Writers |-> FALSE]
    /\ failed = [i \in Writers |-> FALSE]
    /\ publishedBy = [i \in Writers |-> FALSE]
    /\ published \in {-1, 0}

CreateTemp(i) ==
    /\ pc[i] = "createTemp"
    /\ \/ /\ pc' = [pc EXCEPT ![i] = "Write"]
          /\ private' = [private EXCEPT ![i] = TRUE]
          /\ UNCHANGED failed
       \/ /\ pc' = [pc EXCEPT ![i] = "failed"]
          /\ failed' = [failed EXCEPT ![i] = TRUE]
          /\ UNCHANGED private
    /\ UNCHANGED <<content, synced, closed, publishedBy, published>>

Write(i) ==
    /\ pc[i] = "Write"
    /\ \/ /\ pc' = [pc EXCEPT ![i] = "Sync"]
          /\ content' = [content EXCEPT ![i] = "complete"]
          /\ UNCHANGED failed
       \/ /\ pc' = [pc EXCEPT ![i] = "Close"]
          /\ content' = [content EXCEPT ![i] = "partial"]
          /\ failed' = [failed EXCEPT ![i] = TRUE]
    /\ UNCHANGED <<private, synced, closed, publishedBy, published>>

Sync(i) ==
    /\ pc[i] = "Sync"
    /\ pc' = [pc EXCEPT ![i] = "Close"]
    /\ \/ /\ synced' = [synced EXCEPT ![i] = TRUE]
          /\ UNCHANGED failed
       \/ /\ failed' = [failed EXCEPT ![i] = TRUE]
          /\ UNCHANGED synced
    /\ UNCHANGED <<content, private, closed, publishedBy, published>>

Close(i) ==
    /\ pc[i] = "Close"
    /\ pc' = [pc EXCEPT ![i] = "checkErrors"]
    /\ closed' = [closed EXCEPT ![i] = TRUE]
    /\ \/ UNCHANGED failed
       \/ failed' = [failed EXCEPT ![i] = TRUE]
    /\ UNCHANGED <<content, private, synced, publishedBy, published>>

CheckErrors(i) ==
    /\ pc[i] = "checkErrors"
    /\ pc' = [pc EXCEPT ![i] = IF failed[i] THEN "removeTemp" ELSE "rename"]
    /\ UNCHANGED <<content, private, synced, closed, failed, publishedBy, published>>

Rename(i) ==
    /\ pc[i] = "rename"
    /\ \/ /\ pc' = [pc EXCEPT ![i] = "saved"]
          /\ published' = i
          /\ publishedBy' = [publishedBy EXCEPT ![i] = TRUE]
          /\ UNCHANGED failed
       \/ /\ pc' = [pc EXCEPT ![i] = "removeTemp"]
          /\ failed' = [failed EXCEPT ![i] = TRUE]
          /\ UNCHANGED <<publishedBy, published>>
    /\ UNCHANGED <<content, private, synced, closed>>

RemoveTemp(i) ==
    /\ pc[i] = "removeTemp"
    /\ pc' = [pc EXCEPT ![i] = "failed"]
    /\ UNCHANGED <<content, private, synced, closed, failed, publishedBy, published>>

Crash(i) ==
    /\ pc[i] \notin {"saved", "failed", "crashed"}
    /\ pc' = [pc EXCEPT ![i] = "crashed"]
    /\ UNCHANGED <<content, private, synced, closed, failed, publishedBy, published>>

Done ==
    /\ \A i \in Writers : pc[i] \in {"saved", "failed", "crashed"}
    /\ UNCHANGED vars

Next ==
    (\E i \in Writers : CreateTemp(i))
    \/ (\E i \in Writers : Write(i))
    \/ (\E i \in Writers : Sync(i))
    \/ (\E i \in Writers : Close(i))
    \/ (\E i \in Writers : CheckErrors(i))
    \/ (\E i \in Writers : Rename(i))
    \/ (\E i \in Writers : RemoveTemp(i))
    \/ (\E i \in Writers : Crash(i))
    \/ Done

PublishedWhole == published \in Writers => content[published] = "complete"
PrivateCredentials == \A i \in Writers : content[i] # "absent" => private[i]
PublishedAfterSyncAndClose == \A i \in Writers : publishedBy[i] => synced[i] /\ closed[i]
FailedSaveDoesNotPublish == \A i \in Writers : failed[i] => ~publishedBy[i]
=============================================================================
