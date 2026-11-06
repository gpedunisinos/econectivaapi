#!/usr/bin/env python3
"""Configure S3 in the protected API runtime env after taking a server backup.
Adds only non-secret settings; AWS credentials come from the existing EC2 role.
"""
from datetime import datetime, timezone
from pathlib import Path
import os, shutil
os.umask(0o077)
root=Path('/var/www/unisinos_usp_api')
source=root/'.env'
assert source.is_file()
backup=Path('/home/ubuntu/backups')/('onlife-s3-env-'+datetime.now(timezone.utc).strftime('%Y%m%dT%H%M%SZ'))
backup.mkdir(mode=0o700)
shutil.copy2(source,backup/'api.env')
settings={'STORAGE_DRIVER':'s3','AWS_REGION':'sa-east-1','S3_BUCKET':'educacao-onlife-arquivos-247433253659-sa-east-1'}
lines=[line for line in source.read_text().splitlines() if line.partition('=')[0].strip() not in settings]
lines.extend(key+'='+value for key,value in settings.items())
# Keep this inode: Docker bind-mounts the file. Atomic replacement would leave
# a running container attached to the previous file until it is recreated.
with source.open('w') as out:
 out.write('\n'.join(lines)+'\n')
source.chmod(0o600)
print('S3 runtime settings prepared; protected env backup: '+str(backup))
