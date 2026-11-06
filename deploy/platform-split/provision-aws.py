#!/usr/bin/env python3
"""Provision Atopos DNS/private storage. No permanent access keys are created."""
import json,subprocess,urllib.parse
ACCOUNT='247433253659';REGION='sa-east-1';ZONE='Z02235572NMNQG08M2PZM';BUCKET='atopos-arquivos-'+ACCOUNT+'-'+REGION
SOURCE='educacao-onlife-arquivos-'+ACCOUNT+'-'+REGION
SOURCE_ROLE='AmazonSSMRoleForInstancesQuickSetup';ROLE='AtoposPrivateAttachments'
def aws(*args):
 r=subprocess.run(['aws',*args,'--region',REGION,'--output','json'],capture_output=True,text=True)
 if r.returncode:raise RuntimeError('AWS operation failed: '+args[0]+' '+args[1])
 return json.loads(r.stdout) if r.stdout.strip() else {}
assert aws('sts','get-caller-identity')['Account']==ACCOUNT
buckets=aws('s3api','list-buckets')['Buckets']
if not any(b['Name']==BUCKET for b in buckets):aws('s3api','create-bucket','--bucket',BUCKET,'--create-bucket-configuration',json.dumps({'LocationConstraint':REGION}))
aws('s3api','put-public-access-block','--bucket',BUCKET,'--public-access-block-configuration',json.dumps(dict(BlockPublicAcls=True,IgnorePublicAcls=True,BlockPublicPolicy=True,RestrictPublicBuckets=True)))
aws('s3api','put-bucket-ownership-controls','--bucket',BUCKET,'--ownership-controls',json.dumps({'Rules':[{'ObjectOwnership':'BucketOwnerEnforced'}]}))
aws('s3api','put-bucket-encryption','--bucket',BUCKET,'--server-side-encryption-configuration',json.dumps({'Rules':[{'ApplyServerSideEncryptionByDefault':{'SSEAlgorithm':'AES256'}}]}))
aws('s3api','put-bucket-versioning','--bucket',BUCKET,'--versioning-configuration','Status=Enabled')
aws('s3api','put-bucket-tagging','--bucket',BUCKET,'--tagging',json.dumps({'TagSet':[{'Key':'Project','Value':'Atopos'},{'Key':'Purpose','Value':'Private platform attachments'}]}))
aws('s3api','put-bucket-policy','--bucket',BUCKET,'--policy',json.dumps({'Version':'2012-10-17','Statement':[{'Sid':'DenyInsecureTransport','Effect':'Deny','Principal':'*','Action':'s3:*','Resource':['arn:aws:s3:::'+BUCKET,'arn:aws:s3:::'+BUCKET+'/*'],'Condition':{'Bool':{'aws:SecureTransport':'false'}}}]}))
principal=aws('iam','get-role','--role-name',SOURCE_ROLE)['Role']['Arn']
trust={'Version':'2012-10-17','Statement':[{'Effect':'Allow','Principal':{'AWS':principal},'Action':'sts:AssumeRole'}]}
roles=aws('iam','list-roles')['Roles']
if not any(r['RoleName']==ROLE for r in roles):aws('iam','create-role','--role-name',ROLE,'--assume-role-policy-document',json.dumps(trust))
else:aws('iam','update-assume-role-policy','--role-name',ROLE,'--policy-document',json.dumps(trust))
arn=aws('iam','get-role','--role-name',ROLE)['Role']['Arn']
aws('iam','put-role-policy','--role-name',ROLE,'--policy-name','AtoposAttachments','--policy-document',json.dumps({'Version':'2012-10-17','Statement':[{'Effect':'Allow','Action':['s3:GetObject','s3:PutObject','s3:DeleteObject'],'Resource':'arn:aws:s3:::'+BUCKET+'/files/*'}]}))
aws('iam','put-role-policy','--role-name',SOURCE_ROLE,'--policy-name','AssumeAtoposAttachments','--policy-document',json.dumps({'Version':'2012-10-17','Statement':[{'Effect':'Allow','Action':'sts:AssumeRole','Resource':arn}]}))
records=aws('route53','list-resource-record-sets','--hosted-zone-id',ZONE)['ResourceRecordSets']
name='api.atopos.spotec.app.'
for r in records:
 if r['Name']==name and r['Type']=='A':assert r.get('ResourceRecords')==[{'Value':'54.94.142.29'}], 'Unexpected existing DNS destination'
change=aws('route53','change-resource-record-sets','--hosted-zone-id',ZONE,'--change-batch',json.dumps({'Changes':[{'Action':'UPSERT','ResourceRecordSet':{'Name':name,'Type':'A','TTL':60,'ResourceRecords':[{'Value':'54.94.142.29'}]}}]}))
print(json.dumps({'bucket':BUCKET,'role':arn,'dns_change':change['ChangeInfo']['Id']}))
