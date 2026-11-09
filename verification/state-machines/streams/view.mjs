export const title = 'Deliver stream frames and return the highest-priority result';
const flushed = v => {
  const count = Object.keys(v.delivered).length;
  return count === v.expected && count > 0 && JSON.stringify(v.delivered[count]) === JSON.stringify(v.finalFrame);
};
const outcomes = {context:'caller context error',readError:'read error',streamError:'retained stream error',missing:'missing completion error',one:'last completion (payload A)',two:'last completion (payload B)',zero:'last completion (zero value)'};
export function state(v) {
  if ('event' in v) {
    if (v.phase === 'failed') return 'Read failed; pending frame discarded';
    if (v.phase === 'eof') return flushed(v) ? 'Final frame delivered' : 'BUG: Final frame or required dispatch lost';
    return v.count === 0 ? 'Waiting for lines' : 'Reading frame lines';
  }
  const faults = [];
  if (v.seenError && !v.retainedError) faults.push('stream error lost');
  if (v.phase === 'returned') {
    const expected = v.cancelled ? 'context' : v.readOutcome === 'readError' ? 'readError' : v.retainedError ? 'streamError' : v.done === 'none' ? 'missing' : v.done;
    if (v.result !== expected) faults.push(`return ${outcomes[v.result]} instead of ${outcomes[expected]}`);
  }
  if (faults.length) return `BUG: ${faults.join('; ')}`;
  if (v.phase === 'returned') return `Return ${outcomes[v.result]}`;
  if (v.phase === 'decide') return 'Select the highest-priority return value';
  return v.seenError ? 'Reading; stream error retained' : v.done !== 'none' ? 'Reading; last completion saved' : 'Reading events';
}
export function step(action, from, to) {
  if (action === 'End') return 'event' in from ? (flushed(to) ? 'EOF delivers the final frame' : 'EOF omits the required final dispatch') : 'input ends';
  return ({EventLine:'an event name arrives',DataLine:'a data line arrives',OtherLine:'another line arrives',Separator:'a blank line dispatches and resets',EmptyEnd:'empty EOF dispatches twice',BlankEnd:'a blank final line and EOF each dispatch',ReadError:'a read fails',ErrorEvent:'an error arrives',DoneEvent:'a completion arrives',DeltaEvent:'a token arrives',InvalidDone:'ignore an invalid completion',OtherEvent:'another event arrives',Return:'select the return value',Terminal:'finish'})[action] || action;
}
