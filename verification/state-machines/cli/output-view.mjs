export const title = 'The first output failure stops later writer calls';
export function state(v) {
  const counts = `${v.attempt} ${v.attempt === 1 ? 'attempt' : 'attempts'}; ${v.writes} writer ${v.writes === 1 ? 'call' : 'calls'}`;
  if (v.afterFailure) return `BUG: Writer called after failure; ${counts}`;
  const error = ({none:'no failure',short:'first short write retained',error:'first write error retained'})[v.latched];
  return v.phase === 'returned' ? `Exit ${v.result}; ${counts}; ${error}` : `Output active; ${counts}; ${error}`;
}
export function step(action, from, to) {
  if (action === 'Write') {
    if (from.latched !== 'none') return to.afterFailure ? 'call the writer despite its earlier failure' : 'return the retained error without a writer call';
    return ({none:'the write succeeds',short:'retain a short-write error',error:'retain the write error'})[to.latched];
  }
  return action === 'Return' ? (from.handler === 0 && from.latched !== 'none' ? 'change handler success to an output-failure exit' : 'return the handler exit status') : 'finish';
}
