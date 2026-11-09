export const title = 'Generate text and attempt cancellation';
export function state(v) {
  const faults = [];
  if (v.attempted && !v.fresh) faults.push('cancellation uses the cancelled request context');
  if (v.phase === 'done' && ['none','empty'].includes(v.last) && v.newline) faults.push('newline added after no final token text');
  if (v.kind === 'continue' && v.json && v.mode === 'stream') faults.push('JSON continuation selects streaming');
  if (faults.length) return `BUG: ${faults.join('; ')}`;
  if (v.phase === 'done') return v.outputFailed ? 'Result received; output failed' : v.newline ? 'Output complete; final newline added' : 'Output complete; no newline added';
  return ({input:'Choose generation options',validate:'Validate the input',client:'Prepare the client',request:'Generation active',success:'Result received',failed:'Generation request failed',invalid:'Reject invalid length',cancelling:'Cancellation detected',cancelled:'Cancellation attempted; server outcome unknown'})[v.phase];
}
export const step = action => ({Choose:'choose generation options',Validate:'validate options',Client:'prepare the client',Token:'receive a token',Success:'receive the result',Finish:'finish the output',Failure:'the request fails',ContinueError:'the continuation fails',CancelRequest:'ask the server to cancel',Done:'finish'})[action] || action;
