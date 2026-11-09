export const title = 'Keep the new conversation and format its reply';
const modes = {'stream-text':'Live text','stream-json':'JSON with silent token callbacks','ordinary-text':'Complete text reply','ordinary-json':'Complete JSON reply'};
export function state(v) {
  const faults = [];
  if (v.made && (!v.created || v.id !== v.returnedID)) faults.push('new conversation details lost');
  if (v.mode === 'stream-json' && (v.deltaPrinted || Object.values(v.output).some(item => ['text','newline'].includes(item)))) faults.push('token text leaked into JSON output');
  if (faults.length) return `BUG: ${faults.join('; ')}`;
  if (v.phase === 'select') return 'Choose a conversation';
  if (v.phase === 'choose') return v.made ? 'New conversation ready' : 'Existing conversation selected';
  if (v.phase === 'request') return `${modes[v.mode]}: ${v.tokens[1] ? 'tokens received' : 'waiting'}`;
  if (v.phase === 'report') return `${modes[v.mode]}: ${v.outcome === 'success' ? 'response received' : 'send failed'}`;
  return `${v.phase === 'done' ? 'Turn complete' : 'Turn failed'}${v.writeFailed ? '; output failed' : ''}${v.made ? '; new conversation details retained' : ''}`;
}
export const step = action => ({Existing:'reuse the conversation',Create:'create a conversation',CreateFail:'creation fails',Choose:'choose the reply mode',Delta:'receive a token',Success:'receive the response',Failure:'sending fails',Report:'report the result',Finished:'finish'})[action] || action;
