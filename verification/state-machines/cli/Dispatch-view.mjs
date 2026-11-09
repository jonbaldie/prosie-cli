export const title = 'Copy output settings and reset each root execution';
export function state(v) {
  if (v.parent !== v.snapshot || (v.phase === 'returned' && v.sibling !== v.snapshot)) return 'BUG: Child output setting changes the parent or sibling';
  const invocation = v.invocation === 1 ? 'First execution' : 'Second execution';
  const mode = v.mode ? 'JSON' : 'text';
  if (v.phase === 'returned') return `${invocation}: ${v.latch ? 'output failed' : ({help:'help shown',unknown:'unknown command','parse-error':'invalid flag','count-error':'invalid positional count',json:'JSON result written',text:'text result written'})[v.outcome]}; parent setting unchanged`;
  return `${invocation}: ${({start:'call the root',dispatch:'choose a command',leaf:'parse command flags',output:'write the result'})[v.phase]}; ${mode} output`;
}
export const step = action => ({Start:'start with fresh execution state',ConsumeJSON:'enable JSON for this command copy',Choose:'choose a command or show help',ScanLeaf:'read a command flag or positional argument',LeafDone:'validate the argument count',Output:'attempt to write the result',Reset:'call the same root again',Done:'finish'})[action] || action;
