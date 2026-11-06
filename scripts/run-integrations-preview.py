#!/usr/bin/env python3
"""Launch an isolated loopback API preview with generated fixture credentials.

The only optional external AI access is enabled explicitly by --gemini. No
credential is placed in a command argument or printed. Stop with Ctrl-C.
"""
import argparse
import json
import os
from pathlib import Path
import secrets
import subprocess

parser=argparse.ArgumentParser()
parser.add_argument('--gemini', action='store_true', help='Explicitly enable authorized Gemini calls (uses quota).')
parser.add_argument('--voice', action='store_true', help='Explicitly enable authorized OpenAI transcription and narration using the API .env (uses quota).')
parser.add_argument('--s3', action='store_true', help='Use the private S3 bucket with temporary AWS CLI credentials for synthetic attachments.')
args=parser.parse_args()
root=Path(__file__).resolve().parent.parent
fixture=Path('/tmp/onlife-integration-preview-credentials.json')
if fixture.exists():
    credentials=json.loads(fixture.read_text())
else:
    credentials={'email':'integrations-preview@example.test','password':secrets.token_urlsafe(30),'signing_key':secrets.token_urlsafe(48)}
    descriptor=os.open(fixture,os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600)
    with os.fdopen(descriptor,'w') as out: json.dump(credentials,out)
fixture.chmod(0o600)
container=json.loads(subprocess.check_output(['docker','inspect','rieonlife-validation-db']))[0]
settings=dict(item.split('=',1) for item in container['Config']['Env'] if '=' in item)
binding=container['HostConfig']['PortBindings']['3306/tcp'][0]
if binding!={'HostIp':'127.0.0.1','HostPort':'33419'}: raise SystemExit('Refusing non-isolated database binding')
database='onlife_validation_preview'
subprocess.run(['docker','exec','rieonlife-validation-db','sh','-c','MYSQL_PWD="$MARIADB_ROOT_PASSWORD" mariadb -uroot -e "CREATE DATABASE IF NOT EXISTS onlife_validation_preview CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"'],check=True,capture_output=True)
env=os.environ.copy()
env.update(OPENAI_API_KEY='', PRESENCE_ALLOWED_ORIGINS='http://127.0.0.1:8773,http://localhost:8773',DB_HOST='127.0.0.1',DB_PORT='33419',DB_NAME=database,DB_USER='root',DB_PASSWORD=settings['MARIADB_ROOT_PASSWORD'],API_SECRET=credentials['signing_key'],ONLIFE_PREVIEW_PASSWORD=credentials['password'],GENAI_API_KEY='',GEMINI_MODEL='gemini-3.8-flash',ALLOW_DESTRUCTIVE_LEGACY_MIGRATIONS='false',ENVIRONMENTAL_CONTEXT_API_URL='http://127.0.0.1:1',GRYPHON_API_BASE_URL='http://127.0.0.1:1')
# Use only the dedicated meeting settings from the protected local preview env.
# Database, session and provider credentials remain isolated above.
meeting_env=root/'.env'
if meeting_env.exists():
    for line in meeting_env.read_text().splitlines():
        key,sep,value=line.partition('=')
        if sep and key in {'JITSI_PUBLIC_URL','JITSI_APP_ID','JITSI_APP_SECRET','JITSI_XMPP_DOMAIN'}:
            env[key]=value.strip().strip('"\'')
# Preview storage is local unless S3 is explicitly requested; never inherit
# production storage settings or public URLs from the checkout .env.
env.update(STORAGE_DRIVER='local', APP_URL='http://127.0.0.1:8082', MAIL_HOST='127.0.0.1', MAIL_PORT='1')
if args.s3:
    exported=json.loads(subprocess.check_output(['aws','configure','export-credentials','--format','process']))
    env.update(STORAGE_DRIVER='s3', AWS_REGION='sa-east-1',
        S3_BUCKET='educacao-onlife-arquivos-247433253659-sa-east-1',
        AWS_ACCESS_KEY_ID=exported['AccessKeyId'],AWS_SECRET_ACCESS_KEY=exported['SecretAccessKey'],
        AWS_SESSION_TOKEN=exported.get('SessionToken',''))
if args.gemini:
    source=Path('/home/odair/projects/susi_ai/.env')
    for line in source.read_text().splitlines():
        key,separator,value=line.partition('=')
        if separator and key.strip()=='GEMINI_API_KEY': env['GENAI_API_KEY']=value.strip().strip('"\''); break
    if not env['GENAI_API_KEY']: raise SystemExit('Authorized provider credential unavailable')
if args.voice:
    source=root/'.env'
    if source.exists():
        for line in source.read_text().splitlines():
            key,separator,value=line.partition('=')
            if separator and key.strip() in {'OPENAI_API_KEY','OPENAI_TRANSCRIBE_MODEL','OPENAI_VOICE_MODEL','OPENAI_VOICE'}:
                env[key.strip()]=value.strip().strip('"\'')
    if not env.get('OPENAI_API_KEY'): raise SystemExit('Configure OPENAI_API_KEY in the API .env before enabling --voice')
log=Path('/tmp/onlife-integration-preview.log'); log.touch(mode=0o600); log.chmod(0o600)
print('Starting isolated preview on 127.0.0.1:8082. Fixture credentials saved in a protected local file; values are not printed.',flush=True)
with log.open('w') as output:
    try:
        result=subprocess.run(['go','run','./tools/integrations-preview'],cwd=root,env=env,stdout=output,stderr=subprocess.STDOUT)
    except KeyboardInterrupt: raise SystemExit(130)
raise SystemExit(result.returncode)
