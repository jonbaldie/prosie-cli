export const title = 'Validate the checker and reject ineffective defects';
export function state(v) {
  const faults = [];
  if (v.accepted && v.cache !== 'good') faults.push('unverified checker accepted');
  if (v.mutantAccepted && v.testExit !== 'nonzero') faults.push('successful defect test accepted as a catch');
  if (v.mutantAccepted && !v.marker) faults.push('failure without the expected assertion accepted');
  if (faults.length) return `BUG: ${faults.join('; ')}`;
  return ({platform:'Check the platform',version:'Check the Go version',start:'Inspect the checker cache',cacheCheck:'Check cached bytes',verifyDownload:'Check downloaded bytes',replace:'Publish the verified checker',cleanupCache:'Remove download scratch',cleanupReject:'Clean up a failed download',rejected:'Reject the verification run',baselineGo:'Run baseline Go tests',baselineTlc:'Run baseline model checks',mutantSetup:'Prepare an isolated defect',test:'Test the Go defect',modelTest:'Test the model defect',mutantCleanup:'Remove defect scratch',mutantCleanupReject:'Remove rejected defect scratch',restoreGo:'Repeat baseline Go tests',restoreTlc:'Repeat baseline model checks',done:'Verification complete'})[v.phase];
}
export function step(action, from, to) {
  if (action === 'RunMutant') return to.testExit === 'zero' ? 'the defect test exits successfully' : to.marker ? 'the expected defect assertion fails' : 'the defect test fails without its expected assertion';
  if (action === 'VerifyCache') return to.accepted ? 'accept the cached checker [checksum matches]' : 'reject the cached checker [checksum differs]';
  return ({Platform:'check the platform',Version:'check the Go version',ChooseCache:'inspect cached bytes',Download:'download the checker',VerifyDownload:'check the download digest',Replace:'attempt to publish the checker',Cleanup:'remove download scratch',BaselineGo:'run baseline Go tests',BaselineTlc:'run baseline model checks',MutantSetup:'prepare scratch sources',MutantCleanup:'remove defect scratch',RestoreGo:'repeat baseline Go tests',RestoreTlc:'repeat baseline model checks',Terminal:'finish'})[action] || action;
}
