#!/usr/bin/env python3
"""Provision the private OnLIFE attachment bucket and scoped EC2 permissions.
Uses the current AWS CLI identity; never creates access keys or public grants.
"""
import json, subprocess
ACCOUNT='247433253659'
REGION='sa-east-1'
BUCKET='educacao-onlife-arquivos-'+ACCOUNT+'-'+REGION
ROLE='AmazonSSMRoleForInstancesQuickSetup'
def aws(*args):
 return subprocess.check_output(['aws',*args,'--region',REGION,'--output','json'],stderr=subprocess.PIPE)
def obj(*args):
 data=aws(*args)
 return json.loads(data) if data.strip() else {}
assert obj('sts','get-caller-identity')['Account']==ACCOUNT
owned=obj('s3api','list-buckets')['Buckets']
if not any(b['Name']==BUCKET for b in owned):
 obj('s3api','create-bucket','--bucket',BUCKET,'--create-bucket-configuration',json.dumps({'LocationConstraint':REGION}))
obj('s3api','put-public-access-block','--bucket',BUCKET,'--public-access-block-configuration',json.dumps(dict(BlockPublicAcls=True,IgnorePublicAcls=True,BlockPublicPolicy=True,RestrictPublicBuckets=True)))
obj('s3api','put-bucket-ownership-controls','--bucket',BUCKET,'--ownership-controls',json.dumps({'Rules':[{'ObjectOwnership':'BucketOwnerEnforced'}]}))
obj('s3api','put-bucket-encryption','--bucket',BUCKET,'--server-side-encryption-configuration',json.dumps({'Rules':[{'ApplyServerSideEncryptionByDefault':{'SSEAlgorithm':'AES256'}}]}))
obj('s3api','put-bucket-versioning','--bucket',BUCKET,'--versioning-configuration','Status=Enabled')
obj('s3api','put-bucket-tagging','--bucket',BUCKET,'--tagging',json.dumps({'TagSet':[{'Key':'Project','Value':'Educacao-OnLIFE'},{'Key':'Purpose','Value':'Private pista attachments'}]}))
obj('s3api','put-bucket-policy','--bucket',BUCKET,'--policy',json.dumps({'Version':'2012-10-17','Statement':[{'Sid':'DenyInsecureTransport','Effect':'Deny','Principal':'*','Action':'s3:*','Resource':['arn:aws:s3:::'+BUCKET,'arn:aws:s3:::'+BUCKET+'/*'],'Condition':{'Bool':{'aws:SecureTransport':'false'}}}]}))
obj('iam','put-role-policy','--role-name',ROLE,'--policy-name','OnLIFEPrivateAttachments','--policy-document',json.dumps({'Version':'2012-10-17','Statement':[{'Effect':'Allow','Action':['s3:PutObject','s3:GetObject','s3:DeleteObject'],'Resource':'arn:aws:s3:::'+BUCKET+'/files/*'}]}))
assert obj('s3api','get-public-access-block','--bucket',BUCKET)['PublicAccessBlockConfiguration']==dict(BlockPublicAcls=True,IgnorePublicAcls=True,BlockPublicPolicy=True,RestrictPublicBuckets=True)
assert obj('s3api','get-bucket-versioning','--bucket',BUCKET)['Status']=='Enabled'
print(json.dumps({'bucket':BUCKET,'region':REGION,'private':True,'versioning':True,'role':ROLE}))
