# agent47

`agent47` is a small Go-first CLI plus template payload for bootstrapping agent-oriented repository scaffolding.

It standardizes a portable repo contract around:

- `AGENTS.md`
- `rules/*.yaml`
- curated `skills/*`
- `skills/AVAILABLE_SKILLS.xml`
- prompt helpers
- `specs/spec.yml` as a template artifact for non-trivial planning work

The public command is `afs`, short for `Agent Forty-Seven`.

## Quickstart

Install locally:

```bash
./install.sh
```

Bootstrap a target repo:

```bash
cd /path/to/project
afs add-agent
```

For automation:

```bash
./install.sh --non-interactive
```

On Windows, use:

```powershell
.\install.ps1
```

## What `add-agent` writes

`afs add-agent` bootstraps:

- `AGENTS.md`
- all template `rules/*.yaml`
- curated `skills/*`
- `skills/AVAILABLE_SKILLS.xml`
- an empty `README.md` if missing
- `specs/spec.yml` if missing

`afs add-agent --force` performs a fresh install of the managed scaffold in the current project:

- replaces `AGENTS.md`
- reconciles `rules/*.yaml`
- replaces `skills/*`
- regenerates `skills/AVAILABLE_SKILLS.xml`
- removes stale managed rules and skills no longer shipped by the current templates
- preserves `README.md`, `specs/spec.yml`, `SNAPSHOT.md`, and root `SPEC.md`

Because `rules/*.yaml` and `skills/*` are managed paths, local custom files under those paths can be replaced or removed during `--force`.

`afs add-agent --only-skills` refreshes only skills. Without `--force`, existing invalid skill files are preserved but omitted from `AVAILABLE_SKILLS.xml`.

## Public commands

```bash
afs help
afs uninstall
afs doctor
afs doctor --check-update
afs doctor --check-update-force
afs doctor --check-update --fail-on-warn
afs add-agent
afs add-agent --force
afs add-agent --only-skills
afs add-agent --only-skills --force
afs add-agent-prompt [--force]
afs add-ss-prompt
```

There is no supported public `afs install`, `afs upgrade`, `afs templates`, `afs check-update`, `afs add-spec`, `afs add-cli-prompt`, `afs add-default-skills`, or `afs init-agent` command.

## Runtime notes

- The installed product runtime is the Go CLI under `cmd/afs` and `internal/*`.
- `install.sh` and `install.ps1` are thin wrappers around the native install service.
- Repo-local `bin/afs` is a launcher for checkout-based development.
- `bin/afs` prefers an explicit compiled Go CLI via `AGENT47_GO_CLI`, then an explicit repo CLI via `AGENT47_REPO_CLI`, then `go run ./cmd/afs`. It no longer falls back to an implicit repo-root binary or to the legacy Bash runtime.
- Installed templates live under `~/.agent47/templates` on Unix-like systems and under `%LOCALAPPDATA%\agent47\templates` by default on Windows.
- On Unix-like systems, the installer publishes `~/bin/afs` plus helper commands. On Windows, it uses the managed bin directory directly.
- `doctor` flags can be combined, for example `afs doctor --check-update --fail-on-warn`.

## Repo maintenance

Primary contributor commands:

```bash
make test
make go-test
make go-build
make lint-shell
make smoke-install
```

Notes:

- `make test` runs the checkout test runner plus installed-artifact verification.
- `make go-test` runs `go test ./...` with a repo-safe `GOCACHE`.
- `make smoke-install` runs an isolated install plus `doctor` verification.
- `scripts/lint-shell` is the remaining shell maintenance entrypoint in this repo.

## Documentation

- [Usage Guide](docs/usage.md)
- [Architecture](docs/architecture.md)
- [Ownership](docs/ownership.md)
- [AGENTS.md](AGENTS.md)
- [SNAPSHOT.md](SNAPSHOT.md)
- [SPEC.md](SPEC.md)

Document roles:

- `SNAPSHOT.md`: concise current-state summary
- root `SPEC.md`: current-state product contract for `agent47`
- `specs/spec.yml`: task-specific spec/plan artifact when work needs one
