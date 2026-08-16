# Conventions

- **Go only.** No Python, no Node in the build. Drawing code runs as
  `render.js` in the browser and is embedded via `//go:embed`.
- The code, comments, identifiers, directories and the website UI are in
  **English**. Only the worksheet content itself (everything a pupil reads,
  including solutions for parents/teachers) is in **German**
  (Swiss context: Franken).
- One worksheet = one directory `/users/<username>/<name>/`, one Go package,
  registered via `sheet.Register` in `init()`. Each `/users/<username>/` is its
  own local Git repository and is ignored by the main repository. Build-time
  symlinks expose these packages below `generate/`. The registered `Subject`
  controls categorization; it is not part of the output path.
- Output always goes to `output/<username>/<name>/index.{html,pdf}` for the
  worksheet and `solutions.{html,pdf}`.
- Task data lives in Go structs (`tasks.go`), not in JSON files.
- Shared layout lives in `internal/sheet` (`BaseCSS`, `Page`, `SolutionPage`,
  ...). Sheet-specific CSS only for sheet-specific things.
- Before returning: `gofmt -l -w .`, `go build ./...`, `make html`, commit
  the changes with a good message, and `git push`.
