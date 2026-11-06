#!/usr/bin/env python3
"""Add the Outros option to Pista de Dados, without changing existing categories.

Dry run by default. Idempotent for active categories of this type and name.
The optional observacoes metadata field keeps the category editable through the
existing administration API, which requires nonempty metadata_tags.
"""
import argparse
import json
import os
from pathlib import Path
import subprocess
import tempfile

CATEGORY = 'Outros'
TYPE = 'Pista de Dados'


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
        return 'NULL' if value is None else "''" if value == '' else 'CONVERT(0x' + value.encode().hex() + ' USING utf8mb4)'
    fd, name = tempfile.mkstemp(prefix='onlife-category-', suffix='.cnf')
    try:
        with os.fdopen(fd, 'w') as out:
            out.write('[client]\n' + ''.join(k+'='+quoted(v)+'\n' for k,v in {'host':host,'port':settings['DB_PORT'],'user':settings['DB_USER'],'password':settings['DB_PASSWORD'],'default-character-set':'utf8mb4'}.items()))
        def query(sql):
            result = subprocess.run(['mysql', '--defaults-extra-file='+name, '--batch', '--raw', '--skip-column-names', '-D', settings['DB_NAME']], input=sql, capture_output=True, text=True)
            if result.returncode: raise SystemExit('Category configuration failed; credential details omitted')
            return result.stdout.splitlines()
        condition = 'deleted_at IS NULL AND BINARY name='+literal(CATEGORY)+' AND BINARY type='+literal(TYPE)
        select = "SELECT JSON_OBJECT('id',id,'name',name,'type',type,'color_hex',color_hex,'metadata_tags',metadata_tags) FROM categories WHERE "+condition
        before = query(select)
        if len(before) > 1: raise SystemExit('Duplicate active categories detected; no changes applied')
        created = False
        if args.apply and not before:
            result = query("START TRANSACTION; INSERT INTO categories (created_at,updated_at,name,type,color_hex,icon_code,description,metadata_tags) SELECT NOW(),NOW(),"+
                ','.join(literal(v) for v in [CATEGORY,TYPE,'#21B6A8','','Outras pistas de dados.','observacoes'])+
                ' WHERE NOT EXISTS (SELECT 1 FROM categories WHERE '+condition+'); SELECT ROW_COUNT(); COMMIT;')
            if result not in (['0'], ['1']): raise SystemExit('Unexpected insert result')
            created = result == ['1']
        after = query(select)
        if args.apply and len(after) != 1: raise SystemExit('Category readback mismatch')
        print(json.dumps({'applied':args.apply,'created':created,'change_required':not before,
            'category':json.loads(after[0]) if after else {'name':CATEGORY,'type':TYPE}},ensure_ascii=False))
    finally:
        Path(name).unlink(missing_ok=True)


if __name__ == '__main__': main()
