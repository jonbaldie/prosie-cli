"""Check fail-closed behavior in the new verification runner."""

import contextlib
import importlib.util
import io
import os
from pathlib import Path
import sys
import tempfile
import unittest

sys.dont_write_bytecode = True
SPEC = importlib.util.spec_from_file_location("subsystem_verify", Path(__file__).with_name("verify.py"))
VERIFY = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(VERIFY)


class RunnerEvidence(unittest.TestCase):
    def call(self, text, code=0, **kwargs):
        command = [sys.executable, "-c", "import sys; print(" + repr(text) + "); sys.exit(" + str(code) + ")"]
        with contextlib.redirect_stdout(io.StringIO()):
            return VERIFY.checked(command, env=dict(os.environ), **kwargs)

    def test_requires_actual_success_marker(self):
        self.assertIn("PASSED", self.call("PASSED", success="PASSED"))
        with self.assertRaises(RuntimeError):
            self.call("nothing ran", success="PASSED")
        with self.assertRaises(RuntimeError):
            self.call("PASSED", 1, success="PASSED")

    def test_requires_a_running_failure(self):
        self.assertIn("--- FAIL: actual", self.call("--- FAIL: actual", 1, failure="--- FAIL: actual"))
        for text, code in [("--- FAIL: actual", 0), ("some other error", 1),
                           ("--- FAIL: actual\n[build failed]", 1),
                           ("--- FAIL: actual\nsetup failed", 1)]:
            with self.subTest(text=text, code=code):
                with self.assertRaises(RuntimeError):
                    self.call(text, code, failure="--- FAIL: actual")

    def test_requires_unique_source_match(self):
        self.assertEqual("before GOOD after", VERIFY.replace_once("before BAD after", "BAD", "GOOD"))
        for text, old in [("absent", "BAD"), ("BAD BAD", "BAD"), ("anything", "")]:
            with self.subTest(text=text, old=old):
                with self.assertRaises(RuntimeError):
                    VERIFY.replace_once(text, old, "GOOD")

    def test_requires_reachable_states(self):
        VERIFY.require_reachable_states("12 states generated, 8 distinct states found, 0 states left on queue.")
        VERIFY.require_reachable_states("2,000 states generated, 1,000 distinct states found, 0 states left on queue.")
        for output in ["Model checking completed. No error has been found.",
                       "0 states generated, 0 distinct states found, 0 states left on queue."]:
            with self.subTest(output=output):
                with self.assertRaises(RuntimeError):
                    VERIFY.require_reachable_states(output)

    def test_requires_a_source_replay_for_known_findings(self):
        with tempfile.TemporaryDirectory() as directory:
            directory = Path(directory)
            (directory / "Example.tla").write_text("example")
            (directory / "Example.cfg").write_text("INIT Init\nNEXT Next\nINVARIANT Safety\n")
            finding_config = directory / "Finding.cfg"
            finding_config.write_text("INIT Init\nNEXT Next\nINVARIANT Safety\n")
            finding = {"name": "observed", "spec": "Example.tla", "config": "Finding.cfg", "invariant": "Safety",
                       "check": {"command": ["example"], "success": "REPLAYED"}}
            manifest = {"models": [{"spec": "Example.tla", "config": "Example.cfg", "invariants": ["Safety"]}],
                        "checks": [{"command": ["example"], "success": "done"}], "findings": [finding]}
            VERIFY.validate_manifest(directory, manifest)
            finding["check"] = {"command": ["example"]}
            with self.assertRaises(RuntimeError):
                VERIFY.validate_manifest(directory, manifest)
            finding["check"]["success"] = "REPLAYED"
            finding_config.write_text("INIT Init\nNEXT Next\nINVARIANT Safety\nCONSTRAINT HideFailure\n")
            with self.assertRaises(RuntimeError):
                VERIFY.validate_manifest(directory, manifest)
            finding_config.write_text("INIT Init\nNEXT Next\nINVARIANT OtherRule\n")
            with self.assertRaises(RuntimeError):
                VERIFY.validate_manifest(directory, manifest)

    def test_rejects_empty_or_pruned_models(self):
        with tempfile.TemporaryDirectory() as directory:
            directory = Path(directory)
            (directory / "Example.tla").write_text("example")
            config = directory / "Example.cfg"
            manifest = {"models": [{"spec": "Example.tla", "config": "Example.cfg", "invariants": ["Safety"]}],
                        "checks": [{"command": ["example"], "success": "done"}]}
            config.write_text("INIT Init\nNEXT Next\nINVARIANT Safety\n")
            VERIFY.validate_manifest(directory, manifest)
            config.write_text("INIT Init\nNEXT Next\nINVARIANT Safety\nCONSTRAINT HideBadStates\n")
            with self.assertRaises(RuntimeError):
                VERIFY.validate_manifest(directory, manifest)
            config.write_text("INIT Init\nNEXT Next\nINVARIANT AnotherProperty\n")
            with self.assertRaises(RuntimeError):
                VERIFY.validate_manifest(directory, manifest)
            with self.assertRaises(RuntimeError):
                VERIFY.validate_manifest(directory, {"models": []})


if __name__ == "__main__":
    unittest.main()
