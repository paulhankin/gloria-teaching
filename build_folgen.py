#!/usr/bin/env python3
"""Baut zahlenfolgen.html: A4 quer, 2 Aufgaben (je halbe Seite) pro Blatt."""
import json, pathlib

D = pathlib.Path(__file__).parent
fonts = (D / 'assets/fonts.css').read_text()
rough = (D / 'assets/rough.js').read_text()
render = (D / 'render_folgen.js').read_text()

# ---------------------------------------------------------------- Aufgaben
A = [
    dict(key='f1', typ='raupe', kopffarbe='#ffd76a',
         titel='Aufgabe 1', name='Raupe Rosi',
         story='Raupe Rosi tr&auml;gt Zahlen auf dem R&uuml;cken. '
               'Wir z&auml;hlen immer vom <b>Kopf</b> aus!',
         zahlen=list(range(1, 11)),
         fragen=[('stelle', 3), ('stelle', 7), ('plus', 2, 5), ('minus', 9, 4)]),

    dict(key='f2', typ='zug',
         titel='Aufgabe 2', name='Der Zahlenzug',
         story='Der Zahlenzug hat viele Waggons. Der erste Waggon ist gleich '
               'hinter der <b>Lok</b>.',
         zahlen=list(range(0, 11)),
         fragen=[('stelle', 4), ('stelle', 8), ('welche_stelle', 6), ('plus', 3, 9)]),

    dict(key='f3', typ='schlange', kopffarbe='#a8e6a1',
         titel='Aufgabe 3', name='Schlange Susi',
         story='Schlange Susi z&auml;hlt in <b>Zweierschritten</b>.',
         zahlen=list(range(2, 21, 2)),
         fragen=[('stelle', 5), ('stelle', 9), ('plus', 1, 6), ('minus', 10, 3)]),

    dict(key='f4', typ='waescheleine',
         titel='Aufgabe 4', name='Die Zahlen-W&auml;scheleine',
         story='An der W&auml;scheleine h&auml;ngen Socken mit Zahlen. '
               'Wir z&auml;hlen von <b>links</b> nach rechts.',
         zahlen=list(range(5, 51, 5)),
         fragen=[('stelle', 6), ('welche_stelle', 30), ('plus', 2, 4),
                 ('minus', 8, 2)]),

    dict(key='f5', typ='drache', kopffarbe='#b6e3f5',
         titel='Aufgabe 5', name='Drache Dodo',
         story='Drache Dodo z&auml;hlt in <b>Zehnerschritten</b>.',
         zahlen=list(range(10, 101, 10)),
         fragen=[('stelle', 7), ('welche_stelle', 40), ('plus', 3, 5),
                 ('minus', 9, 6)]),

    dict(key='f6', typ='sterne',
         titel='Aufgabe 6', name='Die Sternenkette',
         story='Die Sterne z&auml;hlen <b>r&uuml;ckw&auml;rts</b> in Zweierschritten. '
               'Der erste Stern ist neben dem Mond.',
         zahlen=list(range(20, 1, -2)),
         fragen=[('stelle', 2), ('stelle', 8), ('minus', 1, 10), ('plus', 4, 7)]),
]

ORD = {1: 'erste', 2: 'zweite', 3: 'dritte', 4: 'vierte', 5: 'f&uuml;nfte',
       6: 'sechste', 7: 'siebte', 8: 'achte', 9: 'neunte', 10: 'zehnte',
       11: 'elfte'}


def frage_text(f, zahlen):
    art = f[0]
    if art == 'stelle':
        i = f[1]
        return ('Welche Zahl steht an der <b>%d. Stelle</b>?' % i,
                'Die %d. Zahl ist <b>%d</b>.' % (i, zahlen[i - 1]))
    if art == 'welche_stelle':
        w = f[1]
        i = zahlen.index(w) + 1
        return ('An welcher <b>Stelle</b> steht die Zahl <b>%d</b>?' % w,
                'Die %d steht an der <b>%d. Stelle</b>.' % (w, i))
    if art == 'plus':
        i, j = f[1], f[2]
        a, b = zahlen[i - 1], zahlen[j - 1]
        return ('Rechne: <b>%d. Zahl + %d. Zahl</b> = ?' % (i, j),
                '%d. Zahl = %d, %d. Zahl = %d &rarr; %d + %d = <b>%d</b>'
                % (i, a, j, b, a, b, a + b))
    i, j = f[1], f[2]
    a, b = zahlen[i - 1], zahlen[j - 1]
    return ('Rechne: <b>%d. Zahl &minus; %d. Zahl</b> = ?' % (i, j),
            '%d. Zahl = %d, %d. Zahl = %d &rarr; %d &minus; %d = <b>%d</b>'
            % (i, a, j, b, a, b, a - b))


folgen = {a['key']: {'typ': a['typ'], 'zahlen': [str(z) for z in a['zahlen']],
                     'kopffarbe': a.get('kopffarbe')} for a in A}

