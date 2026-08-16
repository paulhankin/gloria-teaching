# Worksheet build links

Worksheet repositories are separate, minimal Git repositories. In production
they live at `/users/<username>/`; an isolated builder may clone them anywhere
and pass their parent directory to `go run ./cmd/generate -users <dir>`.

The generator creates ignored symlinks here so each worksheet package is
compiled as part of the core Go module. User repositories therefore contain no
`go.mod`, shared fonts, framework libraries, or generated output. Isolated
request workspaces may instead place a user's Git worktree directly at
`generate/<username>/`.
