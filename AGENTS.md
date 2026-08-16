# Conventions

- **Go only.** No Python, no Node in the build. Drawing code runs as
  `render.js` in the browser and is embedded via `//go:embed`.
- The code, comments, identifiers, directories and the website UI are in
  **English**. Only the worksheet content itself (everything a pupil reads,
  including solutions for parents/teachers) is in **German**
  (Swiss context: Franken).
- One worksheet = one directory `generate/<username>/<name>/`, one Go package,
  registered via `sheet.Register` in `init()`. Each `generate/<username>/` is its
  own local Git repository and is ignored by the main repository. The registered
  `Subject` controls the generated output category; it is not the source directory.
- Output always goes to `output/<subject>/<name>/index.{html,pdf}` for the
  worksheet and `solutions.{html,pdf}`, using the registered subject.
- Task data lives in Go structs (`tasks.go`), not in JSON files.
- Shared layout lives in `internal/sheet` (`BaseCSS`, `Page`, `SolutionPage`,
  ...). Sheet-specific CSS only for sheet-specific things.
- Before returning: `gofmt -l -w .`, `go build ./...`, `make html`, commit
  the changes with a good message, and `git push`.
