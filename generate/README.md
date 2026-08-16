# Worksheet build links

Worksheet repositories live at `/users/<username>/`. The import-discovery
command creates ignored symlinks here so their Go packages remain inside the
main module's import tree. Isolated request workspaces instead place the user's
Git worktree directly at `generate/<username>/`.
