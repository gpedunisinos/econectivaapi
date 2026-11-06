#!/usr/bin/env python3
"""Check user optional-phone persistence against the isolated USP preview, then clean up."""
import argparse
import json
from pathlib import Path
import subprocess
from urllib.error import HTTPError
from urllib.request import Request, urlopen
import uuid

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--credentials', type=Path,
                    default=Path('/tmp/onlife-integration-preview-credentials.json'))
parser.add_argument('--report', type=Path,
                    default=Path('tests/validation/usp-user-optional.json'))
args = parser.parse_args()
if args.credentials.stat().st_mode & 0o077:
    parser.error('Credentials must be private (chmod 600).')
container = json.loads(subprocess.check_output(['docker', 'inspect', 'rieonlife-validation-db']))[0]
assert container['HostConfig']['PortBindings']['3306/tcp'][0] == {
    'HostIp': '127.0.0.1', 'HostPort': '33419'}
base = 'http://127.0.0.1:8082/api'
credentials = json.loads(args.credentials.read_text())
token = None
def api(method, path, data=None):
    headers = {'Content-Type': 'application/json'}
    if token:
        headers['Authorization'] = 'Bearer '+token
    body = json.dumps(data).encode() if data is not None else None
    with urlopen(Request(base+path, body, headers, method=method), timeout=30) as response:
        raw = response.read()
        return json.loads(raw) if raw else None

login = api('POST', '/login', {'email': credentials['email'],
    'password': credentials['password'], 'device_name': 'sample-update-regression'})
token = login.get('token') or login.get('plain_text_token')
assert token
api('PUT', '/me/preferences', {'usage_mode': 'inventor'})
import secrets
unique = 'usp-user-optional-'+uuid.uuid4().hex
password = secrets.token_urlsafe(24)
body = {'name': 'USP user optional regression', 'email': unique+'@example.test',
        'password': password, 'profile': 10, 'phone': '00000000000'}
created = api('POST', '/users', body)
own_id = created['id']
cases = []
try:
    from urllib.parse import quote
    baseline = api('GET', f'/users/{own_id}')
    for search in (body['email'], body['name']):
        matches = api('GET', '/users?limit=20&search='+quote(search))
        assert any(row['id'] == own_id for row in matches)
    assert not api('GET', '/users?search='+quote(unique+'-absent'))
    update = {k: v for k, v in body.items() if k != 'password'}
    for case in ('omitted', 'null', 'empty'):
        payload = dict(update)
        if case == 'omitted':
            del payload['phone']
        else:
            payload['phone'] = None if case == 'null' else ''
        api('PUT', f'/users/{own_id}', payload)
        actual = api('GET', f'/users/{own_id}')
        assert (actual.get('phone') or '') == ('' if case == 'empty' else '00000000000')
        for key in ('name', 'email', 'profile'):
            assert actual[key] == baseline[key]
        login = api('POST', '/login', {'email': body['email'], 'password': password,
                                      'device_name': 'optional-preservation-regression'})
        assert login['id'] == own_id
        cases.append({'phone_input': case, 'passed': True, 'password_retained': True})
finally:
    own = api('GET', f'/users/{own_id}')
    assert own['email'] == body['email']
    api('DELETE', f'/users/{own_id}')
try:
    api('GET', f'/users/{own_id}')
    raise AssertionError('Own disposable account still exists')
except HTTPError as error:
    assert error.code == 404
args.report.parent.mkdir(parents=True, exist_ok=True)
args.report.write_text(json.dumps({'status': 'aprovado', 'cases': cases,
    'other_fields_preserved': True, 'search_name_and_email': True, 'cleanup_status': 404,
    'database': 'isolated onlife_validation_preview', 'production_changed': False}, indent=2)+'\n')
print('User optional-field regression passed; own account removed/404.', flush=True)
