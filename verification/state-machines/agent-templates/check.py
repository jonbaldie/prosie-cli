#!/usr/bin/env python3
"""Run both actual templates with fake inputs and isolated external tools."""
import os
from pathlib import Path
import subprocess
import tempfile

BASE_ROOT = Path(__file__).resolve().parents[3]
WIZARD = '.agents/skills/wizard/template.sh'
HITL = '.agents/skills/diagnosing-bugs/scripts/hitl-loop.template.sh'


def source(path):
    if os.environ.get('PROSIE_VERIFY_TARGET') == path:
        return Path(os.environ['PROSIE_VERIFY_SOURCE']).resolve()
    return BASE_ROOT / path


def require(condition, detail):
    if not condition:
        print('FAIL_AGENT_TEMPLATES: ' + detail, flush=True)
        raise SystemExit(1)


def run(script, data, directory, env):
    return subprocess.run(['bash', str(source(script))], input=data, text=True,
                          cwd=directory, env=env, capture_output=True, timeout=10)


def main():
    with tempfile.TemporaryDirectory(prefix='prosie-agent-templates-') as tmp:
        root = Path(tmp)
        tools = root / 'tools'
        tools.mkdir()
        for name in ('gh', 'wslview', 'explorer.exe', 'xdg-open', 'open'):
            path = tools / name
            path.write_text('#!/bin/sh\nexit 1\n')
            path.chmod(0o755)
        env = {'PATH': str(tools) + ':/usr/bin:/bin', 'HOME': str(root),
               'ENV_FILE': str(root / 'example.env'), 'LC_ALL': 'C'}
        first = run(WIZARD, '\nexample-public\nexample-secret\n', root, env)
        require(first.returncode == 0, 'wizard first run failed')
        require('GitHub secret STRIPE_SECRET_KEY' in first.stdout, 'failed secret not reported')
        require((root / 'example.env').read_text() ==
                'STRIPE_PUBLISHABLE_KEY=example-public\nSTRIPE_SECRET_KEY=example-secret\n',
                'wizard did not persist example values')
        second = run(WIZARD, '\n\n\n', root, env)
        require(second.returncode == 0, 'wizard rerun failed')
        require((root / 'example.env').read_text() ==
                'STRIPE_PUBLISHABLE_KEY=example-public\nSTRIPE_SECRET_KEY=example-secret\n',
                'empty input did not retain saved defaults')
        # Exercise reusable helpers with synthetic stages, not a Prosie integration.
        library = source(WIZARD).read_text().split('# STAGES: author this section.')[0]
        helper = root / 'helpers.sh'
        helper.write_text(library + '\nset_secret FAKE_SECRET fake\nset_var FAKE_VAR fake\nfinish\n')
        skipped = subprocess.run(['bash', str(helper)], cwd=root, env=env,
                                 capture_output=True, text=True, timeout=10)
        require(skipped.returncode == 0 and 'GitHub secret FAKE_SECRET (set it manually:' in skipped.stdout
                and 'GitHub variable FAKE_VAR' in skipped.stdout,
                'helper skip summary missing')
        (tools / 'gh').write_text('#!/bin/sh\nexit 0\n')
        succeeded = subprocess.run(['bash', str(helper)], cwd=root, env=env,
                                   capture_output=True, text=True, timeout=10)
        require(succeeded.returncode == 0 and '1 GitHub secret(s): FAKE_SECRET' in succeeded.stdout
                and 'still to do by hand:' not in succeeded.stdout,
                'helper success summary incorrect')
        for data in ('', '\n'):
            early = run(HITL, data, root, env)
            require(early.returncode != 0 and '--- Captured ---' not in early.stdout,
                    'early EOF did not abort before summary')
        eof_wizard = run(WIZARD, '', root, env)
        require(eof_wizard.returncode == 0 and (root / 'example.env').read_text() ==
                'STRIPE_PUBLISHABLE_KEY=example-public\nSTRIPE_SECRET_KEY=example-secret\n',
                'wizard EOF did not keep defaults')
        bad_env = dict(env, ENV_FILE=str(root / 'missing' / 'example.env'))
        failed_write = run(WIZARD, '\na\nb\n', root, bad_env)
        require(failed_write.returncode != 0 and 'Setup complete' not in failed_write.stdout,
                'write error did not abort wizard')
        complete = run(HITL, '\ny\nexample error\n', root, env)
        require(complete.returncode == 0 and 'ERRORED=y\nERROR_MSG=example error\n' in complete.stdout,
                'completed reproduction did not report observations')
        eof = run(HITL, '\ny\n', root, env)
        require(eof.returncode != 0 and '--- Captured ---' not in eof.stdout,
                'EOF in final capture did not abort before summary')
    print('PASS_AGENT_TEMPLATES')


if __name__ == '__main__':
    main()
