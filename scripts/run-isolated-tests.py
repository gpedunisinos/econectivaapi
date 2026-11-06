#!/usr/bin/env python3
"""Run Go tests against a local, synthetic MariaDB database, never the server DB.

Secrets are read in memory from the local container and are not printed. The
database is created if absent, never dropped. Provider calls are disabled.
"""
import argparse
from datetime import datetime, timezone
import json
import os
from pathlib import Path
import re
import subprocess
import sys

parser = argparse.ArgumentParser()
parser.add_argument('--container', default='rieonlife-validation-db')
parser.add_argument('--database', default='onlife_validation_integrations')
parser.add_argument('--fresh-database', action='store_true', help='Create a new uniquely named synthetic database; preserve previous runs.')
parser.add_argument('--run', default='TestClueIntegrations|TestIareteMCP')
parser.add_argument('--package', default='./tests')
parser.add_argument('--log', default='/tmp/onlife-isolated-tests.log')
args = parser.parse_args()
if not re.fullmatch(r'onlife_validation_[a-z0-9_]+', args.database):
    parser.error('Use an isolated onlife_validation_ database name.')
if args.fresh_database:
    args.database = args.database[:40] + '_' + datetime.now(timezone.utc).strftime('%Y%m%d_%H%M%S_%f')
if len(args.database) > 64:
    parser.error('Database name exceeds the MySQL identifier limit.')
root = Path(__file__).resolve().parent.parent
container = json.loads(subprocess.check_output(['docker', 'inspect', args.container]))[0]
values = dict(item.split('=', 1) for item in container['Config']['Env'] if '=' in item)
secret = values.get('MARIADB_ROOT_PASSWORD') or values.get('MYSQL_ROOT_PASSWORD')
if not secret:
    parser.error('Local container does not expose its test database credential.')
bindings = container['HostConfig']['PortBindings'].get('3306/tcp', [])
if not bindings or bindings[0]['HostIp'] != '127.0.0.1':
    parser.error('Test database must bind explicitly to loopback.')
command = 'mariadb' if 'MARIADB_ROOT_PASSWORD' in values else 'mysql'
password_name = 'MARIADB_ROOT_PASSWORD' if command == 'mariadb' else 'MYSQL_ROOT_PASSWORD'
create = f'MYSQL_PWD="${password_name}" {command} -uroot -e "CREATE DATABASE IF NOT EXISTS {args.database} CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"'
subprocess.run(['docker', 'exec', args.container, 'sh', '-c', create], check=True, capture_output=True)
env = os.environ.copy()
env.update(DB_HOST='127.0.0.1', DB_PORT=bindings[0]['HostPort'], DB_NAME=args.database,
           DB_USER='root', DB_PASSWORD=secret, GIN_MODE='test', GENAI_API_KEY='',
           API_SECRET='isolated-onlife-test-signing-key-not-a-server-credential', OPENAI_API_KEY='',
           ENVIRONMENTAL_CONTEXT_API_URL='http://127.0.0.1:1', GRYPHON_API_BASE_URL='http://127.0.0.1:1',
           MAIL_HOST='127.0.0.1', MAIL_PORT='1', MAIL_USERNAME='', MAIL_PASSWORD='', SENTRY_DSN='')
log = Path(args.log)
log.touch(mode=0o600)
log.chmod(0o600)
with log.open('w') as output:
    result = subprocess.run(['go', 'test', args.package, '-run', args.run, '-count=1'],
                            cwd=root, env=env, stdout=output, stderr=subprocess.STDOUT)
print(json.dumps({'database': args.database, 'exit_code': result.returncode, 'log': str(log)}))
if result.returncode:
    print(log.read_text().replace(secret, '[REDACTED]')[-10000:])
sys.exit(result.returncode)
