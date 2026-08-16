# Learning material

Printable worksheets (A4 landscape) for German-speaking primary school kids.
Everything is Go: a small framework (`internal/sheet`), one worksheet per
directory under `generate/`, and the result lands in a mirrored layout under
`output/`.

Code, comments and the website UI are English; the worksheets themselves are
written in German.

## Layout

```
internal/sheet/               framework: registry, doc/HTML building, shared CSS, assets
internal/pdf/                 HTML -> PDF via headless Chrome (CDP, own WS client)
internal/store/               SQLite: requests / work items and settings
internal/pipeline/            drives work items through the agent
internal/site/                the front page (worksheet index + request/review UI)
generate/<subject>/<sheet>/   one worksheet (Go + render.js)
output/<subject>/<sheet>/     index.html/.pdf + solutions.html/.pdf (generated)
output/worksheets.json         runtime worksheet catalog (generated atomically)
data/                         database, worktrees, previews (not in the repo)
cmd/generate/                 builds all worksheets + the index page
cmd/serve/                    server: site, requests, work item pipeline
```

Currently: `generate/math/answer_checks`, `generate/math/least_common_multiple`,
`generate/math/metric_conversions`, `generate/math/venn_diagrams`,
`generate/math/ordinal_numbers`, and `generate/math/price_puzzles`.

## Building

```
make                          # HTML + PDF for all worksheets into output/
make html                     # HTML only (fast, no Chrome)
go run ./cmd/generate venn    # only matching worksheets (substring of the path)
make build                    # binaries into bin/
```

PDFs use `/headless-shell/headless-shell`
(override with `HEADLESS_SHELL=...`).

## Deploying

```
make serve                    # locally, using account sign-in
```

- systemd: run `make && make build`, copy `learningmaterial.service` to
  `/etc/systemd/system/`, run `sudo systemctl daemon-reload`, then
  `sudo systemctl enable --now learningmaterial`.
- For an upgrade that changes server code or its flags, run `make build`, copy
  the updated unit, run `daemon-reload`, and restart the service.
- Worksheet publication is dynamic; ordinary worksheet additions and content
  changes need none of those server deployment steps.
- Configuration: `/etc/learningmaterial/env` (`SITE_SECRET`; optional comma-separated
  `SITE_ALLOWED_EMAILS`, defaulting to `paul.hankin@pobox.com,g.n.hankin@gmail.com`;
  optional `SITE_BASE_URL` for links in account emails) — not in the repo.
- Accounts use an email address as the username. New allowlisted accounts must confirm
  their email before signing in; password-reset links are also sent by email.

## Requests and the work item pipeline

A request submitted on the front page becomes a work item that the server
drives through the Shelley agent (`shelley client`, one conversation per item):

```
queued -> working -> review -> done (approved: merged + pushed)
                           \-> rejected (branch and preview thrown away)
                           \-> refine (back to working, same conversation)
```

- Each item is developed on its own branch `req-<id>` in its own git worktree
  under `data/worktrees/`, so items never interfere with each other.
- Lanes: one lane per worksheet (plus one per new-worksheet request). Items in
  the same lane run strictly sequentially — a queued item only starts once the
  previous one has been approved or rejected. Different worksheets run in
  parallel.
- Whenever an item reaches `review`, the whole site (HTML + PDF) is rendered
  from its worktree into `data/preview/<id>/`, reachable at `/preview/<id>/`
  (admin mode only).
- Approving rebases onto `main`, fast-forward merges, pushes and regenerates
  `output/`, including an atomic `worksheets.json` catalog. The running server
  reads that catalog dynamically, so worksheet additions and content changes
  require no server rebuild or restart.
- The review UI lives in admin mode on the front page: **Approve & push**,
  **Reject**, **Refine** (free text sent back into the same conversation) and
  **Retry** for failed items, plus a log of recent pipeline events.

Server flags: `-repo`, `-work`, `-preview`, `-db`, `-push`.

## Adding a worksheet

1. Create `generate/<subject>/<name>/` and name the package.
2. In `init()` call `sheet.Register(sheet.Worksheet{Subject, Name, Title, Meta, Build})`.
3. `Build() *sheet.Doc` returns the title, extra CSS, worksheet body and
   solution body. Put pupil pages in `Doc.Body` and `sheet.SolutionPage(...)` in
   `Doc.Solutions`; they are emitted as separate documents. Building blocks:
   `sheet.Page`, `sheet.SolutionPage`, `sheet.SolutionBox`, `sheet.Lines`,
   `sheet.NameLine`; `sheet.BaseCSS` is always applied.
4. Drawings: embed `render.js` via `//go:embed`, pass data with
   `doc.Set("NAME", v)` as a JS constant, `doc.Rough = true` embeds rough.js.
5. Add a blank import in `cmd/generate/main.go`.

The index page (`output/index.html`) is built automatically from the registry.
