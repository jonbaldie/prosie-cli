#!/usr/bin/env python3
"""Check bounded subsystem models and isolated real-source defects."""

import argparse
import importlib.util
import json
import os
from pathlib import Path
import py_compile
import re
import subprocess
import sys
import tempfile

sys.dont_write_bytecode = True
HERE = Path(__file__).resolve().parent
ROOT = HERE.parent.parent
SYNC_COMMIT = "3dd1171b29bd4dccc8c4e995438f658f18b818b1"
GO_VERSION = "go1.26.3"


def checked(command, *, env, success=None, failure=None):
    result = subprocess.run(command, cwd=ROOT, env=env, text=True,
                            stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=300)
    print(result.stdout, end="", flush=True)
    if failure is not None:
        if result.returncode == 0 or failure not in result.stdout:
            raise RuntimeError("The scratch defect did not reach its expected failure: " + failure)
        if "[build failed]" in result.stdout or "setup failed" in result.stdout:
            raise RuntimeError("A build/setup error is not a running-test failure")
    elif result.returncode != 0 or (success and success not in result.stdout):
        raise RuntimeError("The baseline check did not report success: " + " ".join(command))
    return result.stdout


def checker():
    spec = importlib.util.spec_from_file_location("config_save_checker", ROOT / "verification/config-save/verify.py")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module.checker()


def replace_once(text, old, new):
    if not old or text.count(old) != 1:
        raise RuntimeError("The source/model mutation no longer matches exactly once; review its mapping")
    return text.replace(old, new, 1)


def require_reachable_states(output):
    summary = re.search(r"(?m)^[0-9,]+ states generated, ([0-9,]+) distinct states found,", output)
    if summary is None or int(summary[1].replace(",", "")) == 0:
        raise RuntimeError("TLC did not confirm a nonempty reachable state space")


def model(jar, spec, config, env, failure=None):
    with tempfile.TemporaryDirectory(prefix="prosie-subsystem-tlc-") as directory:
        output = checked(["java", "-XX:+UseParallelGC", "-cp", str(jar), "tlc2.TLC",
                          "-workers", "1", "-metadir", directory, "-config", str(config), str(spec)],
                         env=env, success="Model checking completed. No error has been found.", failure=failure)
    if failure is None:
        require_reachable_states(output)
    return output


def baseline(directory, manifest, jar, env):
    for check in manifest.get("checks", []):
        checked(check["command"], env=env, success=check["success"])
    for item in manifest["models"]:
        model(jar, directory / item["spec"], directory / item["config"], env)
    for finding in manifest.get("findings", []):
        check = finding["check"]
        checked(check["command"], env=env, success=check["success"])
        model(jar, directory / finding["spec"], directory / finding["config"], env,
              failure="Invariant " + finding["invariant"] + " is violated")
        print("REPRODUCED known production finding: " + finding["name"], flush=True)


def self_test(directory, manifest, jar, env):
    expected = {(item["spec"], name) for item in manifest["models"] for name in item["invariants"]}
    demonstrated = set()
    for mutant in manifest["mutants"]:
        print("SCRATCH DEFECT: " + manifest["name"] + "/" + mutant["name"], flush=True)
        code = mutant["code"]
        mutation = mutant["model"]
        source_path = ROOT / code["path"]
        model_path = directory / mutation["spec"]
        item = next(item for item in manifest["models"] if item["spec"] == mutation["spec"])
        with tempfile.TemporaryDirectory(prefix="prosie-subsystem-mutant-") as scratch:
            scratch = Path(scratch)
            changed = scratch / source_path.name
            changed.write_text(replace_once(source_path.read_text(), code["old"], code["new"]))
            mutant_env = dict(env, PROSIE_VERIFY_TARGET=code["path"], PROSIE_VERIFY_SOURCE=str(changed))
            check = manifest["checks"][mutant.get("check", 0)]
            command = list(check["command"])
            if source_path.suffix == ".go":
                overlay = scratch / "overlay.json"
                overlay.write_text(json.dumps({"Replace": {str(source_path): str(changed)}}))
                if command[:2] != ["go", "test"]:
                    raise RuntimeError("A Go source defect needs a real Go test")
                command[2:2] = ["-overlay", str(overlay)]
            elif source_path.suffix == ".py":
                py_compile.compile(str(changed), cfile=str(scratch / "syntax.pyc"), doraise=True)
            elif source_path.suffix == ".sh":
                checked(["bash", "-n", str(changed)], env=env)
            checked(command, env=mutant_env, failure=mutant["failure"])
            mutated_spec = scratch / model_path.name
            mutated_spec.write_text(replace_once(model_path.read_text(), mutation["old"], mutation["new"]))
            original_config = (directory / item["config"]).read_text()
            config = scratch / "mutant.cfg"
            lines = [line for line in original_config.splitlines() if not line.strip().startswith("INVARIANT ")]
            config.write_text("\n".join(lines) + "\nINVARIANT " + mutation["invariant"] + "\n")
            model(jar, mutated_spec, config, env,
                  failure="Invariant " + mutation["invariant"] + " is violated")
        demonstrated.add((mutation["spec"], mutation["invariant"]))
        print("CAUGHT real-source and model defects: " + mutant["name"], flush=True)
    if demonstrated != expected:
        raise RuntimeError("Invariant mutation coverage differs: " + repr(expected ^ demonstrated))


