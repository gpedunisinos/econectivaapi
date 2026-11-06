#!/usr/bin/env python3
"""Post-cutover checks with disposable accounts/data. No secrets in output.
Run on the host after activating both isolated APIs. Removes only its own data.
"""
import base64,datetime,hashlib,io,json,os,pathlib,secrets,subprocess,time,urllib.error,urllib.parse,urllib.request,uuid
ROOT=pathlib.Path('/home/ubuntu/backups/platform-split');ROOT.mkdir(exist_ok=True,mode=0o700)
marker='split-check-'+uuid.uuid4().hex
report={'marker':marker,'platforms':{},'checks':[]}
platforms=[{'id':'onlife','db':'unisinos_usp_api','base':'https://api.uniusp.gpedu.com.br','origin':'https://educacaoonlife.gpedu.com.br','bucket':'educacao-onlife-arquivos-247433253659-sa-east-1'}, {'id':'atopos','db':'atopos_api','base':'https://api.atopos.spotec.app','origin':'https://plataforma.atopos.spotec.app','bucket':'atopos-arquivos-247433253659-sa-east-1'}]
def sql(s):
 r=subprocess.run(['sudo','mysql','--defaults-file=/etc/mysql/debian.cnf','--batch','--raw','--skip-column-names'],input=s.encode(),capture_output=True)
 if r.returncode:raise RuntimeError('Fixture SQL failed')
 return r.stdout.decode().splitlines()
def lit(v):return 'CONVERT(0x'+v.encode().hex()+' USING utf8mb4) COLLATE utf8mb4_unicode_ci'
def request(p,method,path,body=None,token=None,status=200):
 headers={'Origin':p['origin'],'Accept':'application/json'}
 if token:headers['Authorization']='Bearer '+token
 data=None
 if body is not None:data=json.dumps(body).encode();headers['Content-Type']='application/json'
 req=urllib.request.Request(p['base']+'/api'+path,data=data,headers=headers,method=method)
 try:
  with urllib.request.urlopen(req,timeout=100) as r:code=r.status;data=r.read();head=r.headers
 except urllib.error.HTTPError as e:code=e.code;data=e.read();head=e.headers
 if code!=status:raise RuntimeError(p['id']+' '+method+' '+path.split('?')[0]+' returned '+str(code)+' expected '+str(status))
 return data,head
def api(p,method,path,body=None,token=None,status=200):
 data,_=request(p,method,path,body,token,status)
 return json.loads(data) if data else None
def listed(p,path):return api(p,'GET',path,token=p['tokens'][0])
def check(condition,message):
 if not condition:raise RuntimeError(message)
 report['checks'].append(message)
def cleanup():
 for p in platforms:
  db=p['db'];ids=p.get('user_ids',[])
  for file in p.get('created_files',[]):
   try:api(p,'DELETE','/files/'+str(file),token=p['tokens'][0])
   except Exception:pass
  if not ids:continue
  users=','.join(map(str,ids));clues=','.join(map(str,p.get('created_clues',[]))) or '0';nodes=','.join(map(str,p.get('created_nodes',[]))) or '0';templates=','.join(map(str,p.get('created_templates',[]))) or '0'
  # These predicates contain only IDs allocated to this private test run.
  sql('USE '+db+'; DELETE m FROM participant_messages m JOIN participant_conversations c ON c.id=m.conversation_id WHERE c.user_a IN ('+users+') OR c.user_b IN ('+users+'); DELETE FROM participant_conversations WHERE user_a IN ('+users+') OR user_b IN ('+users+'); DELETE FROM ai_messages WHERE user_id IN ('+users+'); DELETE FROM chat_messages WHERE user_id IN ('+users+'); DELETE FROM living_clue_chat_messages WHERE user_id IN ('+users+'); DELETE FROM clue_edits WHERE user_id IN ('+users+') OR living_clue_id IN ('+clues+'); DELETE FROM environmental_contexts WHERE living_clue_id IN ('+clues+'); DELETE FROM clue_preview_references WHERE living_clue_id IN ('+clues+'); DELETE FROM sensor_readings WHERE living_clue_id IN ('+clues+'); DELETE FROM living_clues WHERE id IN ('+clues+'); DELETE FROM nodes WHERE id IN ('+nodes+'); DELETE FROM chat_templates WHERE id IN ('+templates+'); DELETE FROM users WHERE id IN ('+users+');')
