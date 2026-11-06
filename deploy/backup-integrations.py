#!/usr/bin/env python3
"""Back up API data/config and web/Apache state on the existing server.

Read-only against application data. Secrets never go to stdout or arguments.
Archives and database dumps remain on this server in a private directory.
"""
import argparse
from datetime import datetime,timezone
import gzip
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile

parser=argparse.ArgumentParser()
parser.add_argument('--api-directory',type=Path,default=Path('/var/www/unisinos_usp_api'))
parser.add_argument('--web-directory',type=Path,default=Path('/var/www/usp_app'))
parser.add_argument('--backup-root',type=Path,default=Path('/home/ubuntu/backups'))
args=parser.parse_args()
os.umask(0o077)
base=args.backup_root/('onlife-integrations-'+datetime.now(timezone.utc).strftime('%Y%m%dT%H%M%SZ'))
base.mkdir(parents=True,exist_ok=False,mode=0o700)

def save(name,data):
    target=base/name
    target.write_bytes(data)
    target.chmod(0o600)
    return target

def environment(text):
    result={}
    for line in text.splitlines():
        key,separator,value=line.partition('=')
        if separator and key.strip() in {'DB_HOST','DB_PORT','DB_NAME','DB_USER','DB_PASSWORD'}:
            result[key.strip()]=value.strip().strip('"\'')
    return result

runtime=subprocess.check_output(['docker','exec','UnisinosUspAPI','cat','/app/src/unisinos_usp/.env'])
settings=environment(runtime.decode())
if not all(settings.get(k) for k in ['DB_HOST','DB_PORT','DB_NAME','DB_USER','DB_PASSWORD']):raise SystemExit('Runtime database parameters incomplete; no data modified')
save('api-runtime.env',runtime)
save('api-host.env',(args.api_directory/'.env').read_bytes())
container_data=subprocess.check_output(['docker','inspect','UnisinosUspAPI'])
save('api-container-before.json',container_data)
save('containers-before.txt',subprocess.check_output(['docker','ps','--format','{{.Names}} {{.Image}} {{.Status}} {{.Ports}}']))
host=settings['DB_HOST']
if host=='host.docker.internal':
    gateways=[network.get('Gateway') for network in json.loads(container_data)[0]['NetworkSettings']['Networks'].values() if network.get('Gateway')]
    if not gateways:raise SystemExit('Cannot determine the application database bridge gateway')
    host=gateways[0]
def quote(value):return '"'+value.replace('\\','\\\\').replace('"','\\"').replace('\n','\\n')+'"'
fd,temporary=tempfile.mkstemp(prefix='onlife-db-',suffix='.cnf')
try:
    with os.fdopen(fd,'w') as out:
        out.write('[client]\n'+''.join(k+'='+quote(v)+'\n' for k,v in {'host':host,'port':settings['DB_PORT'],'user':settings['DB_USER'],'password':settings['DB_PASSWORD']}.items()))
    os.chmod(temporary,0o600)
    common=['--defaults-extra-file='+temporary]
    tables=subprocess.run(['mysql',*common,'--batch','--skip-column-names','-D',settings['DB_NAME'],'-e','SHOW TABLES'],capture_output=True,text=True)
    if tables.returncode:raise SystemExit('Cannot read the configured database; secret details were not printed')
    counts={}
    for name in tables.stdout.splitlines():
        if not re.fullmatch(r'[A-Za-z0-9_]+',name):raise SystemExit('Unexpected table identifier')
        value=subprocess.run(['mysql',*common,'--batch','--skip-column-names','-D',settings['DB_NAME'],'-e','SELECT COUNT(*) FROM `'+name+'`'],capture_output=True,text=True,check=True)
        counts[name]=int(value.stdout.strip())
    save('database-counts-before.json',json.dumps(counts,indent=2).encode())
    partial=base/'database.sql.gz.partial'
    dump=subprocess.Popen(['mysqldump',*common,'--single-transaction','--quick','--no-tablespaces','--routines','--triggers','--events',settings['DB_NAME']],stdout=subprocess.PIPE,stderr=subprocess.PIPE)
    with gzip.open(partial,'wb') as output:shutil.copyfileobj(dump.stdout,output)
    dump.stdout.close();errors=dump.stderr.read();dump.wait()
    if dump.returncode:raise SystemExit('SQL dump failed; partial file retained privately, no data modified')
    if partial.stat().st_size<1000:raise SystemExit('SQL dump unexpectedly small; do not proceed with deploy')
    partial.rename(base/'database.sql.gz')
finally:
    Path(temporary).unlink(missing_ok=True)

for name,paths in [('apache-tls-before.tgz',['/etc/apache2','/etc/letsencrypt']),('web-before.tgz',[str(args.web_directory)]),('storage-before.tgz',[str(args.api_directory/'storage')])]:
    result=subprocess.run(['sudo','tar','-czf',str(base/name),*paths],capture_output=True)
    if result.returncode:raise SystemExit('Archive preparation failed: '+name+'; no data modified')
    subprocess.run(['sudo','chmod','600',str(base/name)],check=True)
manifest={}
for file in base.iterdir():
    if file.suffix=='.tgz':
        value=subprocess.check_output(['sudo','sha256sum',str(file)],text=True).split()[0]
    else:value=hashlib.sha256(file.read_bytes()).hexdigest()
    manifest[file.name]={'sha256':value,'bytes':file.stat().st_size}
save('manifest.json',json.dumps(manifest,indent=2).encode())
print(json.dumps({'backup':str(base),'tables':len(counts),'archives':len(manifest),'complete':True}))
