# Conventions

- **Go only.** No Python, no Node in the build. Drawing code runs as
  `render.js` in the browser and is embedded via `//go:embed`.
- The code, comments, identifiers, directories and the website UI are in
  **English**. Only the worksheet content itself (everything a pupil reads,
  including solutions for parents/teachers) is in **German**
  (Swiss context: Franken).
- One worksheet = one directory `generate/<subject>/<name>/`, one Go package,
  registered via `sheet.Register` in `init()`.
- Output always goes to `output/<subject>/<name>/index.{html,pdf}` — mirroring
  `generate/`. `output/` is generated and not in the repo.
- Task data lives in Go structs (`tasks.go`), not in JSON files.
- Shared layout lives in `internal/sheet` (`BaseCSS`, `Page`, `SolutionPage`,
  ...). Sheet-specific CSS only for sheet-specific things.
- Before committing: `gofmt -l -w .`, `go build ./...`, `make html`.
