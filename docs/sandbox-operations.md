# Sandbox operations (Stage 1: Shelley worksheet sandboxes)

This document is for the operator of the learning-material service. It
describes what the per-request Shelley sandboxes protect, how they are
wired, how to configure them, and how to diagnose the common failures.
The design rationale and test matrix live in
[stage-1-shelley-sandbox-plan.md](stage-1-shelley-sandbox-plan.md).

## Threat model (read this first)

Stage 1 provides **filesystem and process isolation per worksheet request**,
nothing more.

The Shelley agent of one worksheet request runs inside a bubblewrap sandbox
with an explicit mount allowlist. Inside the sandbox the agent process
(including anything it executes via its tools) can only see:

- the request's disposable workspace (the standalone clone of the core
  repository plus the standalone clone of the requester's worksheet
  repository);
- the request's synthetic home, Go build cache, private `/tmp`, and
  request-local Shelley state (database, socket, log);
- a read-only base system (`/usr`, selected `/etc` files, the Go toolchain
  and module cache, the exe.dev Shelley configuration).

Protected from the agent (absent from the sandbox mount namespace):

- the **live repositories** (the served core checkout and the per-user
  worksheet repositories below `/users`): the agent works on disposable
  standalone clones; only the trusted host pipeline imports commits back;
- the **`/users` tree** and the **service request database**;
- the **primary Shelley state**: the primary socket, database, and
  `~/.config/shelley` are not mounted; the agent cannot reach the primary
  Shelley server at its default address (no socket file, no `SHELLEY_URL`);
- the **real home directory** and everything else on the host: a tmpfs
  covers `/home`, and no host path outside the allowlist is mounted;
- **host processes**: a fresh PID namespace (`--unshare-pid`) plus a fresh
  `/proc` shows only sandbox processes, and `--unshare-ipc` detaches SysV
  IPC. `/proc/1/cmdline` and `/proc/1/environ` are masked so the launcher's
  command line and environment are not readable from inside.

Explicitly **NOT** protected in Stage 1:

- **Network.** The sandbox shares the host network namespace on purpose:
  Shelley needs it to reach the exe.dev LLM integration. The agent can open
  any connection the host can — the LLM integration, the worksheet Git
  service, anything reachable from the VM, including services that accept
  ambient VM identity. There is no exfiltration prevention. If host files
  must stay confidential from the LLM backend, the mount allowlist (not the
  network) is the control that protects them, because their contents cannot
  be read in the first place. Network isolation is Stage 2 work.
- The isolated server's **random TCP listener** is on the shared loopback
  and is reachable from anywhere on the host (including from inside the
  sandbox). It is guarded by a random per-run required header **name**
  (shelley 0.930 checks header presence, not a secret value). That is
  defense in depth for the TCP listener only; it is not a trust boundary.
  The pipeline's client calls use the request's Unix socket, which is
  mounted only into that request's sandbox.
- **Interactive access** to the isolated Shelley's web UI is out of scope.
- **PDF generation** runs outside the sandbox (trusted preview rebuild).

So: describe this as "each request works in a disposable isolated
workspace" — filesystem and process isolation. Do not call it "fully
isolated" or a "secure network sandbox".

## Architecture

Two components cooperate:

- **Trusted host pipeline** (`internal/pipeline`, the `cmd/serve` process):
  runs outside bubblewrap. It is the only component that touches the live
  repositories. It creates the request clones, launches and stops the
  sandbox, talks to the isolated Shelley over its Unix socket, validates
  and imports the resulting commits, renders the preview, pushes, rebuilds
  the public output, and cleans up.
- **Sandboxed Shelley** (`internal/sandbox` launcher): one dedicated
  `shelley serve` per active request inside bubblewrap, with its own
  database, Unix socket, random TCP port, and synthetic home.

Directory layout (`SandboxRoot`, default `data/sandboxes/`):

```text
data/sandboxes/req-<id>/
  workspace/                 standalone core repository clone
    generate/<username>/     standalone worksheet repository clone
  home/                      synthetic HOME (/home/shelley inside)
    .cache/go-build/         writable Go build cache (GOCACHE)
  state/
    shelley.db               isolated conversation database
    shelley.sock             isolated CLI socket (present only while running)
    server.log               isolated Shelley server output (tee'd)
    metadata.json            lifecycle metadata (written atomically)
  tmp/                       host-backed private /tmp
```

