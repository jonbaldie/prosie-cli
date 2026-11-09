#!/usr/bin/env python3
import os
from pathlib import Path
import shutil
import subprocess
import tempfile

BASE_ROOT = Path(__file__).resolve().parents[3]

def source(path):
    if os.environ.get('PROSIE_VERIFY_TARGET') == path:
        return Path(os.environ['PROSIE_VERIFY_SOURCE'])
    return BASE_ROOT / path

def require(value, message):
    if not value:
        print('FAIL_RELEASE_CHECK: ' + message, flush=True)
        raise SystemExit(1)

with tempfile.TemporaryDirectory(prefix='prosie-release-check-') as d:
    root = Path(d)
    (root / 'scripts').mkdir()
    (root / 'bin').mkdir()
    (root / 'dist').mkdir()
    (root / 'dist/prosie_old').write_text('old')
    (root / 'dist/checksums.txt').write_text('old')
    shutil.copy(source('scripts/build-release.sh'), root / 'scripts/build-release.sh')
    go = root / 'bin/go'
    go.write_text('#!/bin/bash\nprintf "%s\\n" "$@" > "$ARGS"\nif [ -e "$DIST_DIR/prosie_old" ] || [ -e "$DIST_DIR/checksums.txt" ]; then echo dirty > "$OBSERVED"; else echo clean > "$OBSERVED"; fi\nexit 42\n')
    go.chmod(0o755)
    env = dict(os.environ, PATH=str(root / 'bin') + os.pathsep + os.environ['PATH'], DIST_DIR=str(root / 'dist'), OBSERVED=str(root / 'observed'), ARGS=str(root / 'args'))
    result = subprocess.run(['bash', str(root / 'scripts/build-release.sh'), 'v1.2.3'], cwd=root, env=env, capture_output=True, text=True)
    if not (root / 'observed').exists():
        raise RuntimeError('stub build did not run: ' + result.stderr)
    require((root / 'observed').read_text().strip() == 'clean', 'old release files remain when build begins')
    require(result.returncode == 42, 'failed build did not stop the script')
    require(not (root / 'dist/checksums.txt').exists(), 'failed build left checksums')

    (root / 'internal/version').mkdir(parents=True)
    (root / 'internal/version/version.go').write_text('const fallbackVersion = "9.9.9"\n')
    git = root / 'bin/git'
    git.write_text('#!/bin/bash\ncase "$3" in status) printf "%s" "$STATUS"; exit "$STATUS_EXIT";; describe) printf "%s" "$TAG"; exit "$TAG_EXIT";; esac\n')
    git.chmod(0o755)
    for arg, version, status, status_exit, tag, tag_exit, expected in [
        ('v1.0', 'v2.0', '', '0', 'v3.0', '0', '1.0'),
        ('', 'v2.0', '', '0', 'v3.0', '0', '2.0'),
        ('', '', '', '1', 'v3.0', '0', '3.0'),
        ('', '', 'dirty', '0', 'v3.0', '0', '9.9.9'),
        ('', '', '', '0', '', '1', '9.9.9'),
    ]:
        case_env = dict(env, VERSION=version, STATUS=status, STATUS_EXIT=status_exit, TAG=tag, TAG_EXIT=tag_exit)
        result = subprocess.run(['bash', str(root / 'scripts/build-release.sh'), arg], cwd=root, env=case_env, capture_output=True, text=True)
        require(result.returncode == 42, 'version scenario did not reach stub build')
        require('internal/version.Version=' + expected in (root / 'args').read_text(), 'version precedence changed')

release = source('.github/workflows/release.yml').read_text()
ci = source('.github/workflows/ci.yml').read_text()
require('\n  release:\n' in release, 'release job missing')
release_job = release.split('\n  release:\n', 1)[1]
require('    needs: build\n' in release_job and "      - 'v*'" in release, 'release trigger or join changed')
require(release.index('Download all built artifacts') < release.index('Generate Checksums') < release.index('Create Release'), 'release order changed')
require(ci.index('Run tests') < ci.index('Verify credential replacement') < ci.index('Verify build'), 'CI gate order changed')
print('PASS_RELEASE_CHECK')
