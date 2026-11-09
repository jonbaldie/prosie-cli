export const title = 'Save setup defaults and report complete observations';
export function state(v) {
  if (v.phase === 'done' && v.answers?.public === 'empty' && v.saved.public !== v.defaults.public) return 'BUG: Setup reports completion after losing the saved public default';
  if (!('saved' in v) && v.phase === 'done' && v.input === 'eof') return 'BUG: Incomplete observation reported after input ended';
  const operations = {touch:'Touch the env file',mktemp:'Create a temporary file',copy:'Copy the other saved entries',append:'Append the selected value',mv:'Publish the temporary file',record:'Record the saved key'};
  if (operations[v.phase]) {
    const key = v.key === 'public' ? 'public value' : 'secret value';
    const missing = v.key === 'private' && (['append','mv'].includes(v.phase) ? v.tmp.public !== v.values.public : v.phase === 'record' && v.saved.public !== v.values.public);
    return `${operations[v.phase]} for the ${key}${missing ? '; public entry omitted' : ''}`;
  }
  const names = {start:'Ready to start',paused:'Continue past the introduction',stage:'Stage announced',prompt:'Browser attempted; ask for public input',asked:'Public input received',privatePrompt:'Public value selected; ask for secret',privateAsked:'Secret input received',secret:'Try to set the repository secret',summary:'Ready to report',variableSummary:'Variable attempted',done:'Complete',signin:'Waiting for sign-in',capture:'Waiting for error answer',message:'Waiting for error message',aborted:'Stopped before summary'};
  const lost = 'saved' in v && ['secret','summary','variableSummary'].includes(v.phase) && v.saved.public !== v.values.public;
  return `${names[v.phase]}${lost ? '; public entry omitted' : ''}`;
}
export function step(action, from, to) {
  if (action === 'ChartNext') {
    if (from.phase === 'copy') return to.tmp.public !== from.saved.public && from.key === 'private' ? 'the copy fails without bytes; continue anyway' : 'copy the other saved entries';
    return ({start:'continue past the introduction',paused:'announce the stage',stage:'the browser fails to open',prompt:'accept the public default',asked:'select the public default',privatePrompt:'accept the secret default',privateAsked:'select the secret default',touch:'touch the env file',mktemp:'create a temporary file',append:'append the selected key',mv:'publish the temporary file',record:'record the saved key',secret:'skip the repository secret',summary:'show the completion summary',done:'finish'})[from.phase];
  }
  if (['SignIn', 'FirstCapture', 'LastCapture'].includes(action) && to.phase === 'aborted') return 'input ends; stop';
  if (action === 'Copy') return from.key === 'private' && to.tmp.public !== from.saved.public ? 'the copy fails without bytes; continue anyway' : 'copy the other entries or suppress a copy error';
  return ({Stage:'begin the stage',Open:'try to open the browser',Ask:'enter public input or accept its default',Choose:'select the public value',PrivateAsk:'enter secret input or accept its default',PrivateChoose:'select the secret',Pause:'continue or reach EOF',Touch:'touch the env file',Mktemp:'create a temporary file',Append:'append the current key',Move:'publish the temporary file',Record:'record the saved key',Secret:'try to set the secret',SetVariable:'try to set a variable',Finish:'show the summary',Rerun:'start again',Abort:'an unhandled command fails',SignIn:'finish sign-in or stop',FirstCapture:'answer or stop',LastCapture:'supply the message or stop',Print:'report observations',Done:'finish'})[action];
}
