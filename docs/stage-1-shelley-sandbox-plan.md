# Stage 1 plan: filesystem-isolated Shelley worksheet jobs

## Goal

Run the Shelley agent for each worksheet request inside a dedicated bubblewrap
sandbox. The sandbox must allow the agent to edit and build a disposable copy of
the project while preventing it from reading or changing unrelated host files,
the live repositories, the primary Shelley database, and host Unix sockets.

Stage 1 deliberately does **not** isolate outbound networking. Shelley needs
network access to reach the exe.dev LLM integration, and separating that traffic
from agent-initiated traffic is deferred to Stage 2. The UI and operational
documentation must describe this as filesystem and process isolation, not as a
complete security boundary.

## Success criteria

A worksheet job must:

1. Run in a standalone core repository clone with a standalone clone of the
   requester's worksheet repository below `generate/<username>/`.
2. Run through a dedicated Shelley server, SQLite database, and Unix socket.
3. See only an explicit filesystem allowlist; in particular it must not see the
   live checkout, `/users`, the primary Shelley state, or arbitrary parts of the
   user's home directory.
4. Be able to edit both clones, commit changes, run `gofmt -l -w .`,
   `go build ./...`, and `make html`.
5. Leave all pushing, publication, and live-repository updates to the trusted
   host pipeline after the sandbox exits.
6. Survive a learning-material service restart without losing completed work or
   Shelley conversation history.
7. Be terminated as a complete process group and have CPU, memory, process, and
   runtime limits.
8. Produce enough structured logs to diagnose startup, agent, build, import,
   cleanup, and recovery failures.

## Non-goals

Stage 1 does not promise:

- restricted internet or localhost access;
- prevention of network-based exfiltration;
- prevention of direct requests to services that use ambient VM identity;
- a general-purpose sandbox API for unrelated Shelley conversations;
- interactive access to the isolated Shelley web UI;
- PDF generation inside the sandbox;
- reuse of the live repositories' linked Git worktrees.

The pipeline should remove Git remotes and unnecessary tools to reduce accidental
network effects, but these measures are not a substitute for Stage 2 network
isolation.

## Proposed directory layout

Use one persistent directory per request:

```text
data/sandboxes/req-<id>/
  workspace/                 standalone core repository clone
    generate/<username>/     standalone worksheet repository clone
  home/                      synthetic HOME visible to Shelley
    .cache/go-build/         writable Go build cache
    .config/go/              optional sandbox-local Go configuration
  state/
    shelley.db               isolated conversation database
    shelley.sock             isolated CLI socket, present only while running
    server.log               isolated Shelley server output
    metadata.json            sandbox version and lifecycle metadata
  tmp/                       optional host-backed temporary storage
```

The request ID determines the path, so recovery does not require a new database
column initially. `metadata.json` should contain at least the request ID,
username, branch name, sandbox format version, creation time, core commit,
worksheet commit, Shelley PID or unit name while active, and cleanup state.
Write it atomically.

Keep `data/sandboxes/` ignored like the existing work and preview directories.
Do not put generated previews in this directory; continue to use
`data/preview/<id>/`.

## Architecture

### Trusted host pipeline

The existing `internal/pipeline` process remains outside bubblewrap and is the
only component allowed to:

- read or modify the live core and user repositories;
- create standalone request clones;
- launch and stop the sandbox;
- communicate with isolated Shelley through its Unix socket;
- inspect and import resulting commits;
- render the trusted preview;
- merge, push, rebuild, and publish;
- delete completed or rejected request state.

### Isolated Shelley server

Each active request gets a dedicated Shelley server. Start the installed Shelley
binary inside bubblewrap with:

- a request-local database;
- a request-local Unix socket;
- `-port 0` for the unavoidable TCP listener;
- the existing exe.dev Shelley configuration mounted read-only;
- the conversation working directory set to the sandbox workspace path.

The pipeline must pass `-url unix://<request socket>` to every `shelley client`
operation. It must never rely on the default socket, because that would address
the primary Shelley server.

