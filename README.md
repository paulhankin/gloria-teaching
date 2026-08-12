# Lernmaterial

Arbeitsblätter zum Ausdrucken (A4 quer). Alles in Go: ein kleines Framework
(`internal/sheet`), ein Arbeitsblatt pro Verzeichnis unter `generate/`,
das Ergebnis landet spiegelbildlich unter `output/`.

## Struktur

```
internal/sheet/             Framework: Registry, Doc/HTML-Bau, gemeinsames CSS, Assets
internal/pdf/               HTML -> PDF via headless Chrome (CDP, eigener WS-Client)
generate/<fach>/<blatt>/    ein Arbeitsblatt (Go + render.js)
output/<fach>/<blatt>/      index.html + index.pdf   (generiert, nicht im Repo)
cmd/generate/               baut alle Blätter + Startseite
cmd/serve/                  statischer Server mit Passwortschutz
```

Aktuell: `generate/mathe/venn_diagramme`, `generate/mathe/zahlenfolgen`,
`generate/mathe/preisraetsel`.

## Bauen

```
make                          # HTML + PDF für alle Blätter nach output/
make html                     # nur HTML (schnell, ohne Chrome)
go run ./cmd/generate venn    # nur passende Blätter (Teilstring im Pfad)
make build                    # Binaries nach bin/
```

Für PDFs wird `/headless-shell/headless-shell` benutzt
(überschreibbar mit `HEADLESS_SHELL=...`).

## Ausliefern

```
make serve                    # lokal, Passwort aus SITE_PASSWORD
```

- systemd: `lernmaterial.service` nach `/etc/systemd/system/`, `enable --now`.
- Konfiguration: `/etc/lernmaterial/env` (`SITE_PASSWORD`, `SITE_SECRET`) — nicht im Repo.

## Neues Arbeitsblatt anlegen

1. `generate/<fach>/<name>/` anlegen, Package benennen.
2. In `init()` `sheet.Register(sheet.Worksheet{Fach, Name, Titel, Meta, Build})`.
3. `Build() *sheet.Doc` liefert Titel, zusätzliches CSS und Body.
   Bausteine: `sheet.Page`, `sheet.SolutionPage`, `sheet.SolutionBox`,
   `sheet.Lines`, `sheet.NameZeile`; `sheet.BaseCSS` ist immer aktiv.
4. Zeichnungen: `render.js` per `//go:embed` einbinden, Daten mit `doc.Set("NAME", v)`
   als JS-Konstante übergeben, `doc.Rough = true` bettet rough.js ein.
5. Blank-Import in `cmd/generate/main.go` ergänzen.

Die Startseite (`output/index.html`) wird automatisch aus der Registry gebaut.
