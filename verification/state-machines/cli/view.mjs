export const title = 'Approve a deletion before its request';
export function state(v) {
  const label = {start:'Validate the delete input',validated:'Wait for confirmation',confirmed:'Approval permits client setup',client:'Client is ready',report:'The delete callback returned success',done:'Success was reported',failed:'The operation failed',cancelled:'The user did not approve deletion'}[v.phase] || v.phase;
  return label + (v.invoked && !v.approved ? ' — request without approval' : '') + (v.invoked && v.phase === 'failed' ? ' after the callback' : '');
}
export const step = action => ({ValidateDelete:'parse and validate the input',Confirm:'use --yes or read the first answer',Authenticate:'load credentials and create the client',Delete:'invoke the delete callback',ReportDelete:'report the result',Terminal:'finish'})[action] || action;