Lifecycle phases recorded in `metadata.json`:

```text
creating -> ready -> agent-running -> agent-finished -> validating
         -> importing -> published -> cleaning -> cleaned
```

`metadata.json` fields: `request_id`, `username`, `branch` (the request
branch name inside both clones), `version` (sandbox format version),
`created_at`, `core_commit` / `worksheet_commit` (recorded base commits the
import validates ancestry against), `phase`, `conversation_id`, `pid` /
`unit` (the launcher process / transient systemd scope while running),
`cleanup_state`, and `imported_core_commit` / `imported_worksheet_commit`
(the rebased tips after a successful import, used for crash-idempotent
re-publication).

Import and publication are host-side only: the pipeline fetches exactly the
recorded request branch of each clone into a temp ref of its live
repository, validates that the tip descends from the recorded base commit
(checked twice, in the clone and in the live repository), rejects core
commits that touch `generate/<username>/`, rebases both tips onto their
live mains, and only then fast-forwards both mains. Live branches are never
force-updated. The preview build and the public rebuild run outside the
sandbox.

## Configuration

Service flags (`cmd/serve`):

- `-sandbox` (default **false**): enable the sandboxed execution model.
  Without it the legacy linked-worktree mode runs.
- `-sandboxes` (default `data/sandboxes`): parent directory of the
  per-request sandboxes (`SandboxRoot`). It is made absolute at startup.
  Keep it on a filesystem with room for clones, Go build caches, and
  retained failed sandboxes, and keep it git-ignored (`/data/` is).

Resource limits live in `pipeline.SandboxLimits` (normalized by
`pipeline.New`, currently not exposed as flags — change them in
`cmd/serve/main.go` if a deployment needs different values):

| Setting             | Default | Meaning                                            |
|---------------------|---------|----------------------------------------------------|
| `MemoryMax`         | `4G`    | cgroup memory limit of one sandbox (systemd scope) |
| `TasksMax`          | `512`   | cgroup process/thread limit                        |
| `CPUQuota`          | `200%`  | cgroup CPU quota                                   |
| `RuntimeMax`        | `90m`   | hard wall clock cap of one agent turn              |
| `GracefulStop`      | `20s`   | SIGTERM grace before the cgroup/pgroup kill        |
| `WorkspaceMaxBytes` | `2 GiB` | maximum size of the whole `req-<id>` directory     |
| `MaxSandboxes`      | `2`     | global semaphore of concurrently active sandboxes  |
| `RetainCompleted`   | `24h`   | how long a done request's sandbox survives         |
| `RetainFailed`      | `7d`    | same for failed/rejected/orphaned sandboxes        |

Retention is enforced by the janitor (10-minute sweep, sandbox mode only)
and once at startup. Published requests are cleaned up synchronously at
publication time; the retention knobs cover the leftovers (failures,
rejections, orphans). The janitor never deletes a sandbox whose request is
still active.

