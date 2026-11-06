#!/usr/bin/env python3
"""Prepare JWT SSO configuration on the existing host without restarting services.

Preserves both protected environments and the meeting proxy config privately.
Does not change database data, accounts, ports or non-Jitsi virtual hosts.
"""
import json, os, re, secrets, shutil, subprocess
from datetime import datetime, timezone
from pathlib import Path
import urllib.request

os.umask(0o077)
base=Path('/opt/onlife-jitsi')
api=Path('/var/www/unisinos_usp_api')
request=urllib.request.Request('http://127.0.0.1:8182/metrics',headers={'Accept':'text/plain'})
with urllib.request.urlopen(request,timeout=10) as response:
    metrics=response.read().decode()
for gauge in ['jitsi_jvb_conferences','jitsi_jvb_current_endpoints']:
    match=re.search(r'^'+gauge+r' ([0-9.eE+-]+)$',metrics,re.M)
    if not match or float(match.group(1)) != 0:
        raise SystemExit('Meetings are active or health cannot be confirmed; no configuration changed')
backup=Path('/home/ubuntu/backups')/('onlife-meeting-jwt-'+datetime.now(timezone.utc).strftime('%Y%m%dT%H%M%SZ'))
backup.mkdir(mode=0o700)
for path,name in [(base/'.env','jitsi.env'),(api/'.env','api.env')]:
    shutil.copyfile(path,backup/name)
    (backup/name).chmod(0o600)
proxy=Path('/etc/apache2/sites-available/meet.spotec.app-ssl.conf')
subprocess.run(['sudo','cp',str(proxy),str(backup/'meet-ssl.conf')],check=True)
subprocess.run(['sudo','chmod','600',str(backup/'meet-ssl.conf')],check=True)
subprocess.run(['sudo','tar','-czf',str(backup/'jitsi-config.tgz'),'-C',str(base),'config','compose.override.yml'],check=True)
subprocess.run(['sudo','chmod','600',str(backup/'jitsi-config.tgz')],check=True)

def read_env(path):
    return dict(line.split('=',1) for line in path.read_text().splitlines() if re.match(r'^[A-Z][A-Z0-9_]*=',line))

def update(path,values):
    lines=[l for l in path.read_text().splitlines() if l.partition('=')[0] not in values]
    path.write_text('\n'.join(lines)+'\n'+''.join(k+'='+v+'\n' for k,v in values.items()))
    path.chmod(0o600)

previous=read_env(base/'.env')
# Preserve an existing SSO secret on subsequent runs; never rotate silently.
secret=previous.get('JWT_APP_SECRET') or secrets.token_hex(48)
assert len(secret)>=32
update(base/'.env',{'ENABLE_AUTH':'1','AUTH_TYPE':'jwt','ENABLE_GUESTS':'1','JWT_APP_ID':'onlife','JWT_APP_SECRET':secret,'JWT_ACCEPTED_ISSUERS':'onlife','JWT_ACCEPTED_AUDIENCES':'onlife','JWT_ALLOW_EMPTY':'0','JWT_ENABLE_DOMAIN_VERIFICATION':'1','JWT_SIGN_TYPE':'HS256','ENABLE_AUTO_OWNER':'0','WAIT_FOR_HOST_DISABLE_AUTO_OWNERS':'1',
'XMPP_MUC_MODULES':','.join(dict.fromkeys(filter(None,previous.get('XMPP_MUC_MODULES','').split(',')+['token_affiliation']))),
'XMPP_MUC_CONFIGURATION':'token_verification_require_token_for_moderation = true,token_verification_allowlist = { "guest.meet.jitsi" }',
'ENABLE_AUTO_LOGIN':'0'})
update(api/'.env',{'JITSI_PUBLIC_URL':'https://meet.spotec.app','JITSI_APP_ID':'onlife','JITSI_APP_SECRET':secret,'JITSI_XMPP_DOMAIN':'meet.jitsi'})
# Query strings can carry the short-lived meeting credential; exclude them
# and Referer from this virtual host's access log and outgoing navigation.
text=proxy.read_text().replace('"strict-origin-when-cross-origin"','"no-referrer"')
text=text.replace('CustomLog ${APACHE_LOG_DIR}/meet.spotec.app-ssl-access.log combined',
'''CustomLog ${APACHE_LOG_DIR}/meet.spotec.app-ssl-access.log "%a %t \\"%m %U %H\\" %>s %b"''')
assert 'no-referrer' in text and ' combined' not in text
staged=backup/'meet-new.conf';staged.write_text(text)
subprocess.run(['sudo','cp',str(staged),str(proxy)],check=True)
check=subprocess.run(['sudo','apache2ctl','configtest'],capture_output=True)
if check.returncode:
    subprocess.run(['sudo','cp',str(backup/'meet-ssl.conf'),str(proxy)],check=True)
    shutil.copyfile(backup/'api.env',api/'.env');shutil.copyfile(backup/'jitsi.env',base/'.env')
    raise SystemExit('Proxy validation failed; previous configuration restored')
subprocess.run(['sudo','systemctl','reload','apache2'],check=True)
print(json.dumps({'backup':str(backup),'prepared':True,'auth':'jwt','guest_access':True,'rooms_active':0,'services_restarted':False}))