CSS = """
  @page { size: A4 landscape; margin: 10mm 12mm; }
  * { box-sizing: border-box; }
  body { font-family: 'Patrick Hand', 'Comic Sans MS', cursive;
         font-size: 15pt; line-height: 1.35; color: #1f3550; margin: 0; background: #fff; }
  .page { page-break-after: always; break-after: page;
          width: 273mm; height: 186mm; max-height: 186mm; overflow: hidden;
          display: flex; flex-direction: column; }
  .page:last-child { page-break-after: auto; }
  .kopf { display: flex; justify-content: space-between; align-items: baseline;
          padding-bottom: 1.5mm; margin-bottom: 3mm; flex: 0 0 auto;
          border-bottom: 2.5px solid #e8548c; }
  h1 { font-family: 'Caveat', cursive; font-size: 24pt; margin: 0; color: #e8548c;
       font-weight: 700; }
  .namenszeile { font-size: 11pt; color: #7a869a; }

  .haelften { display: flex; flex-direction: column; gap: 5mm; flex: 1 1 auto; min-height: 0; }
  .haelfte { flex: 1 1 50%; min-height: 0; display: flex; }
  .haelfte + .haelfte { border-top: 2px dashed #d8dfe8; padding-top: 5mm; }

  .aufgabe { width: 100%; display: flex; flex-direction: column; }
  .aufgabe h2 { font-family: 'Caveat', cursive; font-size: 21pt; font-weight: 700;
                margin: 0 0 1mm; color: #e8548c; }
  .aufgabe h2 small { font-size: 15pt; color: #7a869a; font-weight: 400; }
  .story { margin: 0 0 1mm; }
  .unten { display: flex; flex-direction: column; flex: 1 1 auto; min-height: 0; }
  .bild { flex: 1 1 auto; min-height: 0; position: relative; margin-bottom: 1mm; }
  .bild svg { position: absolute; top: 0; left: 0; width: 100%; height: 100%; }
  .text { flex: 0 0 auto; }

  ol.fragen { margin: 0; padding-left: 6mm;
              column-count: 2; column-gap: 8mm; }
  ol.fragen li { margin-bottom: 2mm; break-inside: avoid; }
  .linie { display: inline-block; border-bottom: 2px dotted #b6c0cf;
           min-width: 22mm; margin-left: 2mm; }

  .loesung { height: auto; max-height: none; overflow: visible; }
  .loesung h1 { color: #2e8b57; }
  .lspalten { column-count: 2; column-gap: 9mm; }
  .lbox { break-inside: avoid; border: 2.5px solid #bfe3cd; background: #f5fff9;
          border-radius: 10px; padding: 2.5mm 3.5mm; margin-bottom: 3.5mm; }
  .lbox h3 { font-family: 'Caveat', cursive; font-weight: 700; font-size: 18pt;
             margin: 0 0 1mm; color: #2e8b57; }
  .lbox p { margin: 0 0 1mm; font-size: 12pt; }
  .lbox .folge { font-size: 11pt; color: #5a6b7a; margin-bottom: 2mm; }
"""

NAME = ('<span class="namenszeile">Name: ____________________ &nbsp; '
        'Datum: __________</span>')


def aufgabe_html(a):
    lis = ''
    for f in a['fragen']:
        q, _ = frage_text(f, a['zahlen'])
        lis += '            <li>%s<span class="linie"></span></li>\n' % q
    return """      <div class="aufgabe">
        <h2>%s &nbsp;<small>%s</small></h2>
        <p class="story">%s</p>
        <div class="unten">
          <div class="bild" data-folge="%s"></div>
          <div class="text">
            <ol class="fragen">
%s            </ol>
          </div>
        </div>
      </div>
""" % (a['titel'], a['name'], a['story'], a['key'], lis)


pages = []
for i in range(0, len(A), 2):
    blatt = i // 2 + 1
    halves = ''
    for a in A[i:i + 2]:
        halves += '    <div class="haelfte">\n%s    </div>\n' % aufgabe_html(a)
    pages.append("""<div class="page">
  <div class="kopf"><h1>Zahlenfolgen &ndash; Blatt %d</h1>%s</div>
  <div class="haelften">
%s  </div>
</div>
""" % (blatt, NAME, halves))

lboxes = ''
for a in A:
    ps = ''
    for k, f in enumerate(a['fragen']):
        _, ans = frage_text(f, a['zahlen'])
        ps += '      <p>%s) %s</p>\n' % ('abcd'[k], ans)
    lboxes += """    <div class="lbox">
      <h3>%s &ndash; %s</h3>
      <p class="folge">Folge: %s</p>
%s    </div>
""" % (a['titel'], a['name'], ', '.join(str(z) for z in a['zahlen']), ps)

pages.append("""<div class="page loesung">
  <div class="kopf"><h1>L&ouml;sungen &#10003;</h1>
    <span class="namenszeile">f&uuml;r Eltern / Lehrerin</span></div>
  <div class="lspalten">
%s  </div>
</div>
""" % lboxes)

html = """<!DOCTYPE html>
<html lang="de">
<head>
<meta charset="utf-8">
<title>Mathe-Arbeitsblatt: Zahlenfolgen</title>
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
const FOLGEN = %s;
%s
</script>
</body>
</html>
""" % (fonts, CSS, ''.join(pages), rough, json.dumps(folgen, ensure_ascii=False), render)

(D / 'zahlenfolgen.html').write_text(html)
print('zahlenfolgen.html:', len(html) // 1024, 'KB')