Test/development-only hooks on `pipeline.Config` (no service flags):
`ChatExtraArgs` (extra `shelley client chat` arguments, e.g. pinning
`-model predictable`), `ShelleyPredictableOnly` (start the sandboxed
server with `-predictable-only`, so the builtin deterministic model is the
only one and no LLM integration is contacted), and `RawPrompt` (pass the
request body as the whole agent prompt — required for the predictable
model's `bash: <command>` pattern match). These exist for
`TestSandboxedPipelineEndToEnd` and must stay off in production.

## Troubleshooting

### bubblewrap missing or broken

Symptom: requests fail with `sandbox self-check failed, missing: executable
bwrap` or `bubblewrap (bwrap) not found in PATH`. Install bubblewrap on the
host (`apt install bubblewrap`). The self-check runs before every launch and
fails the request with a concise diagnostic instead of a half-started
sandbox.

### User namespaces disabled

bwrap needs unprivileged user namespaces. Check:

```sh
bwrap --ro-bind / / true                      # must print nothing and exit 0
sysctl kernel.unprivileged_userns_clone       # Debian: must be 1 (or absent)
cat /proc/sys/kernel/unprivileged_userns_clone 2>/dev/null
```

If the probe fails with `Operation not permitted`, enable user namespaces
(`sysctl kernel.unprivileged_userns_clone=1` on Debian-derived kernels) or
fix the AppArmor/sandbox restrictions that block them. The service runs
unprivileged; bwrap cannot fall back to setuid here.

### `systemd-run --user` unavailable

The launcher probes `systemd-run --user --scope` once per process. When the
user manager is missing (no logind session, container), sandboxes run in
the **process-group fallback**: no cgroup resource limits are applied, and
teardown is `kill(-pgid)` plus `--die-with-parent`. Everything else works
identically. Log/diagnostic: `sandbox.LaunchMode()` reports `systemd` or
`pgid`.

When systemd mode is active, a crashed service can leave scopes behind.
List and clean them:

```sh
systemctl --user list-units 'learningmaterial-sandbox-*'
systemctl --user stop learningmaterial-sandbox-req-<id>-<suffix>.scope
```

The startup sweep (`cleanStaleSandboxes`) normally does this for you using
the recorded metadata; manual cleanup is only needed for units created
after the last metadata write.

### Go build cache misses / slow builds

Inside the sandbox `GOCACHE` points at the writable
`home/.cache/go-build` (per request, fresh) and `GOMODCACHE` at the
read-only host module cache. The module cache is bind-mounted at the fixed
`/home/shelley/.gomodcache` — never at its host path — and its bind must
come AFTER the writable home bind in the bubblewrap arguments: mounted
earlier, the home bind shadows it and every build silently re-downloads
all modules into the per-request home (slow, and the downloads pollute the
retained sandbox). A unit test pins the order; if builds suddenly start
downloading modules, check that ordering first. exe.dev VMs pin
`GOFLAGS=-mod=mod` system-wide, so builds work against the read-only
module cache; if a build ever tries to write to the module cache it fails
fast instead of corrupting shared state. A cold build cache is expected on
the first build of each request — that is the price of a disposable
workspace. The workspace size budget (`WorkspaceMaxBytes`) includes the
build cache.

### Stale sockets

The isolated server normally removes `state/shelley.sock` when it stops
gracefully. After a crash the file can remain. Before every start the
pipeline runs `killStaleSandbox` (stops the recorded unit/pid) and only
then removes a socket file **after confirming no server owns it**. Never
delete a socket file while its server is alive; if in doubt, check the
metadata `pid`/`unit` first:

```sh
ps -p <pid> -o args=
systemctl --user status <unit>
```

### Retained failed workspaces

Failed (and rejected/orphaned) sandboxes are kept for `RetainFailed`
(default 7 days) below `data/sandboxes/req-<id>/`. To inspect one:

- `state/metadata.json` — phase, base commits, conversation id, pid/unit,
  cleanup state;
- `state/server.log` — the isolated Shelley's stdout/stderr;
- `workspace/` — the exact clone state the agent left (it is a real git
  repository; `git log`, `git status`, `git diff` all work);
- `state/shelley.db` — the isolated conversation database;
  `shelley -db state/shelley.db client list` / `read` inspect it (stop any
  recorded process first — see above).

To delete one manually:

```sh
rm -rf data/sandboxes/req-<id>
```

Only do this when the request is in a terminal state (check the request
row or the admin log); the janitor will do it on its own after the
retention period.

### Common failure messages

| Message | Meaning |
|---|---|
| `sandbox self-check failed, missing: ...` | A required executable or mount source is absent on the host; the request was not started. Fix the host, then retry. |
| `start sandboxed shelley: ...` | bwrap/systemd-run refused to launch (usually user namespaces or a broken bwrap). The sandbox output follows the message. |
| `sandboxed shelley did not become ready` | The server started but its Unix socket never answered; the captured server output is attached. Inspect `state/server.log`. |
| `the agent did not finish within ... (runtime limit)` | `RuntimeMax` expired; the isolated server was stopped (grace, then cgroup/pgroup kill) and the request failed. |
| `request #N sandbox directory exceeded its ... byte limit` | `WorkspaceMaxBytes` tripped during the turn or before the import; the request failed. Inspect before deleting. |
| `cannot recover the sandbox of request #N: ...` | After a restart the surviving clone or database failed integrity checks; the pipeline refuses to discard possibly completed work. The message names the directory to delete before retrying. |
| `the agent produced no commits` | Both clones are still at their recorded base commits; nothing to import. |
| `... does not descend from the recorded base ...` | The request branch was rewritten inside the sandbox; the import refuses it. |
| `the ... changes conflict with the current main` | Rebase conflict during import; no live branch was modified. Retry the request to rebase onto the newer main. |
| `the recorded imported tip ... is not an ancestor of main` | A live main was force-updated behind the pipeline's back (or the metadata is stale); publication refuses to continue. |

## Rollout checklist

Mirrors the plan's rollout section; the current state is noted inline.

1. [x] Land the implementation disabled by default (`-sandbox` defaults to
   false; worktree mode untouched).
2. [x] Run integration tests on this VM (`go test ./... -count=1`; the
   sandbox tests self-skip without bwrap/shelley and under `-short`).
3. [x] Add the configuration switch selecting sandboxed execution
   (`-sandbox` / `-sandboxes` flags).
4. [ ] Enable it for new requests while retaining failed sandboxes and
   verbose logs (`-sandbox`, defaults `RetainFailed=7d`,
   `RetainCompleted=24h`; pipeline events are in the admin log and the
   service journal).
5. [ ] Monitor startup time, build time, memory, disk use, failed imports,
   and stale processes (`systemctl --user list-units
   'learningmaterial-sandbox-*'`, `du -sh data/sandboxes/*`).
6. [ ] Remove the legacy agent-worktree path after a stable observation
   period (`WorkRoot`, `addWorkspace`, `removeWorktree`, `merge`,
   `hasCommits`, `branchesMerged` and their tests).
7. [ ] Begin Stage 2 network-isolation work only after Stage 1 recovery
   and publication behavior is reliable.

### Manual end-to-end verification (real LLM)

The automated end-to-end coverage uses the builtin `predictable` model
(`TestSandboxedPipelineEndToEnd`), which exercises the pipeline mechanics —
clone, sandboxed server, bash tool, commits, preview, import, push,
rebuild, cleanup — deterministically and for free. What it deliberately
does not cover is a **real LLM turn** inside the sandbox (tool-call
behavior of a real model, AGENTS.md compliance, prompt fit). Do that once
manually before enabling `-sandbox` for everyone:

1. Run the service with `-sandbox` and submit one change request and one
   new-worksheet request from the UI.
2. Watch `data/sandboxes/req-<id>/state/server.log` and the admin log.
3. Confirm the preview looks right, the import lands on both live mains,
   the push succeeds, and the sandbox directory is removed afterwards.

## Test inventory

Plan test-plan items and where they live:

- Marker-file isolation (live checkout, `/users`, real home, primary
  Shelley config, service DB path):
  `internal/sandbox` `TestIntegrationMarkerFileIsolation`.
- Primary Shelley socket unreachable inside the sandbox, `shelley client`
  default address fails: `TestIntegrationPrimaryShelleySocketUnreachable`.
- Host processes invisible: `TestIntegrationFilesystemIsolation`
  (`/proc` count, masked `/proc/1/*`) and the concurrent-sandbox test.
- Concurrent sandboxes cannot see each other:
  `TestIntegrationConcurrentSandboxes`.
- Malicious clone symlinks across agent build / import / cleanup:
  `internal/pipeline` `TestMaliciousCloneSymlinks`.
- Continue a conversation after an isolated server restart:
  `TestSandboxedShelleyRestartRecovery` (real predictable-model turns
  before and after the restart, same database).
- Whole-cgroup teardown and runtime cap: `TestIntegrationStopKillsGroup`,
  `TestIntegrationDieWithParent`, `TestSandboxRuntimeCapKillsServer`.
- Workspace size limit: `TestEnforceWorkspaceLimit`,
  `TestWaitForTurnWorkspaceLimitFailsFast`.
- Global concurrency semaphore: `TestSandboxSlotSemaphore`,
  `TestStartWaitsForSandboxSlot`.
- Import validation and crash-idempotent import:
  `sandbox_import_test.go` (`TestImportCommits*`).
- Retention and startup cleanup: `sandbox_cleanup_test.go`.
- Client routing (socket + header always set; TCP listener guarded):
  `sandbox_server_test.go`, `TestSandboxedShelleyServer`.
- Full pipeline mechanics with the predictable model:
  `TestSandboxedPipelineEndToEnd`.

Not automated (manual rollout steps): a real-LLM end-to-end run inside the
sandbox, and a service-level restart-during-turn test (killing the actual
`cmd/serve` process mid-turn; the pieces — stale-process kill, socket
cleanup, conversation continuation — are covered individually).
