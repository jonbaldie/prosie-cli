# Verification checker lifecycle: T04 and T05

`Checker.tla` is a bounded abstraction of `verification/config-save/verify.py`, not a proof of Python, TLC, Go, the OS, or the common harness. The common harness is trusted infrastructure, added after the sync commit.

## Source mapping

- T04 `checker()`: `ChooseCache`, `Download`, `VerifyDownload`, `Replace`, `Cleanup`, `VerifyCache`. A missing cache creates scratch, downloads and checks bytes, replaces the cache and checks again. A corrupt existing cache is rejected without repair. Download and replacement failure clean scratch. `CacheValid` prevents a bad digest being accepted.
- T05 `main()` and `self_test()`: `Platform`, `Version`, `BaselineGo`, `BaselineTlc`, four numbered `MutantSetup`/`RunMutant`/`MutantCleanup` iterations, and `RestoreGo`/`RestoreTlc`. Both Go and model defects require a nonzero exit and expected marker. `MutantFailed` checks exit; `MarkerRequired` checks the marker. A wrong result must not be reported as a caught defect. Non-self-test ends after baseline TLC. Setup/replacement mismatch, subprocess failure and failed restoration reject. Temporary directories are cleaned on the modeled failures.

Subprocess outcomes, platform, cache digest and replacement result are finite classes. SHA-256, subprocesses, file operations and context-manager cleanup are assumptions. The model does not cover concurrent cache changes, interrupted cleanup, filesystem read errors as distinct causes, mutation content, arbitrary Python errors, or S18 itself. The lifecycle beyond the three invariants is represented observationally, not proved by extra invariants. No liveness claim is made.

## Evidence and replay

From the repository root:

```
python3 verification/state-machines/verify.py verification
python3 verification/state-machines/verify.py verification --self-test
```

Read every result with /statechart. `check.py` imports the actual verifier in isolated processes, sets its expected digest for fixture bytes, and stubs `curl`; it checks cache contents and cleanup. It also executes actual `main()` and `self_test()` with external Go and Java executables stubbed. The stubs record commands, overlay files and model configs while scratch exists. The check observes the ordered baseline, four Go/model pairs, restored baseline, expected markers and scratch removal. These stubs do not validate Go or TLC behavior; the common runner separately runs real TLC. No network is used by `check.py`.

The cache mutant removes the cache digest guard. The exit and marker mutants change the `run()` rejection disjunction to a conjunction; separate actual subprocess cases detect a successful command with a marker and a failing command without one. Their model mutants make `RunMutant` accept either condition. Neither mutation changes an invariant. Source overrides are limited to `PROSIE_VERIFY_TARGET=verification/config-save/verify.py` and `PROSIE_VERIFY_SOURCE`. Setup errors do not emit `FAIL_FORMAL_CHECKER`.
