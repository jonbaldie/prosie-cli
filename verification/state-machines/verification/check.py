#!/usr/bin/env python3
"""Exercise the actual config-save verifier in isolated subprocesses."""
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile

BASE_ROOT = Path(__file__).resolve().parents[3]
TARGET = "verification/config-save/verify.py"

CHILD = r'''
import contextlib, hashlib, importlib.util, io, json, os
from pathlib import Path
import sys
sys.dont_write_bytecode = True
spec = importlib.util.spec_from_file_location("covered_verifier", sys.argv[1])
m = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)
mode = sys.argv[2]
cache = Path(os.environ["XDG_CACHE_HOME"]) / "prosie-verification"
jar = cache / "tla2tools-1.7.4.jar"
m.TOOL_SHA256 = hashlib.sha256(b"good").hexdigest()
out = {"outcome": "returned"}
with contextlib.redirect_stdout(io.StringIO()):
    try:
        if mode.startswith("cache"):
            if mode == "cache_good" or mode == "cache_bad":
                cache.mkdir(parents=True)
                jar.write_bytes(b"good" if mode == "cache_good" else b"bad")
            result = m.checker()
            out["returned_jar"] = str(result) == str(jar)
        elif mode == "mutant_missing_marker":
            m.run([sys.executable, "-c", "import sys; sys.exit(1)"], failure="EXPECTED FAILURE")
        else:
            # The child command prints the expected marker but exits successfully.
            m.run([sys.executable, "-c", "print('EXPECTED FAILURE')"], failure="EXPECTED FAILURE")
    except SystemExit as exc:
        out["outcome"] = "rejected"
        out["message"] = str(exc)
out["jar"] = jar.read_bytes().decode() if jar.exists() else None
out["entries"] = sorted(p.name for p in cache.iterdir()) if cache.exists() else []
print(json.dumps(out))
'''


# The real orchestration runs with external Go and Java programs replaced by stubs.
MAIN_CHILD = r'''
import hashlib, importlib.util, os, sys
from pathlib import Path
sys.dont_write_bytecode = True
spec = importlib.util.spec_from_file_location("covered_verifier", sys.argv[1])
m = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)
m.TOOL_SHA256 = hashlib.sha256(b"good").hexdigest()
sys.argv = [sys.argv[1], "--self-test"]
m.main()
'''

STUB = r'''#!/usr/bin/env python3
import json, os, sys
from pathlib import Path
name = Path(sys.argv[0]).name
args = sys.argv[1:]
record = {"tool": name, "args": args}
if "-overlay" in args:
    overlay = Path(args[args.index("-overlay") + 1])
    record["scratch"] = str(overlay.parent)
    record["overlay_exists"] = overlay.exists()
    record["source_exists"] = all(Path(p).exists() for p in json.loads(overlay.read_text())["Replace"].values())
if name == "java":
    config = Path(args[args.index("-config") + 1])
    record["config"] = config.read_text()
    record["scratch"] = str(config.parent)
with open(os.environ["STUB_LOG"], "a") as f:
    f.write(json.dumps(record) + "\n")
if name == "go" and args == ["version"]:
    print("go version go1.26.3 stub")
elif name == "go" and "-overlay" in args:
    print("--- FAIL: TestSaveVerificationFailureMatrix")
    sys.exit(1)
elif name == "go":
    print("--- PASS: TestSaveVerificationFailureMatrix")
elif "INVARIANT PublishedWhole\\n" in record["config"] and "CONSTANT Writers" not in record["config"]:
    raise SystemExit("unexpected config")
elif "prosie-save-mutant-" in record["scratch"]:
    invariant = next(x.split()[1] for x in record["config"].splitlines() if x.startswith("INVARIANT "))
    print("Invariant " + invariant + " is violated")
    sys.exit(1)
else:
    print("Model checking completed. No error has been found.")
'''


