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
users/<username>/             represented on disk as /users/<username>; local Git repository
  <sheet>/                    one worksheet package (Go + render.js)
generate/<username>/          ignored build-time symlink or isolated request worktree
output/<username>/<sheet>/    generated index.html/.pdf + solutions.html/.pdf
output/worksheets.json         runtime worksheet catalog (generated atomically)
data/                         database, worktrees, previews (not in the repo)
cmd/generate/                 builds all worksheets + the index page
cmd/serve/                    server: site, requests, work item pipeline
```

The worksheet source is deliberately not stored in the main GitHub repository.
Each `/users/<username>/` directory is an independent local repository; for
example, the existing worksheets live in `/users/gloriahankin/`. The generator
discovers these repositories, creates ignored package symlinks under
`generate/`, and writes an ignored import file before compiling itself.

## User repository contract

A user's repository is intentionally only worksheet source:

```
<worksheet-name>/
  worksheet.go
  tasks.go          # optional
  render.js         # optional, embedded by worksheet.go
```

It has no `go.mod`, vendored Go packages, fonts, framework code, or generated
files. Those all belong to this core repository. Each worksheet imports
`learningmaterial/internal/sheet`; the build-time symlink makes the separately
cloned source part of this Go module.

For an isolated build, clone user repositories below one parent directory (the
last path component is the username), check out this repository separately,
and run:

```
go run ./cmd/generate -users /workspace/users
```

For example, `/workspace/users/alice/fractions/worksheet.go` produces
`output/alice/fractions/index.pdf` and `solutions.pdf`. The `-users` flag may
point at any parent directory, so the user repositories need not be inside the
core checkout. Pushing user changes and exporting generated PDFs are deliberately
outside the generator's responsibility.

## Building

```
make                          # HTML + PDF for all worksheets into output/
make html                     # HTML only (fast, no Chrome)
go run ./cmd/generate venn    # only matching worksheets (substring)
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
- Accounts have a unique lower-case username and a verified email address. Usernames may
  contain Unicode letters, numbers and hyphens; users can sign in with either their username
  or email address. New allowlisted accounts must confirm their email before signing in;
  password-reset links are also sent by email.
- Worksheets have a persistent owner and are private by default. Owners can mark them
  public or share private worksheets with existing users using view/edit grants. A user's
  published worksheets are listed at `/worksheets/<username>/index`; worksheet pages at
  `/worksheets/<username>/sheet/<name>` are available to the owner, signed-in users for
  public worksheets, and explicitly shared users for private worksheets. Existing worksheets
  are assigned to `g.n.hankin@gmail.com`, while newly requested worksheets belong to their requester.

## Requests and the work item pipeline

A request submitted on the front page becomes a work item that the server
drives through the Shelley agent (`shelley client`, one conversation per item):

```
queued -> working -> review -> done (approved: merged + pushed)
                           \-> rejected (branch and preview thrown away)
                           \-> refine (back to working, same conversation)
```

- Each item gets an isolated main-repository worktree under `data/worktrees/`
  plus a nested worktree of the requester's local repository at
  `generate/<username>/`. Framework changes and worksheet changes are therefore
  versioned independently.
- Lanes: one lane per worksheet (plus one per new-worksheet request). Items in
  the same lane run strictly sequentially — a queued item only starts once the
  previous one has been approved or rejected. Different worksheets run in
  parallel.
- Whenever an item reaches `review`, the whole site (HTML + PDF) is rendered
  from its worktree into `data/preview/<id>/`, reachable at `/preview/<id>/`
  (admin mode only).
- Approving rebases and fast-forward merges both request branches, pushes main-repository
  changes to its `origin`, pushes worksheet changes to
  `https://worksheet-gits.exe.xyz/user/<username>.git`, and regenerates `output/`, including
  an atomic `worksheets.json` catalog. The running server reads that catalog dynamically,
  so worksheet additions and content changes require no server rebuild or restart.
- The review UI lives in admin mode on the front page: **Approve & push**,
  **Reject**, **Refine** (free text sent back into the same conversation) and
  **Retry** for failed items, plus a log of recent pipeline events.

Server flags: `-repo`, `-users`, `-work`, `-preview`, `-db`, `-push`.

## Adding a worksheet

1. Ensure `/users/<username>/` is a Git repository on branch `main` (the
   request pipeline creates it automatically), then create `<name>/` inside it.
2. In `init()` call `sheet.Register(sheet.Worksheet{Username, Subject, Name, Title, Meta, Build})`.
3. `Build() *sheet.Doc` returns the title, extra CSS, worksheet body and
   solution body. Put pupil pages in `Doc.Body` and `sheet.SolutionPage(...)` in
   `Doc.Solutions`; they are emitted as separate documents. Building blocks:
   `sheet.Page`, `sheet.SolutionPage`, `sheet.SolutionBox`, `sheet.Lines`,
   `sheet.NameLine`; `sheet.BaseCSS` is always applied.
4. Drawings: embed `render.js` via `//go:embed`, pass data with
   `doc.Set("NAME", v)` as a JS constant, `doc.Rough = true` embeds rough.js.
5. Run `make html`; worksheet packages are discovered automatically.

The index page (`output/index.html`) is built automatically from the registry.
