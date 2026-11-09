const phases = {
  createTemp: "Previous configuration is unchanged",
  Write: "Private temporary file is ready",
  Sync: "Complete credentials await sync",
  Close: "Temporary file awaits close",
  checkErrors: "Write and close results are checked",
  rename: "Complete private file awaits publication",
  removeTemp: "Failed save awaits cleanup",
  saved: "New credentials are published",
  failed: "Failed save did not publish",
  crashed: "Process stopped before publication",
};

export const title = "One credential save";
export const state = (vars) => phases[vars.pc[1]];
export const step = (action) => ({
  CreateTemp: "Create a private temporary file or return an error",
  Write: "Write the new credentials or record a partial-write error",
  Sync: "Sync the file or record an error",
  Close: "Close the file and retain any error",
  CheckErrors: "Allow publication only if all operations succeeded",
  Rename: "Replace the destination or record a rename error",
  RemoveTemp: "Attempt to remove the temporary file",
  Crash: "The process stops",
  Done: "The save has ended",
})[action];
