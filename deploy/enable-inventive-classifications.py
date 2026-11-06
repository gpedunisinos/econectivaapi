#!/usr/bin/env python3
"""Enable optional classification selectors for one existing category.

Dry run by default. Only metadata_tags is patched, guarded by its prior value.
Never classifies pistas, creates categories, or changes any other column.
"""
import argparse
import json
import os
from pathlib import Path
import subprocess
import tempfile

KEYS = ('praticas_inventivas', 'metodologias_inventivas')
CATEGORY = 'Vivência de Cidadania Digital'


def enabled_tags(previous):
    # Keep the original spelling/order/spacing of existing entries.
    missing = [key for key in KEYS if key not in [t.strip() for t in (previous or '').split(',')]]
    if not missing:
        return previous
    return (previous + ('' if previous.endswith(',') else ',') if previous else '') + ','.join(missing)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--api-directory', type=Path, default=Path('/var/www/unisinos_usp_api'))
    parser.add_argument('--apply', action='store_true')
    args = parser.parse_args()
    os.umask(0o077)
    settings = {}
    for line in (args.api_directory / '.env').read_text().splitlines():
        key, sep, value = line.partition('=')
        if sep: settings[key.strip()] = value.strip().strip('"\'')
    host = settings['DB_HOST']
    if host == 'host.docker.internal':
        info = json.loads(subprocess.check_output(['docker', 'inspect', 'UnisinosUspAPI']))[0]
        host = next(n['Gateway'] for n in info['NetworkSettings']['Networks'].values() if n.get('Gateway'))
    def quoted(value):
        return '"' + value.replace('\\', '\\\\').replace('"', '\\"').replace('\n', '\\n') + '"'
    def literal(value):
        return 'NULL' if value is None else 'CONVERT(0x' + value.encode().hex() + ' USING utf8mb4)'
    fd, name = tempfile.mkstemp(prefix='onlife-category-', suffix='.cnf')
    try:
        with os.fdopen(fd, 'w') as out:
            out.write('[client]\n' + ''.join(k+'='+quoted(v)+'\n' for k,v in {'host':host,'port':settings['DB_PORT'],'user':settings['DB_USER'],'password':settings['DB_PASSWORD'],'default-character-set':'utf8mb4'}.items()))
        def query(sql):
            result = subprocess.run(['mysql', '--defaults-extra-file='+name, '--batch', '--raw', '--skip-column-names', '-D', settings['DB_NAME']], input=sql, capture_output=True, text=True)
            if result.returncode: raise SystemExit('Category configuration failed; credential details omitted')
            return result.stdout.splitlines()
        rows = query("SELECT JSON_OBJECT('id',id,'name',name,'tags',metadata_tags) FROM categories WHERE deleted_at IS NULL AND BINARY name = " + literal(CATEGORY))
        if len(rows) != 1: raise SystemExit('Expected exactly one active category named '+CATEGORY+'; no changes applied')
        category = json.loads(rows[0]); previous = category['tags']; updated = enabled_tags(previous)
        changed = updated != previous
        if args.apply and changed:
            count = query('START TRANSACTION; UPDATE categories SET metadata_tags='+literal(updated)+' WHERE id='+str(int(category['id']))+' AND CAST(metadata_tags AS BINARY) <=> CAST('+literal(previous)+' AS BINARY); SELECT ROW_COUNT(); COMMIT;')
            if count != ['1']: raise SystemExit('Concurrent category change detected; rerun dry run before applying')
        actual = json.loads(query('SELECT JSON_OBJECT(\'tags\',metadata_tags) FROM categories WHERE id='+str(int(category['id'])))[0])['tags']
        if args.apply and actual != updated: raise SystemExit('Configuration readback mismatch')
        print(json.dumps({'category': CATEGORY, 'id':category['id'], 'applied':args.apply and changed,'change_required':changed,'previous_tags':previous,'configured_tags':actual,'planned_tags':updated}, ensure_ascii=False))
    finally:
        Path(name).unlink(missing_ok=True)


if __name__ == '__main__': main()
