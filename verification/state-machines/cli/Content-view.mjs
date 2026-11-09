export const title = 'Choose content without consuming conflicting stdin';
export function state(v) {
  const conflict = v.path !== 'empty' && v.inline !== 'empty' && (v.flow === 'rewrite' || v.filePresent);
  if (conflict && v.consumed) return 'BUG: Stdin consumed despite conflicting content sources';
  if (v.phase === 'done') {
    if (v.result === 'ready' && v.flow === 'chapter') return ({absent:'Chapter patch omits content',empty:'Chapter patch sets empty content',space:'Chapter patch sets whitespace content',text:'Chapter patch sets text content'})[v.patch];
    return ({conflict:'Reject conflicting sources before reading stdin',readError:'Content read failed',ready:'Rewrite input ready',selectionError:'Reject an empty selection',promptError:'Reject an empty instruction and action'})[v.result];
  }
  return ({select:'Choose inline, file or stdin content',read:'Read the selected content source',build:'Build the chapter patch',validate:'Validate rewrite input'})[v.phase];
}
export const step = action => ({Select:'select the content source',Read:'attempt to read the content',Build:'build the patch',Validate:'validate selection and prompt',Done:'finish'})[action] || action;
