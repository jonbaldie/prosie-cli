---
name: verify-agent-template-flows
description: Verify wizard saved defaults and human reproduction capture when changing executable agent templates or their verification.
---

# Verify agent template flows

Use Python 3, Java 11+, curl and Go on Linux or macOS. Read `verification/state-machines/README.md` for pinned tool versions. First-run downloads can need network access. HTTP fixtures use loopback only.

A wizard rerun must keep a nonempty saved default when the user enters nothing; otherwise it can erase configuration. A reproduction script must stop on EOF before it reports a completed final capture; otherwise missing input looks like an observation.

Coverage at sync commit `3dd1171b29bd4dccc8c4e995438f658f18b818b1`:

- `.agents/skills/wizard/template.sh`: `banner`, `pause`, `stage`, `open_url`, `_existing`, `ask`, `ask_secret`, `write_env`, `set_secret`, `finish` and example stages.
- `.agents/skills/diagnosing-bugs/scripts/hitl-loop.template.sh`: `step`, `capture`, example calls and final output.

Inspect changes with:

```sh
git diff 3dd1171b29bd4dccc8c4e995438f658f18b818b1 -- .agents/skills/wizard/template.sh .agents/skills/diagnosing-bugs/scripts/hitl-loop.template.sh verification/state-machines/agent-templates
```

From the repository root run:

```sh
python3 verification/state-machines/verify.py agent-templates
python3 verification/state-machines/verify.py agent-templates --self-test
```

Read EVERY result with /statechart. These tools add no CLI runtime dependency. The check runs real scripts in scratch with fake inputs and stub external tools. The models assume sequential execution and standard Bash semantics. The wizard baseline assumes grep copies work; the separate fault configuration exposes a reproduced suppressed-copy failure. Unhandled failures can abort with partial effects. The models do not prove filesystem atomicity, remote writes, all inputs or the accuracy of human observations. Use the wizard chart scenario named in the README for its large state graph. See `verification/state-machines/agent-templates/README.md` for mapping and bounds.

On failure, distinguish setup errors from the `FAIL_AGENT_TEMPLATES` observation and inspect the shortest model path. Replay the path against the actual template; a model-only failure can indicate a model error. Run the self-test to confirm that each source defect fails the actual check and its matching model defect violates the named invariant; the final baseline must pass again. If behavior intentionally changes, reconcile the property, model, tests, defects and view with the source. For stale verification use /maintain-formal-verification-skill. Move the sync commit forward only after checking every covered function against the new revision and rerunning both commands.