try:
 for p in platforms:
  p.update({'user_ids':[],'tokens':[],'created_files':[],'created_clues':[],'created_nodes':[],'created_templates':[]})
  for number in range(2):
   password=secrets.token_urlsafe(24);email=marker+'-'+p['id']+'-'+str(number)+'@example.test'
   digest=subprocess.check_output(['htpasswd','-nbBC','10','test',password],stderr=subprocess.DEVNULL).decode().split(':',1)[1].strip()
   user=int(sql('INSERT INTO '+p['db']+'.users (created_at,updated_at,name,email,password,profile,usage_mode,email_verified_at) VALUES(NOW(),NOW(),'+lit('Teste'+p['id']+str(number))+','+lit(email)+','+lit(digest)+',40,\'inventor\',NOW()); SELECT LAST_INSERT_ID();')[0]);p['user_ids'].append(user)
   auth=api(p,'POST','/login',{'email':email,'password':password});check(auth.get('id')==user and bool(auth.get('token')),p['id']+' password login');p['tokens'].append(auth['token'])
  status,headers=request(p,'GET','/status');check(headers.get('X-Platform')==p['id'],p['id']+' public HTTPS and platform header')
  api(p,'GET','/living_clues',status=401)
 for i,p in enumerate(platforms):
  api(p,'GET','/living_clues',token=platforms[1-i]['tokens'][0],status=401)
  check(True,p['id']+' rejects other platform JWT')
  wrong=p.copy();wrong['origin']=platforms[1-i]['origin'];request(wrong,'GET','/living_clues',token=p['tokens'][0],status=403)
  categories=listed(p,'/categories');nodes=listed(p,'/node_types');check(bool(categories) and bool(nodes),p['id']+' administrative catalogs')
  layers=listed(p,'/environmental_layers');check(bool(layers),p['id']+' shared Geo API catalog')
  base=next((l for l in layers if l.get('available') and l.get('map_role')=='basemap'),None)
  if base:
   tile,head=request(p,'GET','/environmental_tiles/'+base['id']+'/3/3/4.'+base.get('tile_format','png'),token=p['tokens'][0]);check(len(tile)>100 and head.get('Content-Type','').startswith('image/'),p['id']+' geographic map tile')
  clue=api(p,'POST','/living_clues',{'name':marker+'-'+p['id'],'category_id':categories[0]['id'],'description':'Validação técnica temporária da separação.','is_active':True},p['tokens'][0]);p['created_clues'].append(clue['id'])
  api(p,'POST','/living_clues/'+str(clue['id'])+'/sensor_readings',{'validacao':marker,'dados_ficticios':True},status=201)
  detail=listed(p,'/living_clues/'+str(clue['id']));check(detail.get('name')==marker+'-'+p['id'] and len(detail.get('sensor_readings',[]))>0,p['id']+' clue create/detail and sensor ingestion')
  node=api(p,'POST','/nodes',{'name':marker+'-'+p['id'],'color':'#21B6A8','node_type_id':nodes[0]['id']},p['tokens'][0]);p['created_nodes'].append(node['id'])
  agent=api(p,'POST','/chat_templates',{'name':marker+'-'+p['id'],'description':'Validação técnica temporária','system_instruction':'Responda apenas OK para este teste técnico.'},p['tokens'][0]);p['created_templates'].append(agent['id'])
  file=api(p,'POST','/files',{'name':marker+'.txt','extension':'txt','base_64':base64.b64encode(marker.encode()).decode(),'living_clue_id':clue['id']},p['tokens'][0]);p['created_files'].append(file['id'])
  grant=listed(p,'/files/'+str(file['id'])+'/access');check(p['bucket'] in urllib.parse.urlparse(grant['url']).hostname,p['id']+' exclusive S3 bucket')
  with urllib.request.urlopen(grant['url'],timeout=20) as r:check(r.read()==marker.encode(),p['id']+' attachment upload and signed download')
  # Names are unique across both DBs, even when copied IDs coincide.
  for resource in ['living_clues','nodes','chat_templates']:
   other_name=marker+'-'+platforms[1-i]['id'];items=listed(p,'/'+resource+'?search='+urllib.parse.quote(marker)+'&limit=200');check(not any(v.get('name')==other_name for v in items),p['id']+' isolated '+resource)
  meeting=api(p,'POST','/me/meetings/access',{'room':'validacao-separacao'},p['tokens'][0]);check(meeting.get('room')==p['id']+'-validacao-separacao',p['id']+' separate meeting namespace')
  report['platforms'][p['id']]={'healthy':True,'created_and_read_clue':True,'file_round_trip':True}
 # Read the cloned real attachment without emitting its bytes or signed URL.
 for p in platforms:
  for item in listed(p,'/files'):
   if item.get('name')==marker+'.txt' or item.get('storage_provider')!='s3':continue
   grant=listed(p,'/files/'+str(item['id'])+'/access')
   with urllib.request.urlopen(grant['url'],timeout=20) as r:check(r.status==200,p['id']+' historical attachment preserved')
   break
 config=ROOT/(marker+'-sessions.json');config.write_text(json.dumps(platforms));config.chmod(0o600)
 try:
  proc=subprocess.run(['/dev/shm/platform-presence-check',str(config)],capture_output=True,text=True,timeout=90)
  if proc.returncode:raise RuntimeError('Real WebSocket/chat isolation checks failed')
  report['presence_and_chat']=json.loads(proc.stdout)
 finally:config.unlink(missing_ok=True)
 report['passed']=True
finally:
 cleanup()
 report['fixtures_removed']=True
 path=ROOT/'smoke.json';path.write_text(json.dumps(report,indent=2));path.chmod(0o600)
print(json.dumps({'passed':report.get('passed',False),'checks':len(report['checks']),'fixtures_removed':True,'report':str(path)}))
