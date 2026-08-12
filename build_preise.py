#!/usr/bin/env python3
"""Baut preisraetsel.html: acht Preisrätsel, je zwei A5-Aufgaben auf A4 quer."""
import json
import pathlib

D = pathlib.Path(__file__).parent
fonts = (D / "assets/fonts.css").read_text()
rough = (D / "assets/rough.js").read_text()
renderer = (D / "render_preise.js").read_text()

AUFGABEN = [
    dict(key="p1", typ="pflanze", titel="Pflanze und Topf",
         text="Eine Pflanze und ihr Topf kosten zusammen <b>11 Franken</b>. "
              "Der Topf kostet <b>3 Franken mehr</b> als die Pflanze.",
         frage="Wie viel kosten die Pflanze und der Topf einzeln?",
         namen=("Pflanze", "Topf"), werte=(4, 7)),
    dict(key="p2", typ="obst", titel="Apfel und Birne",
         text="Ein Apfel und eine Birne kosten zusammen <b>12 Franken</b>. "
              "Die Birne kostet <b>doppelt so viel</b> wie der Apfel.",
         frage="Wie viel kosten der Apfel und die Birne einzeln?",
         namen=("Apfel", "Birne"), werte=(4, 8)),
    dict(key="p3", typ="lesen", titel="Buch und Lesezeichen",
         text="Ein Buch und ein Lesezeichen kosten zusammen <b>14 Franken</b>. "
              "Das Buch kostet <b>6 Franken mehr</b> als das Lesezeichen.",
         frage="Wie viel kosten das Buch und das Lesezeichen einzeln?",
         namen=("Buch", "Lesezeichen"), werte=(10, 4)),
    dict(key="p4", typ="znüni", titel="Sandwich und Getränk",
         text="Ein Sandwich und ein Getränk kosten zusammen <b>15 Franken</b>. "
              "Das Getränk kostet <b>halb so viel</b> wie das Sandwich.",
         frage="Wie viel kosten das Sandwich und das Getränk einzeln?",
         namen=("Sandwich", "Getränk"), werte=(10, 5)),
    dict(key="p5", typ="winter", titel="Mütze und Schal",
         text="Eine Mütze und ein Schal kosten zusammen <b>22 Franken</b>. "
              "Der Schal kostet <b>2 Franken mehr</b> als die Mütze.",
         frage="Wie viel kosten die Mütze und der Schal einzeln?",
         namen=("Mütze", "Schal"), werte=(10, 12)),
    dict(key="p6", typ="spiel", titel="Ball und Springseil",
         text="Ein Ball und ein Springseil kosten zusammen <b>24 Franken</b>. "
              "Der Ball kostet <b>dreimal so viel</b> wie das Springseil.",
         frage="Wie viel kosten der Ball und das Springseil einzeln?",
         namen=("Ball", "Springseil"), werte=(18, 6)),
    dict(key="p7", typ="baeckerei", titel="Kuchen und Brot",
         text="Ein Kuchen und ein Brot kosten zusammen <b>28 Franken</b>. "
              "Das Brot kostet <b>4 Franken weniger</b> als der Kuchen.",
         frage="Wie viel kosten der Kuchen und das Brot einzeln?",
         namen=("Kuchen", "Brot"), werte=(16, 12)),
    dict(key="p8", typ="velo", titel="Velohelm und Velolicht",
         text="Ein Velohelm und ein Velolicht kosten zusammen <b>30 Franken</b>. "
              "Der Velohelm kostet <b>6 Franken mehr</b> als das Velolicht.",
         frage="Wie viel kosten der Velohelm und das Velolicht einzeln?",
         namen=("Velohelm", "Velolicht"), werte=(18, 12)),
]

