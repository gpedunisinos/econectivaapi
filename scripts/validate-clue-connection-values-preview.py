#!/usr/bin/env python3
"""Exercise explicit empty/zero relation values only in the loopback preview."""
import argparse
import json
from pathlib import Path
import urllib.error
import urllib.request
import uuid

p = argparse.ArgumentParser()
p.add_argument('--defines', default='/tmp/usp-plan-defines.json')
p.add_argument('--report', default='/tmp/usp-connection-values-checks.json')
args = p.parse_args()
c = json.loads(Path(args.defines).read_text())
base = 'http://127.0.0.1:8082/api'
token = None

def call(method, path, data=None):
    headers = {'Content-Type': 'application/json'}
    if token:
        headers['Authorization'] = 'Bearer ' + token
    req = urllib.request.Request(base + path, method=method, headers=headers,
                                 data=None if data is None else json.dumps(data).encode())
    try:
        with urllib.request.urlopen(req, timeout=30) as r:
            return r.status, json.load(r)
    except urllib.error.HTTPError as e:
        try:
            body = json.load(e)
        except ValueError:
            body = {}
        return e.code, body

status, auth = call('POST', '/login', {'email': c['E2E_EMAIL'], 'password': c['E2E_PASSWORD']})
assert status == 200
token = auth.get('token') or auth.get('plain_text_token')
assert token
status, clues = call('GET', '/living_clues?limit=2')
assert status == 200
if isinstance(clues, dict):
    clues = clues.get('data', clues.get('items', []))
assert len(clues) >= 2
source, target = clues[0]['id'], clues[1]['id']
name = 'USP relation value disposable ' + uuid.uuid4().hex
status, kind = call('POST', '/connection_types', {'name': name, 'is_directional': True, 'stroke_style': 'solid'})
assert status == 200
owned = []
checks = {}
try:
    body = {'source_clue_id': source, 'target_clue_id': target, 'connection_type_id': kind['id'], 'evidence': 'Retain me'}
    status, link = call('POST', '/clue_connections', body)
    assert status == 200
    owned.append(link['id'])
    assert link['strength'] == 1
    checks['create_omitted_strength_defaults_one'] = True
    path = '/clue_connections/' + str(link['id'])
    changed = dict(body, strength=3)
    assert call('PUT', path, changed)[0] == 200
    for mode in ['omit', 'null']:
        omitted = {k: v for k, v in body.items() if k != 'evidence'}
        if mode == 'null':
            omitted.update(strength=None, evidence=None)
        assert call('PUT', path, omitted)[0] == 200
        status, saved = call('GET', path)
        assert status == 200 and saved['strength'] == 3 and saved['evidence'] == 'Retain me'
        checks[mode + '_preserves_strength_and_evidence'] = True
    assert call('PUT', path, dict(body, strength=0, evidence=''))[0] == 200
    status, saved = call('GET', path)
    assert status == 200 and saved['strength'] == 0 and saved['evidence'] == ''
    checks['explicit_zero_and_empty_persisted'] = True
    zero_body = dict(body, source_clue_id=target, target_clue_id=source, strength=0)
    status, zero = call('POST', '/clue_connections', zero_body)
    assert status == 200
    owned.append(zero['id'])
    status, saved = call('GET', '/clue_connections/' + str(zero['id']))
    assert status == 200 and saved['strength'] == 0
    checks['create_explicit_zero_persisted'] = True
finally:
    for identifier in owned:
        path = '/clue_connections/' + str(identifier)
        status, link = call('GET', path)
        assert status == 200 and link['connection_type_id'] == kind['id']
        assert call('DELETE', path)[0] == 200
        assert call('GET', path)[0] == 404
    status, current = call('GET', '/connection_types/' + str(kind['id']))
    assert status == 200 and current['name'] == name
    assert call('DELETE', '/connection_types/' + str(kind['id']))[0] == 200
    checks['own_links_and_type_cleaned'] = True
for clue in clues[:2]:
    status, saved = call('GET', '/living_clues/' + str(clue['id']))
    assert status == 200 and saved['name'] == clue['name']
checks['preexisting_clues_preserved'] = True
Path(args.report).write_text(json.dumps({'status': 'aprovado no escopo descrito', 'checks': checks,
                                         'production_mutated': False}, indent=2) + '\n')
print('Relation HTTP value semantics and scoped cleanup passed.')