def main():
    source = BASE_ROOT / TARGET
    if os.environ.get("PROSIE_VERIFY_TARGET") == TARGET:
        source = Path(os.environ["PROSIE_VERIFY_SOURCE"]).resolve()
    with tempfile.TemporaryDirectory(prefix="prosie-checker-evidence-") as directory:
        root = Path(directory)
        tools = root / "bin"
        tools.mkdir()
        curl = tools / "curl"
        curl.write_text('#!/bin/sh\nwhile [ "$1" != "--output" ]; do shift; done\ncp "$FAKE_DOWNLOAD" "$2"\n')
        curl.chmod(0o755)
        observations = {}
        for mode in ("cache_good", "cache_bad", "cache_download_good", "cache_download_bad", "mutant_success", "mutant_missing_marker"):
            work = root / mode
            work.mkdir()
            fixture = work / "download"
            fixture.write_bytes(b"bad" if mode.endswith("bad") else b"good")
            env = dict(os.environ, XDG_CACHE_HOME=str(work / "cache"), FAKE_DOWNLOAD=str(fixture),
                       PATH=str(tools) + os.pathsep + os.environ.get("PATH", ""), PYTHONDONTWRITEBYTECODE="1")
            result = subprocess.run([sys.executable, "-c", CHILD, str(source), mode], cwd=work,
                                    env=env, text=True, capture_output=True)
            if result.returncode:
                raise RuntimeError("Verifier subprocess setup failed: " + result.stderr + result.stdout)
            observations[mode] = json.loads(result.stdout)
        expected = {
            "cache_good": ("returned", "good", ["tla2tools-1.7.4.jar"]),
            "cache_bad": ("rejected", "bad", ["tla2tools-1.7.4.jar"]),
            "cache_download_good": ("returned", "good", ["tla2tools-1.7.4.jar"]),
            "cache_download_bad": ("rejected", None, []),
            "mutant_success": ("rejected", None, []),
            "mutant_missing_marker": ("rejected", None, []),
        }
        modes = list(expected)
        if "--marker-first" in sys.argv:
            modes.remove("mutant_missing_marker")
            modes.insert(0, "mutant_missing_marker")
        for mode in modes:
            outcome, jar, entries = expected[mode]
            actual = observations[mode]
            if (actual["outcome"], actual["jar"], actual["entries"]) != (outcome, jar, entries):
                print("FAIL_FORMAL_CHECKER: " + mode + " " + repr(actual), flush=True)
                return 1
            if outcome == "returned" and not actual.get("returned_jar"):
                print("FAIL_FORMAL_CHECKER: wrong checker path " + mode, flush=True)
                return 1
        for name in ("go", "java"):
            tool = tools / name
            tool.write_text(STUB)
            tool.chmod(0o755)
        cache = root / "main-cache" / "prosie-verification"
        cache.mkdir(parents=True)
        (cache / "tla2tools-1.7.4.jar").write_bytes(b"good")
        log = root / "calls.jsonl"
        env = dict(os.environ, XDG_CACHE_HOME=str(cache.parent), STUB_LOG=str(log),
                   PATH=str(tools) + os.pathsep + os.environ.get("PATH", ""), PYTHONDONTWRITEBYTECODE="1")
        result = subprocess.run([sys.executable, "-c", MAIN_CHILD, str(source)], cwd=root,
                                env=env, text=True, capture_output=True)
        if result.returncode:
            raise RuntimeError("main orchestration setup failed: " + result.stdout + result.stderr)
        calls = [json.loads(line) for line in log.read_text().splitlines()]
        kinds = [(x["tool"], "version" if x["args"] == ["version"] else
                  "mutant" if ("-overlay" in x["args"] or "prosie-save-mutant-" in x.get("scratch", "")) else "baseline") for x in calls]
        expected_calls = [("go", "version"), ("go", "baseline"), ("java", "baseline")] + [(tool, "mutant") for _ in range(4) for tool in ("go", "java")] + [("go", "baseline"), ("java", "baseline")]
        mutants = [x for x in calls if "-overlay" in x["args"]]
        model_mutants = [x for x in calls if x["tool"] == "java" and "prosie-save-mutant-" in x.get("scratch", "")]
        names = [next(line.split()[1] for line in x["config"].splitlines() if line.startswith("INVARIANT ")) for x in model_mutants]
        if (kinds != expected_calls or names != ["PublishedWhole", "PrivateCredentials", "PublishedAfterSyncAndClose", "FailedSaveDoesNotPublish"] or
            not all(x["overlay_exists"] and x["source_exists"] and not Path(x["scratch"]).exists() for x in mutants) or
            "PASS. Read this TLA+ result" not in result.stdout):
            print("FAIL_FORMAL_CHECKER: main orchestration " + repr(calls), flush=True)
            return 1
    print("PASS_FORMAL_CHECKER")
    return 0


if __name__ == "__main__":
    sys.exit(main())