Use a random per-run required-header name for the isolated server and pass that
header in all client calls. This is defense in depth for Shelley's random TCP
listener; it does not provide network isolation. Do not write the header name to
the agent workspace or inject it into agent tool environments.

### Standalone Git clones

Do not expose linked worktrees to the sandbox. Their `.git` files reference the
live repositories' shared Git metadata, which would require writable host mounts.

For both repositories:

1. Clone from the local repository without shared writable metadata or hardlinks.
2. Check out a request branch based on the current `main` commit.
3. Configure a local request-specific Git identity.
4. Remove all remotes, or replace them with an invalid local URL.
5. Record the starting commit in `metadata.json` and the request database/log.
6. Verify that `git rev-parse --git-common-dir` resolves inside the request
   workspace.

The core repository is small enough that correctness and isolation are more
important than optimizing clone time. Optimizations such as read-only alternates
may be considered later only if tests prove that no writable path escapes the
sandbox.

### Import and publication

After the isolated Shelley turn ends, the trusted host pipeline must:

1. Stop the isolated Shelley server and its entire cgroup.
2. Verify that each clone is still a valid repository and remains inside the
   request directory.
3. Commit leftovers using the existing fallback behavior.
4. Require at least one clone to be ahead of its recorded starting commit.
5. Run the preview build outside the sandbox from the request workspace.
6. Under the existing publication mutex, import the resulting commits into the
   live repositories.
7. Rebase or cherry-pick them onto the then-current live `main` branches.
8. Abort cleanly on conflicts; never force-update a live branch.
9. Push and rebuild using the existing trusted publication path.

Prefer importing explicit commit IDs over accepting arbitrary refs from the
sandbox. Validate that imported commits descend from the recorded starting
commits and that their changed paths are appropriate for their repository.

## Bubblewrap policy

Build bubblewrap arguments in Go as an argument slice. Do not construct a shell
command string.

### Namespace and process options

Use at least:

```text
--unshare-user
--unshare-pid
--unshare-ipc
--unshare-uts
--unshare-cgroup-try
--disable-userns
--new-session
--die-with-parent
--cap-drop ALL
```

Keep the host network namespace in Stage 1. Do not use `--unshare-all`, because
that would unshare networking unless it is explicitly shared again.

Mount a fresh `/proc` and minimal `/dev`. Omit `/sys`, host `/run`, host `/tmp`,
DBus sockets, Docker sockets, SSH agent sockets, and the primary Shelley socket.

### Environment

Start with `--clearenv` and add only required variables, including:

- `HOME=/home/shelley`;
- a fixed minimal `PATH`;
- `GOCACHE` pointing to the writable sandbox cache;
- `GOPATH` pointing to the sandbox Go hierarchy;
- locale variables if builds require them;
- the minimum exe.dev variables verified to be required by Shelley.

Do not propagate API keys, SSH variables, Git credentials, proxy credentials,
`SHELLEY_URL`, or unrelated service environment variables.

### Read-only mounts

Mount only the runtime files required for Shelley and the worksheet build:

- `/usr`;
- `/bin`, `/lib`, `/lib64`, and `/sbin` as the corresponding symlinks;
- selected TLS certificates and DNS/NSS files from `/etc`;
- `/exe.dev/shelley.json`;
- the installed Shelley executable if it is not already covered by `/usr`;
- a read-only Go module/toolchain cache if needed;
- selected root guidance copied into the synthetic home, if desired.

Never use `--ro-bind / /`. Read-only access would still disclose host secrets.
Do not mount the real home directory or all of `/etc`.

### Writable mounts

Only these paths should be writable:

- the request workspace;
- the request's synthetic home and caches;
- request-local Shelley state;
- a private, size-limited `/tmp`.

The sandbox must not receive writable access to the live core repository, any
live `/users/<username>` repository, their `.git` directories, `output/`, the
request database, or service configuration.

### Tool availability

The sandbox needs at least:

- `bash`;
- `git`;
- `go` and `gofmt`;
- `make`;
- standard Unix text tools;
- `rg` if keyword search remains enabled.

