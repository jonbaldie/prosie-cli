#!/usr/bin/env python3
"""Reproduce the suppressed second grep-copy failure in the actual template."""
import os
from pathlib import Path
import subprocess
import tempfile

root = Path(__file__).resolve().parents[3]
wizard = root / '.agents/skills/wizard/template.sh'
with tempfile.TemporaryDirectory(prefix='wizard-copy-fault-') as td:
    d = Path(td)
    tools = d / 'tools'
    tools.mkdir()
    envfile = d / 'saved.env'
    envfile.write_text('STRIPE_PUBLISHABLE_KEY=public-old\nSTRIPE_SECRET_KEY=private-old\n')
    grep = tools / 'grep'
    grep.write_text('''#!/bin/sh
if [ "$1" = "-vE" ]; then
  n=0
  [ ! -f "$COUNT_FILE" ] || n=$(/bin/cat "$COUNT_FILE")
  n=$((n + 1))
  printf '%s' "$n" > "$COUNT_FILE"
  [ "$n" -ne 2 ] || exit 2
fi
exec /usr/bin/grep "$@"
''')
    grep.chmod(0o755)
    for name in ('gh', 'wslview', 'explorer.exe', 'xdg-open', 'open'):
        tool = tools / name
        tool.write_text('#!/bin/sh\nexit 1\n')
        tool.chmod(0o755)
    env = dict(os.environ, PATH=str(tools) + ':/usr/bin:/bin', HOME=str(d),
               ENV_FILE=str(envfile), COUNT_FILE=str(d / 'count'), LC_ALL='C')
    result = subprocess.run(['bash', str(wizard)], input='\n\n\n', text=True,
                            capture_output=True, cwd=d, env=env, timeout=10)
    assert result.returncode == 0, (result.returncode, result.stderr)
    assert (d / 'count').read_text() == '2'
    assert result.stdout.count('[Enter keeps current]') == 2, result.stdout
    assert envfile.read_text() == 'STRIPE_SECRET_KEY=private-old\n', envfile.read_text()
    assert 'Setup complete' in result.stdout
    print('REPLAYED_WIZARD_SUPPRESSED_COPY_LOSS')
