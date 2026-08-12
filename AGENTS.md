# Konventionen

- **Nur Go.** Kein Python, kein Node im Build. Zeichen-Code läuft als
  `render.js` im Browser und wird per `//go:embed` eingebettet.
- Ein Arbeitsblatt = ein Verzeichnis `generate/<fach>/<name>/`, ein Go-Package,
  Anmeldung per `sheet.Register` in `init()`.
- Ausgabe immer nach `output/<fach>/<name>/index.{html,pdf}` — spiegelbildlich
  zu `generate/`. `output/` ist generiert und nicht im Repo.
- Aufgaben-Daten als Go-Structs (`aufgaben.go`), nicht als JSON-Dateien.
- Gemeinsames Layout lebt in `internal/sheet` (`BaseCSS`, `Page`,
  `SolutionPage`, ...). Blattspezifisches CSS nur für Blattspezifisches.
- Texte für die Kinder auf Deutsch (Schweizer Kontext: Franken).
  Kommentare/Doku auf Deutsch, Go-Kommentare möglichst ohne Umlaute.
- Vor dem Commit: `gofmt -l -w .`, `go build ./...`, `make html`.
