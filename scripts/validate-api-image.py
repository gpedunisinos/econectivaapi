#!/usr/bin/env python3
"""Exercise the built API image with a disposable container and synthetic DB.

No server, provider, user account or history is used. The dedicated database is
retained; only the test container and generated credential file are removed.
"""
import argparse
import json
import os
from pathlib import Path
import secrets
import subprocess
import tempfile
import time
import urllib.request
import urllib.error

parser=argparse.ArgumentParser()
parser.add_argument('--image',default='onlife-api:integrations-validation')
args=parser.parse_args()
inspect=json.loads(subprocess.check_output(['docker','image','inspect',args.image]))[0]
assert not any(v.startswith('GENAI_API_KEY=') or v.startswith('GEMINI_API_KEY=') for v in inspect['Config'].get('Env',[]))
check=subprocess.run(['docker','run','--rm','--entrypoint','sh',args.image,'-c','test ! -f /app/src/unisinos_usp/.env && command -v pdftotext >/dev/null && go version'],capture_output=True,text=True)
if check.returncode:raise SystemExit('Image contains environment data or lacks required runtime tools')
container=json.loads(subprocess.check_output(['docker','inspect','rieonlife-validation-db']))[0]
binding=container['HostConfig']['PortBindings']['3306/tcp'][0]
if binding!={'HostIp':'127.0.0.1','HostPort':'33419'}:raise SystemExit('Refusing unexpected database binding')
settings=dict(item.split('=',1) for item in container['Config']['Env'] if '=' in item)
networks=container['NetworkSettings']['Networks'];network=next(iter(networks));address=networks[network]['IPAddress']
subprocess.run(['docker','exec','rieonlife-validation-db','sh','-c','MYSQL_PWD="$MARIADB_ROOT_PASSWORD" mariadb -uroot -e "CREATE DATABASE IF NOT EXISTS onlife_validation_container CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"'],check=True,capture_output=True)
name='onlife-image-test-'+secrets.token_hex(4)
with tempfile.TemporaryDirectory(prefix='onlife-image-') as directory:
    envfile=Path(directory)/'.env'
    values={'DB_HOST':address,'DB_PORT':'3306','DB_NAME':'onlife_validation_container','DB_USER':'root','DB_PASSWORD':settings['MARIADB_ROOT_PASSWORD'],'API_SECRET':secrets.token_urlsafe(32),'GENAI_API_KEY':'','GEMINI_MODEL':'gemini-3.8-flash','ALLOW_DESTRUCTIVE_LEGACY_MIGRATIONS':'false','GRYPHON_API_BASE_URL':'http://127.0.0.1:1','ENVIRONMENTAL_CONTEXT_API_URL':'http://127.0.0.1:1','GIN_MODE':'release'}
    with envfile.open('w') as output:
        for key,value in values.items():output.write(key+'='+json.dumps(value)+'\n')
    envfile.chmod(0o600)
    try:
        subprocess.run(['docker','run','-d','--name',name,'--label','onlife.synthetic.validation=true','--network',network,'-p','127.0.0.1::8080','-v',str(envfile)+':/app/src/unisinos_usp/.env:ro',args.image],check=True,capture_output=True)
        metadata=json.loads(subprocess.check_output(['docker','inspect',name]))[0]
        port=metadata['NetworkSettings']['Ports']['8080/tcp'][0]['HostPort']
        base='http://127.0.0.1:'+port
        ready=False
        for _ in range(40):
            try:
                with urllib.request.urlopen(base+'/api/status',timeout=2) as response:ready=response.status==200
                if ready:break
            except (urllib.error.URLError,TimeoutError,ConnectionResetError):time.sleep(.5)
        if not ready:raise SystemExit('Image API did not become healthy; no server data was used')
        with urllib.request.urlopen(base+'/api/policy/privacy',timeout=5) as response:
            assert response.status==200
        try:
            request=urllib.request.Request(base+'/api/mcp/iarete',data=b'{"jsonrpc":"2.0","id":1,"method":"tools/list"}',headers={'Content-Type':'application/json'})
            urllib.request.urlopen(request,timeout=5)
            raise AssertionError('MCP accepted an unauthenticated request')
        except urllib.error.HTTPError as error:assert error.code==401
        print(json.dumps({'image':args.image,'go':check.stdout.strip(),'status':200,'policy':200,'mcp_without_auth':401,'environment_not_baked':True,'provider_calls':0}))
    finally:
        subprocess.run(['docker','rm','-f',name],capture_output=True)
