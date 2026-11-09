# Credential replacement verification

The check targets `config.Save`. A failed replacement can destroy a user's saved credentials. A permissive replacement can expose the new token to other local users.

## Tool choice

TLC checks every reachable state of a small TLA+ model of the file-operation order. This fits the question better than a proof of JSON encoding or short strings. `verify.py` also runs a generated failure matrix against the real Go implementation, because the model does not execute Go.

Gobra is a Go verifier, but it requires annotated packages and specifications for imported operations. Its [v26.09 library specifications](https://github.com/viperproject/gobra/tree/v26.09/src/main/resources/stubs) do not include `os` or `path/filepath`. A proof would need trusted filesystem specifications anyway. The model makes those assumptions explicit and uses the existing Go file hooks to check the implementation. The [Gobra tutorial](https://github.com/viperproject/gobra/blob/v26.09/docs/tutorial.md) describes package specifications.

Deletion confirmation was the first candidate. Existing `internal/cmd/delete_behavior_test.go` already checks cancellation and HTTP requests across all five delete commands. Credential replacement has a stronger ordering risk and is the selected target.

## Scope and data shape

The model has two writers. Each writer attempts one save, without a retry. Each has a program counter and records for content, permissions, sync, close, failure, and publication. `published` names the file visible at the destination. Its initial value is either absent or an existing complete file. The existing file may have permissive permissions.

The model explores all interleavings of the two saves, success and failure of each file operation, and process crashes between operations. Each writer has a distinct temporary file. There is no fairness or eventual-success claim.

These properties hold in the checked model:

| Property | Meaning |
| --- | --- |
| `PublishedWhole` | A new file at the destination contains the complete new configuration. |
| `PrivateCredentials` | Each new file is private before any token bytes enter it and remains private after publication. |
| `PublishedAfterSyncAndClose` | A published file passed sync and close first. |
| `FailedSaveDoesNotPublish` | A writer that gets an I/O error never publishes its file, even if another writer succeeds. |

The Go test runs all 32 combinations of create, write, sync, close, and rename failure, with both absent and existing destinations. It checks errors, bytes, permissions, temporary-file cleanup, and the actual call order. It uses real files beneath the injected failures. The matrix covers single saves, not the model's concurrent schedules.

## Code mapping and atomic steps

| Model action | Code in `internal/config/config.go` |
| --- | --- |
| `CreateTemp` | `replaceFile` calls `createTemp`, whose default is `os.CreateTemp`. |
| `Write` | `writeAndPublish` calls `tmp.Write(data)`. |
| `Sync` | `writeAndPublish` calls `tmp.Sync()` only after a successful write. |
| `Close` | `writeAndPublish` calls `tmp.Close()` after either write outcome. |
| `CheckErrors` | `errors.Join(writeErr, closeErr)` decides whether rename is allowed. |
| `Rename` | `writeAndPublish` calls `rename(tmp.Name(), path)`. |
| `RemoveTemp` | `replaceFile` attempts `os.Remove(tmp.Name())` on failure. |
| `Crash` | The process stops before its next operation. |

A rename is one atomic publication step, as assumed for the same-directory filesystem operation. Write is not assumed to be atomic. Its abstract completion records either complete bytes or a partial failed write. Intermediate bytes remain in a private, unpublished temporary file. A crash during a write cannot publish those bytes. The model records content history rather than temporary-file existence, so `RemoveTemp` does not erase that history.

## Assumptions and exclusions

- A successful `os.CreateTemp` creates a unique file with mode 0600. The system does not later make that file more permissive.
- A successful `os.File.Write` writes all bytes. A failed write may leave partial bytes.
- A successful sync and close mean those operations succeeded. This is not a proof about disk hardware.
- A successful same-directory rename atomically replaces the destination. A failed rename leaves the destination unchanged.
- Both paths resolve to the same stable destination. Hostile symlink changes, directory replacement, network filesystems with different rename guarantees, and external file mutation are excluded.
- JSON encoding, default path selection, directory creation, and `resolveLinks` are outside the model. They must complete before the modeled replacement starts.
- POSIX permission bits are the security boundary. Windows ACL security is not verified. The Go test omits its mode assertions on Windows.
- Power-loss durability is excluded. `Save` does not sync the directory. A process crash can leave a private temporary file. Cleanup failures are excluded from the model's safety claims.
- The model covers two saves and their orderings, not an arbitrary number of writers. The Go test does not simulate process crashes or prove filesystem guarantees.

## Failure demonstrations

`python3 verification/config-save/verify.py --self-test` checks these isolated defects without changing tracked files. Go overlays compile the broken production source. Temporary model variants express the matching design error.

| Defect | Expected property failure |
| --- | --- |
| Rename the temporary file before writing bytes. | `PublishedWhole`. A reader can see an empty new file. |
| Create a mode-0644 replacement. | `PrivateCredentials`. Another local user can read the new token. |
| Skip sync. | `PublishedAfterSyncAndClose`. Publication occurs without the promised sync. |
| Ignore the joined write and close errors. | `FailedSaveDoesNotPublish`. A failed save replaces the old configuration. |

The runner requires each Go mutant to fail the matrix and each model mutant to violate its named invariant. It then repeats the unchanged checks. A test failure alone is not accepted if the source does not compile. The model and the Go matrix are complementary checks. Neither automatically keeps the other in step when product code changes.

No defect was found in the covered code under these assumptions. The sync commit and update procedure are in `.agents/skills/verify-config-save/SKILL.md`.
