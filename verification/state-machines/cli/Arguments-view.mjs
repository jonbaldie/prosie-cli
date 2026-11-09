export const title = 'Retain each argument and consume flag values once';
const sequence = value => Object.keys(value).sort((a, b) => Number(a) - Number(b)).map(key => value[key]);
const tokens = value => `[${sequence(value).join(' ')}]`;
export function state(v) {
  const phase = ({scan:'Scan input',parse:'Parse reordered input',done:'Return'})[v.phase];
  const outcome = ({pending:'not returned',ok:'accepted',help:'show help','count-error':'too many positional arguments','parser-error':'unknown flag','missing-value':'missing flag value'})[v.result];
  return `${phase} ${tokens(v.input)}; next token ${v.i}; flags ${tokens(v.flags)}; positional ${tokens(v.pos)}; missing value ${v.missing ? 'yes' : 'no'}; ${outcome}`;
}
export function step(action, from, to) {
  if (action === 'ScanStep' && from.input[from.i] === '--text' && from.i < sequence(from.input).length && to.i === from.i + 1) return 'leave the flag value to be scanned again';
  return ({ScanStep:'scan a token and consume its value if required',FinishScan:'put flags before positional arguments',ParseStep:'parse flags and validate the positional count',Terminal:'return'})[action] || action;
}
