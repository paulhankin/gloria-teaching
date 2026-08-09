#!/usr/bin/env python3
import urllib.parse
"""HTML -> PDF via headless-shell CDP. Usage: topdf.py <url> <out.pdf> [landscape]"""
import json, subprocess, sys, time, base64, urllib.request, shutil, os, signal
import websocket

url, out = sys.argv[1], sys.argv[2]
landscape = (len(sys.argv) < 4) or sys.argv[3].lower() not in ('0','false','portrait')
udd = '/tmp/cdp-pdf-run'
shutil.rmtree(udd, ignore_errors=True)
p = subprocess.Popen(['/headless-shell/headless-shell','--no-sandbox','--disable-dev-shm-usage',
      '--headless=old','--user-data-dir='+udd,'--remote-debugging-port=9444','--remote-allow-origins=*','about:blank'],
      stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, start_new_session=True)
try:
    for _ in range(60):
        try:
            v=json.load(urllib.request.urlopen('http://127.0.0.1:9444/json/version')); break
        except Exception: time.sleep(0.3)
    req=urllib.request.Request('http://127.0.0.1:9444/json/new?'+urllib.parse.quote(url,safe=':/?=&.-'), method='PUT')
    tgt=json.load(urllib.request.urlopen(req))
    ws=websocket.create_connection(tgt['webSocketDebuggerUrl'], timeout=60)
    i=[0]
    def cmd(m,params=None):
        i[0]+=1
        ws.send(json.dumps({'id':i[0],'method':m,'params':params or {}}))
        while True:
            r=json.loads(ws.recv())
            if r.get('id')==i[0]: return r.get('result',{})
    cmd('Page.enable'); cmd('Page.navigate',{'url':url}); time.sleep(4)
    r=cmd('Page.printToPDF',{'landscape':landscape,'printBackground':True,
          'marginTop':0,'marginBottom':0,'marginLeft':0,'marginRight':0,
          'preferCSSPageSize':True})
    open(out,'wb').write(base64.b64decode(r['data']))
    print('wrote', out, os.path.getsize(out), 'bytes')
finally:
    os.killpg(os.getpgid(p.pid), signal.SIGKILL)
