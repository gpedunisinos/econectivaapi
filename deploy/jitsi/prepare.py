#!/usr/bin/env python3
"""Prepare the pinned official Jitsi release and an isolated protected config.

Does not start containers, publish ports, change Apache, DNS or create people.
Run as the server's uid 1000 after creating an owned target directory.
"""
import argparse
import hashlib
import ipaddress
import json
import os
from pathlib import Path
import secrets
import shutil
import urllib.request
import zipfile
from io import BytesIO

RELEASE='stable-11248'
URL='https://api.github.com/repos/jitsi/docker-jitsi-meet/zipball/stable-11248'
SHA256='bee9f07f893b63332b2d30294bc6c55aace98ffa035118e922872db0fd851fcd'
parser=argparse.ArgumentParser()
parser.add_argument('--directory',type=Path,default=Path('/opt/onlife-jitsi'))
parser.add_argument('--public-ip',required=True)
parser.add_argument('--archive',type=Path,help='Optional already-downloaded official ZIP; checksum is still mandatory')
args=parser.parse_args()
address=ipaddress.ip_address(args.public_ip)
if address.version!=4 or not address.is_global:parser.error('Use the server public IPv4 address')
if os.getuid()!=1000:parser.error('Run as uid 1000 so rootless container storage has the correct owner')
base=args.directory.resolve()
base.mkdir(parents=True,exist_ok=True,mode=0o750)
if (base/'.env').exists():parser.error('Protected config already exists; refusing to replace secrets')
if args.archive: data=args.archive.read_bytes()
else:
    with urllib.request.urlopen(URL,timeout=60) as response:data=response.read(20*1024*1024)
if hashlib.sha256(data).hexdigest()!=SHA256:parser.error('Official archive checksum differs; review release source before proceeding')
release=base/'release';release.mkdir(exist_ok=True,mode=0o750)
archive=zipfile.ZipFile(BytesIO(data));prefix=archive.namelist()[0].split('/')[0]
for filename in ['docker-compose.yml','env.example','LICENSE']:
    source=prefix+'/'+filename
    if source in archive.namelist():(release/filename).write_bytes(archive.read(source))
for filename in ['compose.override.yml','meet.spotec.app.conf','meet.spotec.app-ssl.conf']:
    shutil.copyfile(Path(__file__).parent/filename,base/filename)
settings=(Path(__file__).parent/'env.example').read_text().replace('/opt/onlife-jitsi/config',str(base/'config')).replace('PUBLIC_IPV4',str(address))
for name in ['JICOFO_AUTH_PASSWORD','JVB_AUTH_PASSWORD']:
    settings=settings.replace(name+'=\n',name+'='+secrets.token_hex(32)+'\n')
fd=os.open(base/'.env',os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600)
with os.fdopen(fd,'w') as output:output.write(settings)
for path in ['web','prosody/config','prosody/prosody-plugins-custom','jicofo','jvb','storage/prosody','storage/transcripts','storage/web','tmp/web-crontabs','tmp/web-load-test']:
    target=base/'config'/path;target.mkdir(parents=True,exist_ok=True,mode=0o750)
    target.chmod(0o750)
manifest={'release':RELEASE,'source':URL,'archive_sha256':SHA256,'archive_prefix':prefix,'public_url':'https://meet.spotec.app','recording':False,'transcription':False,'authentication':'internal-secure-domain','guest_access':True}
(base/'release-manifest.json').write_text(json.dumps(manifest,indent=2)+'\n')
print('Pinned Jitsi files and isolated configuration prepared; secrets were not printed. Containers, DNS and Apache have not been changed.')