Run a startup self-check before starting the conversation. Fail the request with
a concise diagnostic if a required executable, mount, or cache is unavailable.

## Shelley conversation policy

Create worksheet conversations with an explicit tool allowlist. The desired
initial set is:

- `bash`;
- `patch`;
- `keyword_search`;
- `change_dir`;
- `llm_one_shot` only if there is a demonstrated worksheet use case;
- `subagent` only if resource limits and accounting cover it.

Disable browser automation, output iframe, scheduling, and other tools that are
not needed to edit and build worksheets. If the current CLI cannot express all
conversation options, have the pipeline call the local Shelley HTTP API over the
Unix socket or add a narrowly scoped client flag.

JIT package installation should be disabled if possible. A read-only system
root means installation should fail safely, but avoiding the attempts provides
clearer behavior and shorter failures.

Keep the existing prompt requirements: follow `AGENTS.md`, build, commit both
repositories as appropriate, do not push, do not switch branches, and finish
with a short summary. Add a truthful sentence explaining that the workspace is
disposable and that only files below the workspace will be imported.

## Resource control

Bubblewrap is not a resource limiter. Launch it through a transient systemd
scope or equivalent cgroup owned by the service. Configure conservative initial
limits, making them settings rather than scattered constants:

- memory maximum;
- process/task maximum;
- CPU quota;
- total runtime maximum;
- graceful-stop timeout followed by cgroup kill.

Also enforce a maximum request-directory size from the trusted pipeline. Check it
periodically while polling the turn and before preview/import. A private tmpfs
should have an explicit size limit.

Add a global semaphore for active sandbox jobs. Existing worksheet lanes still
provide logical ordering, but they do not bound total resource consumption when
many different worksheets are requested at once.

## Lifecycle and recovery

Define explicit states in logs and metadata:

```text
creating -> ready -> agent-running -> agent-finished -> validating
         -> importing -> published -> cleaning -> cleaned
```

On service startup:

1. Inspect active requests and their sandbox metadata.
2. Kill any stale recorded cgroup or process before restarting work.
3. Remove a stale socket file only after confirming no server owns it.
4. Reuse the existing Shelley database and clones when they pass integrity
   checks.
5. Restart isolated Shelley and continue the stored conversation when the
   request has a conversation ID.
6. Recreate the sandbox from recorded base commits only when recovery is safe;
   otherwise mark the request failed with an actionable explanation.

Shelley SQLite state must reside on a persistent bind mount so normal service
restarts do not discard conversation history. Stop Shelley gracefully before
copying, inspecting, or deleting its database.

Completed request sandboxes may be retained for a short configurable debugging
period, then removed by trusted cleanup. Failed sandboxes should be retained
longer unless disk pressure requires removal. Rejected requests should follow the
existing product semantics while retaining enough logs to explain what happened.

## Implementation sequence

### 1. Introduce sandbox configuration and path helpers

- Add `SandboxRoot` and resource-limit settings to `pipeline.Config`.
- Add deterministic helpers for request workspace, home, state, socket, log, and
  metadata paths.
- Add validation that every derived path remains below `SandboxRoot`.
- Update service flags and `learningmaterial.service`.

### 2. Replace agent worktrees with standalone clones

- Add clone creation for core and worksheet repositories.
- Remove remotes and configure local Git identity.
- Record and validate base commits.
- Adapt commit detection and cleanup.
- Keep the existing worktree implementation temporarily behind a development
  switch only if needed for migration.

### 3. Add a sandbox launcher package

Create a small internal package responsible for:

- bubblewrap argument construction;
- environment and mount allowlists;
- startup self-checks;
- process/cgroup lifecycle;
- readiness detection through the Unix socket;
- graceful stop and forced cleanup;
- structured diagnostics.

Keep bubblewrap mechanics out of the request state machine where possible.

### 4. Route Shelley client operations

- Start one isolated server per active request.
- Add the request socket and required header to `chat`, `list`, `read`, and
  continuation calls.
- Ensure no isolated operation can fall back to the default Shelley socket.
- Persist the isolated conversation ID as today.

