#!/usr/bin/env python3
"""Deploy exactly one API. Intended for the server/pipeline; no schema cloning."""
import argparse,json,pathlib,re,subprocess,time
p=argparse.ArgumentParser();p.add_argument('--platform',choices=['onlife','atopos'],required=True);p.add_argument('--build',action='store_true');p.add_argument('--image');a=p.parse_args()
repo=pathlib.Path('/var/www/unisinos_usp_api');target=repo if a.platform=='onlife' else pathlib.Path('/var/www/atopos_api')
rev=subprocess.check_output(['git','rev-parse','HEAD'],cwd=repo,text=True).strip();assert re.fullmatch('[0-9a-f]{40}',rev)
image=a.image or 'onlife-api:release-'+rev[:12]
assert re.fullmatch(r'[a-z0-9][a-z0-9:._/-]+',image)
if a.build:subprocess.run(['docker','build','--build-arg','APP_REVISION='+rev,'-f','docker/golang.Dockerfile','-t',image,'.'],cwd=repo,check=True)
env=target/'.env';before=env.read_text();updates={'API_IMAGE':image,'APP_REVISION':rev};lines=[]
for line in before.splitlines():
 key=line.split('=',1)[0].strip()
 if key in updates:continue
 lines.append(line)
env.write_text('\n'.join(lines)+'\n'+''.join(k+'='+v+'\n' for k,v in updates.items()));env.chmod(0o600)
service='unisinos_usp_api_go' if a.platform=='onlife' else 'api';port=8081 if a.platform=='onlife' else 8084
try:
 subprocess.run(['docker','compose','up','-d','--no-deps',service],cwd=target,check=True)
 for attempt in range(20):
  r=subprocess.run(['curl','-fsS','-D','-','http://127.0.0.1:'+str(port)+'/api/status'],capture_output=True,text=True)
  if r.returncode==0 and 'x-platform: '+a.platform in r.stdout.lower():break
  time.sleep(1)
 else:raise RuntimeError('API health/platform check failed')
except BaseException:
 env.write_text(before);subprocess.run(['docker','compose','up','-d','--no-deps',service],cwd=target,check=True);raise
print(json.dumps({'platform':a.platform,'revision':rev,'image':image,'healthy':True}))
