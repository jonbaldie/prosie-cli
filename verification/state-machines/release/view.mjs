export const title = 'Release packaging and version selection';
const terminal = {done:'Release stopped', 'done-published':'Release published', 'done-checkout-failed':'Checkout failed', 'done-download-failed':'Download failed', 'done-download-partial':'Download failed; partial files remain', 'done-checksum-failed':'Checksum failed', 'done-checksum-partial':'Checksum failed; partial file remains', 'done-publish-failed':'Publication failed', 'done-publish-partial':'Publication failed; partial remote effects may remain'};
export function state(v) {
  if (!v.mode) {
    if (v.value === 'tag' && v.status !== 'clean') return 'BUG: Release tag accepted without a clean checkout';
    if (v.phase === 'done') return ({linked:'Use the injected build version',fallback:'Use the development version',tag:'Use the verified exact tag'})[v.value];
    return ({start:'Choose the version source',status:'Find a clean checkout',tag:'Read its exact tag'})[v.phase];
  }
  if (v.mode === 'local') {
    if (v.builds > 0 && v.oldArchive) return 'BUG: Build starts before old archives are removed';
    if (v.failed) return 'Local release failed; partial files may remain';
    if (['scratch','build','license','readme','package','remove'].includes(v.phase)) return 'Build and package five targets';
    return ({start:'Choose the release version',mkdir:'Prepare the output folder',clean:'Remove old archives',checksum:'Write checksums',done:'Local release complete'})[v.phase];
  }
  if (v.mode === 'ci') return v.phase === 'done' ? (v.failed ? 'CI failed' : 'CI passed') : 'Run ordered CI checks';
  if (['checkout','download','checksum','publish'].includes(v.phase) && !Object.values(v.jobs).every(job => job === 'success')) return 'BUG: Release job starts before all builds succeed';
  return ({start:'Wait for the release trigger',jobs:'Build and upload five targets',checkout:'Check out the release source',download:'Download all build artifacts',checksum:'Write release checksums',publish:'Publish the release'})[v.phase] || terminal[v.phase];
}
export function step(action, from, to) {
  if (action === 'ReleaseJoin') return to.phase === 'checkout' ? (Object.values(from.jobs).every(job => job === 'success') ? 'continue [all builds succeeded]' : 'continue despite unfinished or failed builds') : 'stop [a build failed or was cancelled]';
  return ({LocalVersion:'select the release version',LocalMkdir:'prepare the output folder',LocalClean:'remove old archives',LocalScratch:'create a scratch folder',LocalBuild:'build a target',LocalCopy:'copy metadata',LocalPackage:'package an archive',LocalRemove:'remove the scratch folder',LocalChecksum:'write checksums',LocalFail:'a local step fails',ReleaseStart:'start five build jobs',ReleaseJob:'advance a build job',ReleaseJobFail:'a build job fails',ReleaseCancel:'cancel a remaining job',ReleaseStep:'advance the release job',ReleaseFail:'a release step fails',CIStep:'run the next CI check',CIFail:'a CI check fails',Start:'choose an injected or fallback version',Status:'check a working tree',TagSuccess:'accept its exact tag',TagFailure:'try the next checkout',Exhausted:'keep the development version',Done:'finish'})[action] || action;
}
