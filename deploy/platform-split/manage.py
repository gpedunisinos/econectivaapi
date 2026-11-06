#!/usr/bin/env python3
"""Run on the production host. SQL uses the protected system client config.
Never prints credentials, dumps, user records, signed URLs or private messages.
Prepare can be repeated; clone --refresh is refused after activation.
"""
import argparse,datetime,gzip,hashlib,json,os,pathlib,re,secrets,shutil,subprocess,tarfile
SOURCE=pathlib.Path('/var/www/unisinos_usp_api');TARGET=pathlib.Path('/var/www/atopos_api')
DB='unisinos_usp_api';NEW_DB='atopos_api'
OLD_BUCKET='educacao-onlife-arquivos-247433253659-sa-east-1';NEW_BUCKET='atopos-arquivos-247433253659-sa-east-1'
STATE=pathlib.Path('/home/ubuntu/backups/platform-split')
def run(args,**kw):
 r=subprocess.run(args,capture_output=True,**kw)
 if r.returncode:raise RuntimeError('Operation failed: '+args[0]+' (details retained privately by the operator)')
 return r.stdout

def sql(text):return run(['sudo','mysql','--defaults-file=/etc/mysql/debian.cnf','--batch','--raw','--skip-column-names','--binary-mode'],input=text.encode()).decode().splitlines()
def settings(path):
 return json.loads(run([str(pathlib.Path(__file__).with_name('env-reader')),str(path)]))
def save_env(path,values):
 def quote(v):return '"'+v.replace('\\','\\\\').replace('"','\\"').replace('$','\\$').replace('\n','\\n')+'"'
 path.write_text(''.join(k+'='+quote(v)+'\n' for k,v in values.items()));path.chmod(0o600)
def identifier(value):
 assert re.fullmatch(r'[a-zA-Z0-9_]+',value);return '`'+value+'`'
def literal(value):return 'CONVERT(0x'+value.encode().hex()+' USING utf8mb4) COLLATE utf8mb4_unicode_ci'
def table_counts(db):
 tables=sql('SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA='+literal(db)+' ORDER BY TABLE_NAME;')
 return {t:int(sql('SELECT COUNT(*) FROM '+identifier(db)+'.'+identifier(t)+';')[0]) for t in tables}
def prepare():
 assert SOURCE.is_dir();STATE.mkdir(mode=0o700,parents=True,exist_ok=True);TARGET.mkdir(mode=0o700,exist_ok=True)
 marker=STATE/'prepared.json'
 if marker.exists():print(json.dumps({'prepared':True,'already_prepared':True}));return
 now=datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%SZ');backup=STATE/now;backup.mkdir(mode=0o700)
 shutil.copyfile(SOURCE/'.env',backup/'onlife.env');(backup/'onlife.env').chmod(0o600)
 shutil.copyfile(SOURCE/'docker-compose.yml',backup/'onlife-compose.yml')
 run(['docker','cp','UnisinosUspAPI:/app/src/unisinos_usp/main',str(backup/'onlife-main')])
 info=json.loads(run(['docker','inspect','UnisinosUspAPI']))[0]
 (backup/'container.json').write_text(json.dumps({'image':info['Image'],'config_image':info['Config']['Image']}))
 for site in ['api.uniusp.gpedu.com.br-le-ssl.conf','plataforma.atopos.spotec.app-ssl.conf']:
  (backup/site).write_bytes(run(['sudo','cat','/etc/apache2/sites-available/'+site]))
 old=settings(SOURCE/'.env');new=old.copy()
 # mysql credentials are deliberately unique and restricted to the cloned DB.
 password=secrets.token_hex(32)
 exists=sql("SELECT User FROM mysql.user WHERE User='atopos_api';")
 assert not exists,'Existing Atopos database account requires inspection'
 sql("CREATE DATABASE IF NOT EXISTS atopos_api CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci; CREATE USER 'atopos_api'@'%' IDENTIFIED BY '"+password+"'; GRANT ALL PRIVILEGES ON atopos_api.* TO 'atopos_api'@'%';")
 new.update({'DB_NAME':NEW_DB,'DB_USER':'atopos_api','DB_PASSWORD':password,'EXPECTED_DATABASE_NAME':NEW_DB,'PLATFORM_ID':'atopos','API_SECRET':secrets.token_hex(48),'APP_NAME':'Atopos','APP_URL':'https://api.atopos.spotec.app','PUBLIC_APP_URL':'https://plataforma.atopos.spotec.app','APP_PORT':'8084','API_CONTAINER_NAME':'AtoposAPI','API_ALLOWED_ORIGINS':'https://plataforma.atopos.spotec.app','PRESENCE_ALLOWED_ORIGINS':'https://plataforma.atopos.spotec.app','S3_BUCKET':NEW_BUCKET,'S3_ROLE_ARN':'arn:aws:iam::247433253659:role/AtoposPrivateAttachments','PLACE_SEARCH_SHARED_DIR':'/var/lib/onlife-place-search','ALLOW_DESTRUCTIVE_LEGACY_MIGRATIONS':'false'})
 save_env(TARGET/'.env',new)
 # Keep a protected rollback baseline and do not change OnLIFE credentials.
 (STATE/'prepared.json').write_text(json.dumps({'backup':str(backup),'atopos_old_root':str(pathlib.Path('/var/www/usp_app_atopos').resolve())},indent=2))
 print(json.dumps({'prepared':True,'backup':str(backup)}))