def validate_manifest(directory, manifest):
    if not manifest.get("models"):
        raise RuntimeError("A subsystem must have at least one model")
    if "delegate" in manifest:
        return
    if not manifest.get("checks"):
        raise RuntimeError("A model needs production-grounded code checks")
    for item in manifest["models"]:
        if not (directory / item["spec"]).is_file():
            raise RuntimeError("The model file is missing")
        config = (directory / item["config"]).read_text()
        declared = re.findall(r"^INVARIANT\s+([A-Za-z][A-Za-z0-9_]*)\s*$", config, re.M)
        if not declared or sorted(declared) != sorted(item["invariants"]):
            raise RuntimeError("Use one INVARIANT Name line per declared property; manifest/config differ")
        if re.search(r"^\s*(CONSTRAINT|ACTION_CONSTRAINT)\b", config, re.M):
            raise RuntimeError("Baseline models must be bounded in their actions, not by pruning states")
    for finding in manifest.get("findings", []):
        covered = next((item for item in manifest["models"] if item["spec"] == finding["spec"]), None)
        if covered is None or finding["invariant"] not in covered["invariants"]:
            raise RuntimeError("A finding must exercise a declared baseline model rule")
        config = (directory / finding["config"]).read_text()
        declared = re.findall(r"^INVARIANT\s+([A-Za-z][A-Za-z0-9_]*)\s*$", config, re.M)
        if declared != [finding["invariant"]] or re.search(r"^\s*(CONSTRAINT|ACTION_CONSTRAINT)\b", config, re.M):
            raise RuntimeError("A finding config must check its exact rule without pruning states")
        if not finding["check"].get("command") or not finding["check"].get("success"):
            raise RuntimeError("A finding requires a real-source reproduction and success marker")


def run_subsystem(name, jar, env, prove):
    directory = HERE / name
    manifest = json.loads((directory / "manifest.json").read_text())
    if manifest["name"] != name:
        raise RuntimeError("The subsystem manifest name differs from its directory")
    print("SUBSYSTEM: " + name, flush=True)
    if "delegate" in manifest:
        command = manifest["delegate"] + (["--self-test"] if prove else [])
        checked(command, env=env, success=manifest["success"])
    else:
        baseline(directory, manifest, jar, env)
        if prove:
            self_test(directory, manifest, jar, env)
            baseline(directory, manifest, jar, env)
    print("PASS: " + name + " checks complete; known findings are reproduced, not fixed; read each TLA+ result with /statechart", flush=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("subsystem", nargs="?", default="all")
    parser.add_argument("--self-test", action="store_true")
    args = parser.parse_args()
    coverage = json.loads((HERE / "coverage.json").read_text())
    names = sorted(set(coverage.values())) if args.subsystem == "all" else [args.subsystem]
    if not set(names) <= set(coverage.values()):
        parser.error("Unknown subsystem")
    for name in names:
        manifest = json.loads((HERE / name / "manifest.json").read_text())
        expected = {key for key, owner in coverage.items() if owner == name}
        if set(manifest["machines"]) != expected or len(manifest["machines"]) != len(expected):
            raise RuntimeError("Inventory coverage mismatch for " + name)
        validate_manifest(HERE / name, manifest)
    env = dict(os.environ, GOTOOLCHAIN=GO_VERSION)
    env.pop("PROSIE_VERIFY_TARGET", None)
    env.pop("PROSIE_VERIFY_SOURCE", None)
    checked(["go", "version"], env=env, success=GO_VERSION)
    jar = checker()
    for name in names:
        run_subsystem(name, jar, env, args.self_test)
    print("PASS: selected checks complete (known findings remain where reported)", flush=True)


if __name__ == "__main__":
    main()
