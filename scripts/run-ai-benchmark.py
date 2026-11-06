#!/usr/bin/env python3
"""Run the education benchmark in a dedicated local synthetic database.

--mode gemini is an explicit paid-provider invocation. Only the authorized
Gemini credential is read from SUSI AI; no prompts, cases or user data are used.
"""
import argparse
import json
import os
from pathlib import Path
import subprocess
import sys

parser=argparse.ArgumentParser()
parser.add_argument('--mode',choices=['deterministic','gemini'],default='deterministic')
parser.add_argument('--case',default='')
parser.add_argument('--tag',default='')
parser.add_argument('--runs',type=int,default=3)
parser.add_argument('--output',default='')
args=parser.parse_args()
root=Path(__file__).resolve().parent.parent
container=json.loads(subprocess.check_output(['docker','inspect','rieonlife-validation-db']))[0]
settings=dict(item.split('=',1) for item in container['Config']['Env'] if '=' in item)
binding=container['HostConfig']['PortBindings']['3306/tcp'][0]
if binding!={'HostIp':'127.0.0.1','HostPort':'33419'}:parser.error('Benchmark database must use the isolated loopback binding')
subprocess.run(['docker','exec','rieonlife-validation-db','sh','-c','MYSQL_PWD="$MARIADB_ROOT_PASSWORD" mariadb -uroot -e "CREATE DATABASE IF NOT EXISTS onlife_validation_benchmark CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"'],check=True,capture_output=True)
env=os.environ.copy()
env.update(DB_HOST='127.0.0.1',DB_PORT='33419',DB_NAME='onlife_validation_benchmark',DB_USER='root',DB_PASSWORD=settings['MARIADB_ROOT_PASSWORD'],API_SECRET='synthetic-benchmark-no-login-tokens',GENAI_API_KEY='',GEMINI_MODEL='gemini-3.8-flash',ALLOW_DESTRUCTIVE_LEGACY_MIGRATIONS='false',ENVIRONMENTAL_CONTEXT_API_URL='http://127.0.0.1:1',GRYPHON_API_BASE_URL='http://127.0.0.1:1',ONLIFE_BENCHMARK_ALLOW_PROVIDER='0')
if args.mode=='gemini':
    for line in Path('/home/odair/projects/susi_ai/.env').read_text().splitlines():
        key,separator,value=line.partition('=')
        if separator and key.strip()=='GEMINI_API_KEY':env['GENAI_API_KEY']=value.strip().strip('"\'');break
    if not env['GENAI_API_KEY']:parser.error('Authorized provider credential unavailable')
    env['ONLIFE_BENCHMARK_ALLOW_PROVIDER']='1'
command=['go','run','./tests/benchmark/cmd','--mode',args.mode,'--runs',str(args.runs)]
for name in ['case','tag','output']:
    if getattr(args,name):command.extend(['--'+name,getattr(args,name)])
result=subprocess.run(command,cwd=root,env=env)
sys.exit(result.returncode)
