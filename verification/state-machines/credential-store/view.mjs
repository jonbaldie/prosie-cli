export const title = 'Publish whole private credentials after successful operations';
const phases = {createTemp:'previous credentials unchanged',Write:'temporary file ready',unsafeWrite:'destination published before writing',Sync:'complete bytes await sync',Close:'temporary file awaits close',checkErrors:'check accumulated errors',rename:'ready to publish',removeTemp:'failed save awaits cleanup',saved:'new credentials published',failed:'save failed',crashed:'process stopped'};
export function state(v) {
  const writers = Object.keys(v.pc);
  const faults = [];
  if (writers.includes(String(v.published)) && v.content[v.published] !== 'complete') faults.push('incomplete credentials published');
  if (writers.some(i => v.content[i] !== 'absent' && !v.private[i])) faults.push('credential bytes have permissive permissions');
  if (writers.some(i => v.publishedBy[i] && !(v.synced[i] && v.closed[i]))) faults.push('credentials published before sync or close');
  if (writers.some(i => v.failed[i] && v.publishedBy[i])) faults.push('a failed save published credentials');
  const progress = writers.map(i => `Save ${i}: ${phases[v.pc[i]]}`).join('; ');
  return faults.length ? `BUG: ${faults.join('; ')}; ${progress}` : progress;
}
export function step(action, from, to) {
  const writer = Object.keys(from.pc).find(i => from.pc[i] !== to.pc[i]);
  if (!writer) return 'all saves have ended';
  const prefix = `Save ${writer}: `;
  if (action === 'CreateTemp') return prefix + (to.pc[writer] === 'failed' ? 'temporary-file creation fails' : to.private[writer] ? 'create a private temporary file' : 'create a permissive temporary file');
  if (action === 'PublishBeforeWrite') return prefix + 'publish the destination before writing credentials';
  if (action === 'Write') return prefix + (to.content[writer] === 'complete' ? 'write complete credential bytes' : 'write partial bytes and record an error');
  if (action === 'Sync') return prefix + (to.synced[writer] ? 'sync the bytes' : to.failed[writer] ? 'the sync fails' : 'skip sync');
  if (action === 'CheckErrors') return prefix + (to.pc[writer] === 'rename' && to.failed[writer] ? 'ignore the recorded failure and permit publication' : to.pc[writer] === 'rename' ? 'permit publication after the error check' : 'reject publication after a failure');
  return prefix + ({Close:'attempt close and retain any error',Rename:'attempt to replace the destination',RemoveTemp:'attempt to remove the temporary file',Crash:'the process stops',Done:'finish'})[action];
}
