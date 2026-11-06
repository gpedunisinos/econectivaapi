#!/usr/bin/env python3
"""Validate sample numeric presence against the isolated loopback preview only."""
import argparse
import json
from pathlib import Path
import urllib.error
import urllib.request
import uuid

parser = argparse.ArgumentParser()
parser.add_argument('--defines', default='/tmp/usp-plan-defines.json')
parser.add_argument('--report', default='/tmp/usp-sample-zero-http-checks.json')
args = parser.parse_args()
defines = json.loads(Path(args.defines).read_text())
base = 'http://127.0.0.1:8082/api'
token = None

def call(method, path, body=None):
    headers = {'Content-Type': 'application/json'}
    if token:
        headers['Authorization'] = 'Bearer ' + token
    request = urllib.request.Request(base + path, method=method, headers=headers,
                                    data=None if body is None else json.dumps(body).encode())
    try:
        with urllib.request.urlopen(request, timeout=30) as response:
            return response.status, json.load(response)
    except urllib.error.HTTPError as error:
        try:
            data = json.load(error)
        except (ValueError, UnicodeDecodeError):
            data = {'message': 'Non-JSON error response'}
        return error.code, data

status, auth = call('POST', '/login', {'email': defines['E2E_EMAIL'], 'password': defines['E2E_PASSWORD']})
assert status == 200
token = auth.get('token') or auth.get('plain_text_token')
assert token
status, details = call('GET', '/sample_details_select?limit=100')
assert status == 200 and isinstance(details, list) and details
name = 'USP numeric disposable ' + uuid.uuid4().hex
payload = {'name': name, 'sample_string': 'Synthetic', 'sample_unique': uuid.uuid4().hex,
           'sample_date': '2026-10-26T03:00:00Z', 'sample_detail_id': details[0]['id'],
           'sample_nullable': 'Optional', 'sample_double': 0, 'order_number': 1}
created = None
checks = {}
try:
    for label, value in [('missing', 'omit'), ('null', None)]:
        invalid = dict(payload)
        if value == 'omit':
            invalid.pop('sample_double')
        else:
            invalid['sample_double'] = value
        status, _ = call('POST', '/sample_models', invalid)
        assert status == 400, label
        checks['create_' + label + '_rejected'] = True
    status, model = call('POST', '/sample_models', payload)
    assert status == 200 and model['sample_double'] == 0
    created = model['id']
    checks['create_zero_accepted'] = True
    for value in [12.5, 0]:
        changed = dict(payload, sample_double=value, sample_nullable='')
        status, _ = call('PUT', '/sample_models/' + str(created), changed)
        assert status == 200
        status, saved = call('GET', '/sample_models/' + str(created))
        assert status == 200 and saved['sample_double'] == value and saved['sample_nullable'] == ''
    checks['nonzero_then_zero_persisted'] = True
    checks['optional_clear_preserved'] = True
    for label in ['missing', 'null']:
        invalid = dict(payload)
        if label == 'missing':
            invalid.pop('sample_double')
        else:
            invalid['sample_double'] = None
        status, _ = call('PUT', '/sample_models/' + str(created), invalid)
        assert status == 400, label
        checks['update_' + label + '_rejected'] = True
finally:
    if created is not None:
        status, model = call('GET', '/sample_models/' + str(created))
        assert status == 200 and model['name'] == name
        status, _ = call('DELETE', '/sample_models/' + str(created))
        assert status == 200
        status, _ = call('GET', '/sample_models/' + str(created))
        assert status == 404
        checks['own_fixture_cleanup_404'] = True
report = {'status': 'aprovado no escopo descrito', 'environment': 'isolated loopback preview',
          'checks': checks, 'production_mutated': False}
Path(args.report).write_text(json.dumps(report, indent=2) + '\n')
print('Sample numeric HTTP validation and scoped cleanup passed.')
