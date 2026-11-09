export const title = 'Resolve links and publish an export file';
export function state(v) {
  if (!('published' in v)) {
    if (v.phase === 'resolved' && v.kind === 'relative' && v.path !== 'dir/target') return 'BUG: Relative link resolves from the wrong folder';
    return ({inspect:'Inspect the current path',readlink:'Read a symbolic link',resolved:'Target resolved',exhausted:'Stop after 40 link reads',error:'Path resolution failed'})[v.phase];
  }
  const faults = [];
  if (v.kind === 'regular' && v.phase === 'done' && v.tempMode !== v.mode) faults.push('existing permissions lost');
  if (v.published && !(v.writeOK && v.syncOK && v.closeOK && v.closed && !v.failed && v.tempBytes === 'new')) faults.push('file published after an incomplete or failed operation');
  if (['regular','missing'].includes(v.kind) && !v.published && v.contents !== 'old') faults.push('old destination changed before publication');
  if (faults.length) return `BUG: ${faults.join('; ')}`;
  if (v.phase === 'done') return v.kind === 'special' ? 'Special file written in place' : 'Replacement published with the selected permissions';
  if (v.phase === 'error') return v.kind === 'special' ? 'Special-file write failed; bytes may have changed' : `Write failed; old destination retained${v.temp ? '; temporary file remains' : ''}`;
  return ({stat:'Inspect the destination',direct:'Write the special file in place',resolve:'Resolve the destination link',create:'Create a private temporary file',chmod:'Set its selected permissions',write:'Write all temporary bytes',sync:'Sync the temporary bytes',close:'Attempt to close the temporary file',rename:'Publish the replacement',cleanup:'Attempt to remove the temporary file'})[v.phase];
}
export const step = action => ({Inspect:'inspect the path',ReadLink:'follow the link',ReadError:'the link read fails',Stat:'inspect the destination',Direct:'write in place',Resolve:'resolve the target',Create:'create a private temporary file',Chmod:'attempt to set its permissions',Write:'attempt to write all bytes',Sync:'attempt to sync the bytes',Close:'attempt to close the file',Rename:'attempt to publish the file',Cleanup:'attempt to remove the temporary file',Done:'finish'})[action] || action;
