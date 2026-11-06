#!/usr/bin/env python3
"""Compare pre-existing database rows without exporting their contents.

Run before/after a deployment on the API host. Snapshots contain column names,
row IDs and SHA-256 values only and remain private. Never restores or deletes.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('phase', choices=['before', 'after'])
parser.add_argument('--snapshot', type=Path, required=True)
parser.add_argument('--api-directory', type=Path, default=Path('/var/www/unisinos_usp_api'))
args = parser.parse_args()
os.umask(0o077)
settings = {}
for line in (args.api_directory / '.env').read_text().splitlines():
    key, sep, value = line.partition('=')
    if sep:
        settings[key.strip()] = value.strip().strip('"\'')
host = settings['DB_HOST']
if host == 'host.docker.internal':
    info = json.loads(subprocess.check_output(['docker', 'inspect', 'UnisinosUspAPI']))[0]
    host = next(n['Gateway'] for n in info['NetworkSettings']['Networks'].values() if n.get('Gateway'))

def identifier(value):
    if not re.fullmatch(r'[A-Za-z0-9_]+', value):
        raise SystemExit('Unexpected SQL identifier; no data changed')
    return '`' + value + '`'

def quoted(value):
    return '"' + value.replace('\\', '\\\\').replace('"', '\\"').replace('\n', '\\n') + '"'

fd, name = tempfile.mkstemp(prefix='onlife-preserve-', suffix='.cnf')
try:
    with os.fdopen(fd, 'w') as stream:
        stream.write('[client]\n' + ''.join(k + '=' + quoted(v) + '\n' for k, v in {
            'host': host, 'port': settings['DB_PORT'], 'user': settings['DB_USER'],
            'password': settings['DB_PASSWORD'],
        }.items()))
    def query(sql):
        result = subprocess.run(['mysql', '--defaults-extra-file=' + name, '--batch',
                                 '--skip-column-names', '-D', settings['DB_NAME'], '-e', sql],
                                capture_output=True, text=True)
        if result.returncode:
            raise SystemExit('Database verification failed; secret details were not printed')
        return result.stdout.splitlines()

    previous = None
    if args.phase == 'after':
        previous = json.loads(args.snapshot.read_text())
    elif args.snapshot.exists():
        raise SystemExit('Refusing to overwrite a preservation snapshot')
    names = list(previous) if previous is not None else query('SHOW TABLES')
    snapshot = {}
    changed = []
    for table in names:
        columns = (previous[table]['columns'] if previous is not None else
                   [line.split('\t')[0] for line in query('SHOW COLUMNS FROM ' + identifier(table))])
        projection = ','.join(identifier(c) for c in columns)
        if 'id' in columns:
            lines = query('SELECT `id`,SHA2(CAST(JSON_ARRAY(' + projection + ') AS CHAR),256) FROM ' + identifier(table))
            rows = dict(line.split('\t', 1) for line in lines)
            entry = {'columns': columns, 'rows': rows}
            if previous is not None:
                old = previous[table]['rows']
                missing = sum(key not in rows for key in old)
                modified = sum(key in rows and rows[key] != digest for key, digest in old.items())
                if missing or modified:
                    changed.append({'table': table, 'missing_rows': missing, 'modified_rows': modified})
        else:
            lines = query('SELECT SHA2(CAST(JSON_ARRAY(' + projection + ') AS CHAR),256) FROM ' + identifier(table))
            digest = hashlib.sha256('\n'.join(sorted(lines)).encode()).hexdigest()
            entry = {'columns': columns, 'table_digest': digest, 'count': len(lines)}
            if previous is not None and entry != previous[table]:
                changed.append({'table': table, 'changed': True})
        snapshot[table] = entry
    if previous is None:
        args.snapshot.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
        with args.snapshot.open('x') as stream:
            json.dump(snapshot, stream, indent=2)
        args.snapshot.chmod(0o600)
    print(json.dumps({'phase': args.phase, 'tables': len(names), 'preserved': not changed,
                      'differences': changed}, ensure_ascii=False))
    if changed:
        raise SystemExit(1)
finally:
    Path(name).unlink(missing_ok=True)
