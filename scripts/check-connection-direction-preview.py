#!/usr/bin/env python3
"""Check connection-direction persistence against the isolated USP preview, then clean up."""
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
                    default=Path('tests/validation/usp-connection-direction.json'))
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
unique = 'usp-direction-regression-'+uuid.uuid4().hex
cases = []
owned = []
try:
    for explicit in (None, False, True):
        body = {'name': unique+str(explicit), 'stroke_style': 'dashed'}
        if explicit is not None:
            body['is_directional'] = explicit
        created = api('POST', '/connection_types', body)
        own_id = created['id']
        owned.append(own_id)
        expected = True if explicit is None else explicit
        assert created['is_directional'] is expected
        actual = api('GET', f'/connection_types/{own_id}')
        assert actual['is_directional'] is expected
        for direction in (True, False, True, False):
            payload = dict(body, is_directional=direction)
            api('PUT', f'/connection_types/{own_id}', payload)
            actual = api('GET', f'/connection_types/{own_id}')
            assert actual['is_directional'] is direction
            assert actual['name'] == body['name']
            assert actual['stroke_style'] == 'dashed'
        # Omitting the field on update must preserve the last stored false.
        api('PUT', f'/connection_types/{own_id}', {'name': body['name'], 'stroke_style': 'dashed'})
        assert api('GET', f'/connection_types/{own_id}')['is_directional'] is False
        cases.append({'create_input': explicit, 'create_expected': expected,
                      'toggle_true_false': True, 'omitted_update_preserved': True})
finally:
    for own_id in owned:
        own = api('GET', f'/connection_types/{own_id}')
        assert own['name'].startswith(unique)
        api('DELETE', f'/connection_types/{own_id}')
remaining = api('GET', '/connection_types?search='+unique)
assert not remaining
args.report.parent.mkdir(parents=True, exist_ok=True)
args.report.write_text(json.dumps({'status': 'aprovado', 'cases': cases,
    'cleanup': True, 'database': 'isolated onlife_validation_preview',
    'production_changed': False}, indent=2)+'\n')
print('Connection direction preview regression passed; own records removed.', flush=True)
