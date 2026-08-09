#!/usr/bin/env python3
"""Baut arbeitsblatt.html: A4 quer, 2 Aufgaben pro Seite, handgezeichnete Diagramme."""
import json, pathlib
D = pathlib.Path(__file__).parent
fonts = (D/'assets/fonts.css').read_text()
rough = (D/'assets/rough.js').read_text()
diagrams = (D/'diagrams.json').read_text()
content = json.load(open(D/'content.json'))
loes = json.load(open(D/'loesungen.json'))
renderer = (D/'render.js').read_text()

keys = ['a1','a2','a3','a4','a5','a6','a7']

CSS = """
  @page { size: A4 landscape; margin: 10mm 12mm; }
  * { box-sizing: border-box; }
  body { font-family: 'Patrick Hand', 'Comic Sans MS', cursive;
         font-size: 13.5pt; line-height: 1.35; color: #1f3550; margin: 0;
         background: #fff; }
  .page { page-break-after: always; width: 273mm; height: 187mm;
          display: flex; flex-direction: column; }
  .page:last-child { page-break-after: auto; }

  .kopf { display: flex; justify-content: space-between; align-items: baseline;
          padding-bottom: 1.5mm; margin-bottom: 3mm; flex: 0 0 auto;
          border-bottom: 2.5px solid #e8548c; }
  h1 { font-family: 'Caveat', cursive; font-size: 24pt; margin: 0; color: #e8548c;
       font-weight: 700; }
  .namenszeile { font-size: 11pt; color: #7a869a; }

  .spalten { display: flex; gap: 7mm; flex: 1 1 auto; min-height: 0; }
  .spalte { flex: 1 1 50%; min-width: 0; display: flex; }
  .spalte + .spalte { border-left: 2px dashed #d8dfe8; padding-left: 7mm; }

  .aufgabe { width: 100%; display: flex; flex-direction: column; }
  .aufgabe h2 { font-family: 'Caveat', cursive; font-size: 20pt; font-weight: 700;
                margin: 0 0 1.5mm; color: #e8548c; }
  .story { margin: 0 0 2mm; }
  .bild { flex: 1 1 auto; min-height: 0; display: flex; justify-content: center;
          align-items: center; margin: 1mm 0 2mm; }
  .bild svg { width: 100%; height: 100%; }
  .text { flex: 0 0 auto; }

  ol.fragen { margin: 0; padding-left: 6mm; }
  ol.fragen li { margin-bottom: 2.8mm; }
  .linie { display: inline-block; border-bottom: 2px dotted #b6c0cf;
           min-width: 24mm; margin-left: 2mm; }

  /* Loesungsblatt */
  .loesung { height: auto; }
  .loesung h1 { color: #2e8b57; }
  .lspalten { column-count: 2; column-gap: 9mm; }
  .lbox { break-inside: avoid; border: 2.5px solid #bfe3cd; background: #f5fff9;
          border-radius: 10px; padding: 2.5mm 3.5mm; margin-bottom: 3.5mm; }
  .lbox h3 { font-family: 'Caveat', cursive; font-weight: 700; font-size: 17pt;
             margin: 0 0 1mm; color: #2e8b57; }
  .lbox p { margin: 0 0 1mm; font-size: 11.5pt; }
  .rechnung { font-family: 'Courier New', monospace; font-size: 10pt; background: #fff;
              border: 1px dashed #bfe3cd; padding: 1mm 2mm; border-radius: 4px; }
"""

NAME = '<span class="namenszeile">Name: ____________________ &nbsp; Datum: __________</span>'

pages = []
for i in range(0, len(content), 2):
    blatt = i//2 + 1
    cols = ''
    for j in range(i, min(i+2, len(content))):
        c = content[j]
        fr = '\n'.join('            <li>%s</li>' % q for q in c['fragen'])
        cols += """    <div class="spalte">
      <div class="aufgabe">
        <h2>%s</h2>
        <p class="story">%s</p>
        <div class="bild" data-diagram="%s"></div>
        <div class="text">
          <ol class="fragen">
%s
          </ol>
        </div>
      </div>
    </div>
""" % (c['titel'], c['story'], keys[j], fr)
    if len(content) - i == 1:
        cols += '    <div class="spalte"></div>\n'
    pages.append("""<div class="page">
  <div class="kopf"><h1>Venn-Diagramme &ndash; Blatt %d</h1>%s</div>
  <div class="spalten">
%s  </div>
</div>
""" % (blatt, NAME, cols))

lboxes = '\n'.join('    <div class="lbox">%s</div>' % b for b in loes)
pages.append("""<div class="page loesung">
  <div class="kopf"><h1>L&ouml;sungen &#10003;</h1><span class="namenszeile">f&uuml;r Eltern / Lehrerin</span></div>
  <div class="lspalten">
%s
  </div>
</div>
""" % lboxes)

html = """<!DOCTYPE html>
<html lang="de">
<head>
<meta charset="utf-8">
<title>Mathe-Arbeitsblatt: Venn-Diagramme</title>
<style>
%s
%s
</style>
</head>
<body>
%s
<script>
%s
</script>
<script>
const DIAGRAMS = %s;
%s
</script>
</body>
</html>
""" % (fonts, CSS, ''.join(pages), rough, diagrams, renderer)

(D/'arbeitsblatt.html').write_text(html)
print('arbeitsblatt.html:', len(html)//1024, 'KB')
