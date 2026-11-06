#!/usr/bin/env python3
"""Add the dedicated WSS proxy route to the existing API virtual host.

Run with sudo on the API host. Saves the complete previous config, validates
Apache syntax and restores the original config if validation/reload fails.
Does not change authentication, TLS, general API proxying or other vhosts.
"""
import argparse
from datetime import datetime, timezone
from pathlib import Path
import subprocess

parser=argparse.ArgumentParser(description=__doc__)
parser.add_argument('--apply',action='store_true')
args=parser.parse_args()
path=Path('/etc/apache2/sites-enabled/api.uniusp.gpedu.com.br-le-ssl.conf').resolve()
original=path.read_text()
anchor='    ProxyPass / http://127.0.0.1:8081/'
rule='    ProxyPass /api/presence ws://127.0.0.1:8081/api/presence retry=0 timeout=120\n    ProxyPassReverse /api/presence ws://127.0.0.1:8081/api/presence\n'
if 'ProxyPass /api/presence ' in original:
    print('Presence WebSocket rule already configured.');raise SystemExit(0)
if 'ServerName api.uniusp.gpedu.com.br' not in original or original.count(anchor)!=1:
    raise SystemExit('Unexpected virtual host; refusing automatic change.')
if not args.apply:
    print(rule);raise SystemExit(0)
backup=Path('/home/ubuntu/backups')/('onlife-presence-proxy-'+datetime.now(timezone.utc).strftime('%Y%m%dT%H%M%SZ')+'.conf')
backup.parent.mkdir(parents=True,exist_ok=True)
backup.write_text(original);backup.chmod(0o640)
try:
    path.write_text(original.replace(anchor,rule+anchor))
    subprocess.run(['apache2ctl','configtest'],check=True)
    subprocess.run(['systemctl','reload','apache2'],check=True)
except BaseException:
    path.write_text(original)
    subprocess.run(['apache2ctl','configtest'],check=False)
    subprocess.run(['systemctl','reload','apache2'],check=False)
    raise
print('Presence proxy configured. Backup: '+str(backup))
