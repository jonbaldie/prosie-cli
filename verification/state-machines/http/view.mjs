export const title = 'Return owned bodies and keep the decoded book';
export function state(v) {
  const faults = [];
  if (v.phase === 'returned' && v.response && !v.closed) faults.push('owned response returned without closure');
  if (v.phase === 'returned' && v.secondary === 'unavailable' && v.result !== 'book') faults.push('primary book lost after a secondary lookup failure');
  if (faults.length) return `BUG: ${faults.join('; ')}`;
  if (v.phase === 'returned') {
    if (v.result === 'book') return v.secondary === 'unavailable' ? 'Book retained without secondary chapters' : 'Book returned with available relations';
    if (v.result === 'series') return 'Series retained after best-effort lookups';
    return v.result === 'error' ? (v.closed ? 'Request failed; body closed' : 'Request failed before body ownership') : 'Request complete; body closed';
  }
  return ({start:'Start an HTTP or relation flow',owned:'Helper owns the response body','stream-status':'Check the stream status','stream-owned':'Caller owns the stream body',book:'Primary book decoded',lookup:'Look up missing chapters','series-books':'Look up missing series books','series-codex':'Look up missing series codex'})[v.phase];
}
export function step(action, from, to) {
  if (action === 'OrdinaryReturn') return to.closed ? 'read the response and close its body' : 'return without closing the owned body';
  if (action === 'LookupReturn') return to.result !== 'book' ? 'discard the book after the secondary lookup fails' : to.secondary === 'unavailable' ? 'the secondary lookup fails; keep the book' : 'read the available chapters';
  return ({BuildFailure:'request setup or transport fails',Receive:'receive an ordinary response',OrdinaryReturn:'read the response and attempt to close it',StreamReceive:'receive a stream response',StreamReject:'reject the stream and close its body',StreamAccept:'give the accepted body to the caller',StreamClose:'the caller finishes and closes the stream',PrimarySeries:'decode the primary series',SeriesBooks:'resolve available books',SeriesCodex:'resolve available codex',PrimaryBook:'decode the primary book',RelationPresent:'retain existing relations',Lookup:'request missing chapters',LookupReturn:'retain the primary book after lookup',Done:'finish'})[action] || action;
}
