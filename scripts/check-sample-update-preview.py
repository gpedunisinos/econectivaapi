#!/usr/bin/env python3
"""Check optional-field clearing against the isolated USP preview, then clean up."""
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
                    default=Path('tests/validation/usp-sample-update.json'))
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
detail = api('GET', '/sample_details_select?limit=1')[0]
unique = 'usp-sample-update-'+uuid.uuid4().hex
body = {'name': 'USP optional-field regression', 'sample_string': 'Synthetic',
    'sample_unique': unique, 'sample_date': '2026-10-15T00:00:00Z',
    'sample_nullable': 'Value to clear', 'sample_double': 2.5,
    'sample_detail_id': detail['id'], 'order_number': 1}
created = api('POST', '/sample_models', body)
sample_id = created['id']
cases = []
try:
    api('PUT', f'/sample_models/{sample_id}', body)
    baseline = api('GET', f'/sample_models/{sample_id}')
    for case, value in [('omitted', 'omitted'), ('null', None), ('empty', '')]:
        payload = dict(body)
        if case == 'omitted':
            del payload['sample_nullable']
        else:
            payload['sample_nullable'] = value
        api('PUT', f'/sample_models/{sample_id}', payload)
        actual = api('GET', f'/sample_models/{sample_id}')
        assert actual['sample_nullable'] == ('' if case == 'empty' else 'Value to clear')
        for key in body.keys()-{'sample_nullable'}:
            assert actual[key] == baseline[key], key
        cases.append({'case': case, 'passed': True})
finally:
    own = api('GET', f'/sample_models/{sample_id}')
    assert own['sample_unique'] == unique
    api('DELETE', f'/sample_models/{sample_id}')
try:
    api('GET', f'/sample_models/{sample_id}')
    raise AssertionError('Disposable sample was not deleted')
except HTTPError as error:
    assert error.code == 404
args.report.parent.mkdir(parents=True, exist_ok=True)
args.report.write_text(json.dumps({'status': 'aprovado', 'cases': cases,
    'other_form_fields_preserved': True, 'cleanup_status': 404,
    'database': 'isolated onlife_validation_preview', 'production_changed': False}, indent=2)+'\n')
print('Sample optional-field preview regression passed; own record removed.', flush=True)
