export const title = 'Import input and deliver an export';
export function state(v) {
  const faults = [];
  if ('owner' in v) {
    if (v.request && !v.inputOK) faults.push('request uses incomplete upload input');
    if (v.phase === 'done' && v.source === 'file' && v.owner !== 'closed') faults.push('upload file remains open');
    if (faults.length) return `BUG: ${faults.join('; ')}`;
    if (v.phase === 'done') return v.result === 'success' ? 'Upload complete; owned input released' : v.result === 'outputError' ? 'Upload response received; output failed' : 'Upload stopped; owned input released';
    if (['fields','part','copy','multipartClose'].includes(v.phase)) return 'Encode the upload input';
  } else {
    if (v.success && v.destination === 'file' && !v.published) faults.push('file export reports success without publication');
    if (v.destination === 'file' && v.phase === 'decodeError') faults.push('raw file bytes rejected by stdout JSON decoding');
    if (faults.length) return `BUG: ${faults.join('; ')}`;
    if (v.phase === 'done') return v.output === 'outputError' ? (v.published ? 'File published; reporting failed' : 'Stdout export failed at output') : v.published ? 'File published and reported' : 'Stdout export complete';
  }
  return ({validate:'Validate input',open:'Open upload input',auth:'Check credentials',build:'Build the upload request',send:'Send the upload request',response:'Wait for the response',read:'Read the response',decode:'Decode the response',cleanup:'Release owned upload input',fetch:'Fetch export bytes',route:'Choose file or stdout',write:'Publish the export file',render:'Render stdout',report:'Report the result',decodeError:'Reject malformed stdout JSON',failed:'Transfer failed'})[v.phase];
}
export const step = action => ({Validate:'validate the input',Open:'open the input',OpenError:'opening fails',Auth:'check credentials',Fields:'write multipart fields',Part:'create the file part',CopyOK:'copy the input',CopyError:'reading the input fails',CloseMultipart:'finish the multipart body',Build:'build the request',Send:'send the request',Response:'receive the response',Read:'read the response',Decode:'decode the response',Cleanup:'release owned input',FetchOK:'receive export bytes',FetchError:'fetching fails',Route:'choose the destination',WriteOK:'publish the file',WriteError:'saving fails',Render:'render stdout',Report:'report the result',Done:'finish'})[action] || action;