CSS = r"""
  @page { size: A4 landscape; margin: 10mm 12mm; }
  * { box-sizing: border-box; }
  body { margin: 0; color: #1f3550; background: #fff;
         font: 15pt/1.35 'Patrick Hand', 'Comic Sans MS', cursive; }
  .page { width: 273mm; height: 186mm; max-height: 186mm; overflow: hidden;
          display: flex; flex-direction: column; break-after: page; page-break-after: always; }
  .page:last-child { break-after: auto; page-break-after: auto; }
  .kopf { display: flex; justify-content: space-between; align-items: baseline;
          flex: 0 0 auto; padding-bottom: 1.5mm; margin-bottom: 3mm;
          border-bottom: 2.5px solid #e8548c; }
  h1 { margin: 0; color: #e8548c; font: 700 24pt 'Caveat', cursive; }
  .namenszeile { color: #7a869a; font-size: 11pt; }
  .spalten { display: flex; gap: 7mm; flex: 1 1 auto; min-height: 0; }
  .spalte { display: flex; flex: 1 1 50%; min-width: 0; }
  .spalte + .spalte { padding-left: 7mm; border-left: 2px dashed #d8dfe8; }
  .aufgabe { display: flex; flex-direction: column; width: 100%; min-height: 0; }
  h2 { margin: 0 0 2mm; color: #e8548c; font: 700 22pt 'Caveat', cursive; }
  .story { margin: 0 0 3mm; min-height: 14mm; }
  .bild { flex: 0 0 auto; margin: 0 0 3mm; line-height: 0; }
  .bild svg { display: block; width: 100%; height: auto; }
  .frage { margin: 0 0 2.5mm; font-weight: 700; }
  .antworten { display: grid; grid-template-columns: 1fr 1fr; gap: 4mm;
               margin-bottom: 3mm; font-size: 13.5pt; }
  .antwort { white-space: nowrap; }
  .linie { display: inline-block; min-width: 24mm; margin-left: 1mm;
           border-bottom: 2px dotted #aab5c4; }
  .rechnung { flex: 1 1 auto; min-height: 24mm; padding: 2mm 3mm; color: #7a869a;
              font-size: 11pt; border: 2px dashed #d8dfe8; border-radius: 8px; }
  .loesung { height: auto; max-height: none; overflow: visible; }
  .loesung h1 { color: #2e8b57; }
  .lspalten { columns: 2; column-gap: 9mm; }
  .lbox { break-inside: avoid; margin-bottom: 3.5mm; padding: 3mm 4mm;
          border: 2.5px solid #bfe3cd; border-radius: 10px; background: #f5fff9; }
  .lbox h3 { margin: 0 0 1mm; color: #2e8b57; font: 700 18pt 'Caveat', cursive; }
  .lbox p { margin: 0 0 1mm; font-size: 12.5pt; }
  .probe { color: #65778a; font-size: 11.5pt !important; }
"""

NAME = '<span class="namenszeile">Name: ____________________ &nbsp; Datum: __________</span>'


def aufgabe_html(a, nr):
    return f"""    <div class="spalte">
      <section class="aufgabe">
        <h2>Aufgabe {nr} &nbsp; <small>{a['titel']}</small></h2>
        <p class="story">{a['text']}</p>
        <div class="bild" data-preis="{a['key']}"></div>
        <p class="frage">{a['frage']}</p>
        <div class="antworten">
          <span class="antwort">{a['namen'][0]}: <span class="linie"></span> Fr.</span>
          <span class="antwort">{a['namen'][1]}: <span class="linie"></span> Fr.</span>
        </div>
        <div class="rechnung">Rechnung:</div>
      </section>
    </div>
"""


seiten = []
for i in range(0, len(AUFGABEN), 2):
    inhalt = "".join(aufgabe_html(a, i + j + 1) for j, a in enumerate(AUFGABEN[i:i + 2]))
    seiten.append(f"""<div class="page">
  <div class="kopf"><h1>Preisrätsel – Blatt {i // 2 + 1}</h1>{NAME}</div>
  <div class="spalten">
{inhalt}  </div>
</div>
""")

loesungen = []
for nr, a in enumerate(AUFGABEN, 1):
    x, y = a["werte"]
    loesungen.append(f"""    <div class="lbox">
      <h3>Aufgabe {nr} – {a['titel']}</h3>
      <p>{a['namen'][0]}: <b>{x} Franken</b> &nbsp; · &nbsp; {a['namen'][1]}: <b>{y} Franken</b></p>
      <p class="probe">Probe: {x} + {y} = {x + y} Franken.</p>
    </div>
""")

seiten.append(f"""<div class="page loesung">
  <div class="kopf"><h1>Lösungen ✓</h1><span class="namenszeile">für Eltern / Lehrperson</span></div>
  <div class="lspalten">
{''.join(loesungen)}  </div>
</div>
""")

specs = {a["key"]: {"typ": a["typ"], "namen": a["namen"], "summe": sum(a["werte"])} for a in AUFGABEN}
html = f"""<!DOCTYPE html>
<html lang="de">
<head>
<meta charset="utf-8">
<title>Mathe-Arbeitsblatt: Preisrätsel</title>
<style>
{fonts}
{CSS}
</style>
</head>
<body>
{''.join(seiten)}
<script>
{rough}
</script>
<script>
const PREISE = {json.dumps(specs, ensure_ascii=False)};
{renderer}
</script>
</body>
</html>
"""

(D / "preisraetsel.html").write_text(html)
print("preisraetsel.html:", len(html) // 1024, "KB")
