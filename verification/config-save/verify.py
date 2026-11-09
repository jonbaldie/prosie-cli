#!/usr/bin/env python3
"""Check the config save model and its real-code failure cases."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile


HERE = Path(__file__).resolve().parent
ROOT = HERE.parent.parent
TOOL_URL = "https://github.com/tlaplus/tlaplus/releases/download/v1.7.4/tla2tools.jar"
TOOL_SHA256 = "936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88"
GO_VERSION = "go1.26.3"
GO_TEST = ["go", "test", "./internal/config", "-run", "^TestSaveVerificationFailureMatrix$", "-count=1", "-v"]


def run(command, *, cwd=ROOT, env=None, failure=None, success=None):
    result = subprocess.run(command, cwd=cwd, env=env, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    print(result.stdout, end="", flush=True)
    if failure is None:
        if result.returncode:
            raise SystemExit(result.returncode)
        if success and success not in result.stdout:
            raise SystemExit("The check did not report its expected success: " + success)
    elif result.returncode == 0 or failure not in result.stdout:
        raise SystemExit("The scratch defect did not produce the expected failure: " + failure)


def checker():
    cache = Path(os.environ.get("XDG_CACHE_HOME", str(Path.home() / ".cache"))) / "prosie-verification"
    cache.mkdir(parents=True, exist_ok=True)
    jar = cache / "tla2tools-1.7.4.jar"
    if not jar.exists():
        with tempfile.TemporaryDirectory(dir=cache) as scratch:
            download = Path(scratch) / jar.name
            run(["curl", "--fail", "--location", "--retry", "2", TOOL_URL, "--output", str(download)])
            if hashlib.sha256(download.read_bytes()).hexdigest() != TOOL_SHA256:
                raise SystemExit("TLC download checksum mismatch")
            os.replace(download, jar)
    if hashlib.sha256(jar.read_bytes()).hexdigest() != TOOL_SHA256:
        raise SystemExit("Cached TLC checksum mismatch. Remove " + str(jar) + " and rerun.")
    return jar


def model(jar, spec, config, scratch, failure=None):
    run(["java", "-XX:+UseParallelGC", "-cp", str(jar), "tlc2.TLC", "-workers", "1", "-metadir", str(scratch), "-config", str(config), str(spec)], failure=failure, success="Model checking completed. No error has been found.")


def replace_once(source, old, new):
    if source.count(old) != 1:
        raise SystemExit("The mutation no longer matches the covered code. Review and update the self-test.")
    return source.replace(old, new, 1)


def self_test(jar, env):
    source_path = ROOT / "internal/config/config.go"
    source = source_path.read_text()
    spec = (HERE / "ConfigSave.tla").read_text()
    mutants = [
        (
            "PublishedWhole",
            "\t_, writeErr := tmp.Write(data)",
            "\tif err := rename(tmp.Name(), path); err != nil { return err }\n\t_, writeErr := tmp.Write(data)",
            'Write(i) ==\n    /\\ pc[i] = "Write"',
            'PublishBeforeWrite(i) ==\n    /\\ pc[i] = "Write"\n    /\\ pc\' = [pc EXCEPT ![i] = "unsafeWrite"]\n    /\\ published\' = i\n    /\\ publishedBy\' = [publishedBy EXCEPT ![i] = TRUE]\n    /\\ UNCHANGED <<content, private, synced, closed, failed>>\n\nWrite(i) ==\n    /\\ pc[i] = "unsafeWrite"',
            "    \\/ (\\E i \\in Writers : Write(i))",
            "    \\/ (\\E i \\in Writers : PublishBeforeWrite(i))\n    \\/ (\\E i \\in Writers : Write(i))",
        ),
        (
            "PrivateCredentials",
            "return os.CreateTemp(dir, pattern)",
            "file, err := os.CreateTemp(dir, pattern); if err == nil { err = file.Chmod(0644) }; return file, err",
            "          /\\ private' = [private EXCEPT ![i] = TRUE]",
            "          /\\ private' = [private EXCEPT ![i] = FALSE]",
            None,
            None,
        ),
        (
            "PublishedAfterSyncAndClose",
            "\t\twriteErr = tmp.Sync()",
            "\t\twriteErr = nil",
            "synced' = [synced EXCEPT ![i] = TRUE]",
            "synced' = [synced EXCEPT ![i] = FALSE]",
            None,
            None,
        ),
        (
            "FailedSaveDoesNotPublish",
            "\tif err := errors.Join(writeErr, closeErr); err != nil {\n\t\treturn err\n\t}",
            "\t_ = errors.Join(writeErr, closeErr)",
            "    /\\ pc' = [pc EXCEPT ![i] = IF failed[i] THEN \"removeTemp\" ELSE \"rename\"]",
            "    /\\ pc' = [pc EXCEPT ![i] = \"rename\"]",
            None,
            None,
        ),
    ]
    for name, old_go, new_go, old_tla, new_tla, extra_old, extra_new in mutants:
        print("Scratch defect for " + name, flush=True)
        with tempfile.TemporaryDirectory(prefix="prosie-save-mutant-") as directory:
            scratch = Path(directory)
            go_file = scratch / "config.go"
            go_file.write_text(replace_once(source, old_go, new_go))
            overlay = scratch / "overlay.json"
            overlay.write_text(json.dumps({"Replace": {str(source_path): str(go_file)}}))
            run(GO_TEST[:2] + ["-overlay", str(overlay)] + GO_TEST[2:], env=env, failure="--- FAIL: TestSaveVerificationFailureMatrix")
            mutated = replace_once(spec, old_tla, new_tla)
            if extra_old:
                mutated = replace_once(mutated, extra_old, extra_new)
            tla_file = scratch / "ConfigSave.tla"
            tla_file.write_text(mutated)
            config = scratch / "ConfigSave.cfg"
            config.write_text("CONSTANT Writers = {1, 2}\nINIT Init\nNEXT Next\nINVARIANT " + name + "\n")
            model(jar, tla_file, config, scratch / "states", failure="Invariant " + name + " is violated")
        print("Caught real-code and model defects for " + name, flush=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--self-test", action="store_true", help="also check isolated broken Go source and model variants")
    args = parser.parse_args()
    if args.self_test and os.name != "posix":
        raise SystemExit("The permission mutation requires a POSIX system. Run the self-test on Linux or macOS.")
    env = dict(os.environ, GOTOOLCHAIN=GO_VERSION)
    run(["go", "version"], env=env, success=GO_VERSION)
    jar = checker()
    run(GO_TEST, env=env, success="--- PASS: TestSaveVerificationFailureMatrix")
    with tempfile.TemporaryDirectory(prefix="prosie-save-tlc-") as directory:
        model(jar, HERE / "ConfigSave.tla", HERE / "ConfigSave.cfg", Path(directory))
    if args.self_test:
        self_test(jar, env)
        run(GO_TEST, env=env, success="--- PASS: TestSaveVerificationFailureMatrix")
        with tempfile.TemporaryDirectory(prefix="prosie-save-restored-") as directory:
            model(jar, HERE / "ConfigSave.tla", HERE / "ConfigSave.cfg", Path(directory))
    print("PASS. Read this TLA+ result with /statechart.", flush=True)


if __name__ == "__main__":
    main()
