# Lernmaterial

## Venn-Diagramme (Mathe, ~9 Jahre, Deutsch)

- `arbeitsblatt.html` — **fertiges Arbeitsblatt** (selbstenthaltend: Fonts + rough.js eingebettet).
  A4 quer, 2 Aufgaben pro Seite (je eine Spalte), 4 Aufgabenblätter + Lösungsblatt.
  Diagramme werden per rough.js handgezeichnet gerendert.
- Quellen: `content.json` (Aufgabentexte), `diagrams.json` (Diagramm-Daten),
  `loesungen.json`, `render.js` (Zeichen-Code), `build.py` (Zusammenbau).
- Neu bauen: `python3 build.py`
- `gen.html` — Vorschau nur der Diagramme.
- `venn-diagramme.html` — ältere SVG-Version (Hochformat).

Drucken: Chrome → Strg+P → Querformat, „Hintergrundgrafiken" aktivieren, Ränder „Standard".

## Zahlenfolgen (Mathe, jünger, Deutsch)

- `zahlenfolgen.pdf` / `zahlenfolgen.html` — 3 Blätter A4 quer, je 2 Aufgaben (halbe Seite),
  plus Lösungsblatt. Motive: Raupe, Zug, Schornstein-Rauch, Wäscheleine (Socken), Badewannen-Blasen, Sternenkette.
  Die Zahlenfolgen sind bewusst UNSORTIERT (zufällige Reihenfolge).
  Lernziel: „die 7. Zahl" (Ordnungszahlen / Position in einer Folge).
- Quellen: `build_folgen.py` (Aufgaben + Layout), `render_folgen.js` (Zeichnungen).
- Neu bauen: `python3 build_folgen.py && python3 topdf.py http://localhost:8000/zahlenfolgen.html zahlenfolgen.pdf`
