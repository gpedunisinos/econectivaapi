#!/usr/bin/env python3
"""Validate the real Jitsi auth protocol using only the isolated loopback fixture.

Requires websocket-client and the protected local meeting config. Creates only
random short-lived empty rooms, never enables media and closes every session.
Tokens and passwords are never printed. Does not access server application data.
"""
import base64,hashlib,hmac,json,secrets,time,xml.etree.ElementTree as ET,urllib.parse,urllib.request
from pathlib import Path
import websocket
creds=json.loads(Path('/tmp/onlife-integration-preview-credentials.json').read_text())
def post(path,data,auth=None):
 headers={'Content-Type':'application/json'}
 if auth:headers['Authorization']='Bearer '+auth
 r=urllib.request.Request('http://127.0.0.1:8082/api'+path,data=json.dumps(data).encode(),headers=headers)
 with urllib.request.urlopen(r,timeout=15) as res:return json.load(res)
auth=post('/login',{'email':creds['email'],'password':creds['password']})['token']
room='onlife-auth-check-'+secrets.token_hex(6)
grant=post('/me/meetings/access',{'room':room},auth)
class Client:
 def __init__(self,token=None,room_name=None):
  query={'room':room_name or room}
  if token:query['token']=token
  self.socket=websocket.create_connection('wss://meet.spotec.app/xmpp-websocket?'+urllib.parse.urlencode(query),subprotocols=['xmpp'],origin='https://meet.spotec.app',timeout=8)
  self.domain='meet.jitsi' if token else 'guest.meet.jitsi';self.nick='validation_'+secrets.token_hex(3)
  self.send(f"<open xmlns='urn:ietf:params:xml:ns:xmpp-framing' to='{self.domain}' version='1.0'/>")
  self.until('features')
  self.send("<auth xmlns='urn:ietf:params:xml:ns:xmpp-sasl' mechanism='ANONYMOUS'/>")
  answer=self.until(('success','failure'))
  if answer.tag.endswith('failure'):self.socket.close();raise ValueError('SASL rejected')
  self.send(f"<open xmlns='urn:ietf:params:xml:ns:xmpp-framing' to='{self.domain}' version='1.0'/>")
  self.until('features')
  self.send(f"<iq xmlns='jabber:client' type='set' id='bind-1'><bind xmlns='urn:ietf:params:xml:ns:xmpp-bind'><resource>{self.nick}</resource></bind></iq>")
  self.until('iq')
 def send(self,s):self.socket.send(s)
 def until(self,names):
  if isinstance(names,str):names=(names,)
  for _ in range(25):
   s=self.socket.recv()
   if not s:raise ValueError('closed')
   try:e=ET.fromstring(s)
   except ET.ParseError:continue
   if e.tag.split('}')[-1] in names:return e
  raise ValueError('expected stanza missing')
 def conference(self,room):
  self.send(f"<iq xmlns='jabber:client' type='set' to='focus.meet.jitsi' id='focus-1'><conference xmlns='http://jitsi.org/protocol/focus' room='{room}@muc.meet.jitsi' machine-uid='{self.nick}'><property name='startAudioMuted' value='0'/><property name='startVideoMuted' value='0'/></conference></iq>")
  e=self.until('iq')
  conf=e.find('{http://jitsi.org/protocol/focus}conference')
  return {'type':e.attrib.get('type'),'ready':conf.attrib.get('ready') if conf is not None else None}
 def join(self,room):
  self.send(f"<presence xmlns='jabber:client' to='{room}@muc.meet.jitsi/{self.nick}'><x xmlns='http://jabber.org/protocol/muc'/><nick xmlns='http://jabber.org/protocol/nick'>Validação técnica</nick></presence>")
  for _ in range(25):
   e=self.until('presence')
   if e.attrib.get('type')=='error':return {'error':True,'conditions':[x.tag.split('}')[-1] for x in e.iter() if x.tag.split('}')[-1] in ['not-allowed','forbidden','registration-required']]}
   if e.attrib.get('from','').endswith('/'+self.nick):
    item=e.find('.//{http://jabber.org/protocol/muc#user}item')
    return {'role':item.attrib.get('role'),'affiliation':item.attrib.get('affiliation')}
  raise ValueError('self presence missing')
 def close(self):
  try:self.send("<close xmlns='urn:ietf:params:xml:ns:xmpp-framing'/>")
  finally:self.socket.close()
clients=[]
try:
 guest=Client();clients.append(guest);guest_focus=guest.conference(room);assert guest_focus['type']=='error';before=guest.join(room)
 print('Guest before host:',before,flush=True)
 host=Client(grant['jwt']);clients.append(host)
 for attempt in range(10):
  focus=host.conference(room)
  if focus.get('ready')=='true':break
  time.sleep(.5)
 print('Conference:',focus,flush=True)
 owner=host.join(room)
 print('Authenticated host:',owner,flush=True)
 guest2=Client();clients.append(guest2);visitor=guest2.join(room)
 print('Guest after host:',visitor,flush=True)
 other_room=room+'-other'
 other_grant=post('/me/meetings/access',{'room':other_room},auth)
 other_host=Client(other_grant['jwt'],other_room);clients.append(other_host);assert other_host.conference(other_room)['ready']=='true';assert other_host.join(other_room)['role']=='moderator'
 wrong=Client(grant['jwt'],other_room);clients.append(wrong);scope=wrong.join(other_room)
 print('Token for another room:',scope,flush=True)
 bad=grant['jwt'][:-4]+'AAAA'
 try:
  invalid=Client(bad);clients.append(invalid);bad_rejected=False
 except ValueError:bad_rejected=True
 print('Invalid signature rejected:',bad_rejected,flush=True)
 # A correctly signed, expired token must also be rejected by the deployed Jitsi.
 values=dict(line.split('=',1) for line in Path(__file__).resolve().parent.parent.joinpath('.env').read_text().splitlines() if line.startswith('JITSI_APP_SECRET='))
 def b64(data):return base64.urlsafe_b64encode(data).rstrip(b'=')
 head,body,_=grant['jwt'].split('.')
 claims=json.loads(base64.urlsafe_b64decode(body+'='*(-len(body)%4)))
 claims.update(exp=int(time.time())-60,nbf=int(time.time())-120,iat=int(time.time())-120)
 unsigned=head.encode()+b'.'+b64(json.dumps(claims).encode())
 expired=(unsigned+b'.'+b64(hmac.new(values['JITSI_APP_SECRET'].encode(),unsigned,hashlib.sha256).digest())).decode()
 try:
  invalid=Client(expired);clients.append(invalid);expiry_rejected=False
 except ValueError:expiry_rejected=True
 print('Expired signed token rejected:',expiry_rejected,flush=True)
 assert before.get('error') and owner=={'role':'moderator','affiliation':'owner'} and visitor['role']!='moderator' and scope.get('error') and bad_rejected and expiry_rejected
 print('PASS: room-scoped host authentication and guest permissions verified without media.')
finally:
 for c in reversed(clients):c.close()