def clone(refresh):
 assert (STATE/'prepared.json').exists()
 assert not (STATE/'activated.json').exists(),'Refusing to replace an activated platform database'
 existing=table_counts(NEW_DB)
 assert not existing or refresh,'Target is not empty; explicit --refresh required before activation'
 if refresh and existing:
  sql('SET FOREIGN_KEY_CHECKS=0; '+''.join('DROP TABLE '+identifier(NEW_DB)+'.'+identifier(t)+';' for t in existing)+' SET FOREIGN_KEY_CHECKS=1;')
 now=datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%SZ');dump=STATE/('snapshot-'+now+'.sql.gz')
 args=['sudo','mysqldump','--defaults-file=/etc/mysql/debian.cnf','--single-transaction','--quick','--skip-lock-tables','--no-tablespaces','--hex-blob','--set-gtid-purged=OFF','--default-character-set=utf8mb4',DB]
 with gzip.open(dump,'wb') as out:
  proc=subprocess.Popen(args,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
  shutil.copyfileobj(proc.stdout,out)
  if proc.wait():raise RuntimeError('Snapshot failed')
 dump.chmod(0o600)
 with gzip.open(dump,'rb') as data:
  proc=subprocess.Popen(['sudo','mysql','--defaults-file=/etc/mysql/debian.cnf',NEW_DB],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
  shutil.copyfileobj(data,proc.stdin);proc.stdin.close()
  if proc.wait():raise RuntimeError('Restore failed')
 before=table_counts(DB);after=table_counts(NEW_DB);assert before==after,'Clone count mismatch'
 checks={}
 for table in before:
  result=sql('CHECKSUM TABLE '+identifier(DB)+'.'+identifier(table)+', '+identifier(NEW_DB)+'.'+identifier(table)+' EXTENDED;')
  values=[r.split('\t')[-1] for r in result];assert len(values)==2 and values[0]==values[1], 'Clone checksum mismatch: '+table
  checks[table]=values[0]
 # All historical content is preserved. Transient login/reset/push bindings must
 # not address a session/device registered through the other front.
 sql("UPDATE atopos_api.users SET token='',remember_token='',reset_password_token='',fcm_token=''; UPDATE atopos_api.files SET storage_bucket="+literal(NEW_BUCKET)+" WHERE storage_provider='s3' AND storage_bucket="+literal(OLD_BUCKET)+";")
 # Copy local attachments/templates, never logs, SQL backups, or stale temp jobs.
 storage=TARGET/'storage';storage.mkdir(exist_ok=True)
 (storage/'temp').mkdir(mode=0o700,exist_ok=True)
 for name in ['files','uploads','images','chat_templates']:
  source=SOURCE/'storage'/name
  if source.is_dir():shutil.copytree(source,storage/name,dirs_exist_ok=True)
 old='https://api.uniusp.gpedu.com.br/storage/';new='https://api.atopos.spotec.app/storage/'
 columns=sql("SELECT TABLE_NAME,COLUMN_NAME FROM information_schema.COLUMNS WHERE TABLE_SCHEMA='atopos_api' AND DATA_TYPE IN ('varchar','text','longtext','mediumtext','json');")
 for pair in columns:
  table,column=pair.split('\t');qualified='atopos_api.'+identifier(table);col=identifier(column)
  sql('UPDATE '+qualified+' SET '+col+'=REPLACE('+col+','+literal(old)+','+literal(new)+') WHERE '+col+' LIKE '+literal('%'+old+'%')+';')
 # Storage snapshot is separate from the database snapshot and contains no secrets.
 archive=STATE/('storage-'+now+'.tar.gz')
 with tarfile.open(archive,'w:gz') as tar:
  for name in ['files','uploads','images','chat_templates']:
   p=SOURCE/'storage'/name
   if p.is_dir():tar.add(p,arcname=name)
 archive.chmod(0o600)
 report={'source_database':DB,'target_database':NEW_DB,'snapshot':str(dump),'sha256':hashlib.sha256(dump.read_bytes()).hexdigest(),'counts':before,'matching_checksums_before_platform_rebinding':checks,'transient_bindings_reset':True,'snapshot_time':now}
 (STATE/'latest-clone.json').write_text(json.dumps(report,indent=2))
 print(json.dumps({'cloned':True,'tables':len(before),'records':sum(before.values()),'snapshot':str(dump)}))
p=argparse.ArgumentParser();p.add_argument('action',choices=['prepare','clone','sync-config']);p.add_argument('--refresh',action='store_true');args=p.parse_args()
os.umask(0o077)
if args.action=='prepare':prepare()
elif args.action=='sync-config':
 old=settings(SOURCE/'.env');current=settings(TARGET/'.env')
 keep=['DB_NAME','DB_USER','DB_PASSWORD','EXPECTED_DATABASE_NAME','PLATFORM_ID','API_SECRET','APP_NAME','APP_URL','PUBLIC_APP_URL','APP_PORT','API_CONTAINER_NAME','API_ALLOWED_ORIGINS','PRESENCE_ALLOWED_ORIGINS','S3_BUCKET','S3_ROLE_ARN','PLACE_SEARCH_SHARED_DIR','ALLOW_DESTRUCTIVE_LEGACY_MIGRATIONS','API_IMAGE','APP_REVISION']
 old.update({key:current[key] for key in keep if key in current})
 save_env(TARGET/'.env',old)
 print(json.dumps({'configuration_synced':True}))
else:clone(args.refresh)