### 5. Adapt import and publication

- Replace shared-branch ancestry checks with recorded clone commit checks.
- Import explicit commits into live repositories.
- Rebase/cherry-pick under the publication mutex.
- Preserve atomic failure behavior across the core and worksheet repositories.
- Continue building the preview before publication and the public output after
  successful import.

### 6. Add resource controls and cleanup

- Add the global concurrency limit.
- Launch each sandbox in a transient scope.
- Monitor runtime and request-directory size.
- Guarantee whole-cgroup termination on success, failure, cancellation, and
  service shutdown.

### 7. Update operational documentation and UI wording

- Document the exact Stage 1 threat model.
- Include sandbox phase and failure details in pipeline logs.
- Avoid labels such as "fully isolated" or "secure network sandbox".
- Add troubleshooting instructions for bubblewrap, user namespaces, caches,
  cgroups, stale sockets, and retained failed workspaces.

## Test plan

### Unit tests

- All derived paths reject traversal and remain below `SandboxRoot`.
- Bubblewrap arguments contain every required namespace and mount option.
- Forbidden host paths are never added to the mount list.
- Environment construction starts clean and rejects sensitive inherited names.
- Clone validation rejects external Git common directories, symlink escapes,
  unexpected remotes, and changed base commits.
- Import validation rejects unrelated histories and non-descendant commits.
- Client commands always include the isolated socket and header.
- Lifecycle transitions and recovery decisions are deterministic.

### Integration tests

Run a predictable-model Shelley inside the real bubblewrap policy and verify:

1. It can create and modify files in both request clones.
2. It can run the required Go formatting and build commands.
3. It can commit in both repositories.
4. It cannot read known marker files placed in:
   - the live checkout;
   - `/users`;
   - the real home;
   - the primary Shelley configuration directory;
   - the service request database.
5. It cannot connect to the primary Shelley Unix socket.
6. It cannot see host processes in `/proc`.
7. It cannot write to system paths or read an omitted host path by traversal or
   symlink.
8. Killing the parent or service kills the complete sandbox cgroup.
9. Restarting the isolated server with the same database can continue a stored
   conversation.
10. Concurrent request sandboxes cannot see or modify each other's files.
11. A malicious clone containing symlinks cannot escape during build, import, or
    cleanup.
12. Resource and disk limits turn abuse into a controlled request failure.

### End-to-end tests

- Existing worksheet change: agent change, preview, import, push, and rebuild.
- New worksheet: creation in the nested repository and publication.
- Core framework plus worksheet change in one request.
- No-op agent response.
- Build failure.
- Conflicting publication after live `main` advances.
- Service restart during the Shelley turn.
- Service restart after the turn but before import.
- Rejection and cleanup.
- Two requests for different worksheets running concurrently.
- Two requests in the same lane remaining serialized.

## Rollout

1. Land the implementation disabled by default.
2. Run integration tests and several manually submitted requests on this VM.
3. Add a configuration switch selecting sandboxed execution.
4. Enable it for new requests while retaining failed sandboxes and verbose logs.
5. Monitor startup time, build time, memory, disk use, failed imports, and stale
   processes.
6. Remove the legacy agent-worktree path after a stable observation period.
7. Begin Stage 2 network-isolation work only after Stage 1 recovery and
   publication behavior is reliable.

## Acceptance checklist

Stage 1 is complete only when:

- [ ] Real Shelley runs inside bubblewrap for worksheet jobs.
- [ ] No live Git metadata is mounted into the sandbox.
- [ ] The real home, primary Shelley state, `/users`, live checkout, and service
      database are absent from the sandbox.
- [ ] Required formatting and builds succeed with the allowlisted mounts.
- [ ] Every Shelley client call targets the request-local socket explicitly.
- [ ] Resource limits and whole-cgroup cleanup are tested.
- [ ] Restart recovery is tested during both agent and publication phases.
- [ ] Commit import validates ancestry and never force-updates live branches.
- [ ] Integration tests prove cross-request and host-file isolation.
- [ ] Documentation clearly states that Stage 1 retains shared network access.
