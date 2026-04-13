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

Verify the local install:

```bash
afs doctor
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

## How It Works

`agent47` has two layers:

1. A local CLI runtime installed on the machine and exposed through `afs`.
2. A template payload under `templates/` that gets copied into target repos.

High-level structure:

```text
agent47/
|
+-- bin/afs              repo launcher for checkout-based development
+-- cmd/afs              native CLI entrypoint
+-- internal/            runtime packages
+-- install.sh           Unix-like install wrapper
+-- install.ps1          Windows install wrapper
+-- scripts/lint-shell   maintainer shell lint entrypoint
+-- templates/           canonical scaffold payload
+-- tests/               repo verification
+-- README.md
+-- RUNBOOK.md
+-- SNAPSHOT.md
+-- SPEC.md
`-- PLAN.md
```

Responsibility split:

- `bin/afs` is the repo-local launcher. It prefers `AGENT47_GO_CLI`, then `AGENT47_REPO_CLI`, then `go run ./cmd/afs`.
- `cmd/afs` and `internal/*` implement command routing, install/uninstall, bootstrap, doctor, update checks, prompts, manifest handling, and skills generation.
- `install.sh` and `install.ps1` are thin public wrappers over the native install flow.
- `templates/manifest.txt` defines the managed and preserved target contract.
- `templates/` is the canonical scaffold source copied into target repositories.

Operational notes:

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

- [RUNBOOK.md](RUNBOOK.md)
- [AGENTS.md](AGENTS.md)
- [SNAPSHOT.md](SNAPSHOT.md)
- [SPEC.md](SPEC.md)

Document roles:

- `README.md`: entrypoint, command surface, and high-level architecture
- `RUNBOOK.md`: operational guide for using the CLI in depth
- `SNAPSHOT.md`: concise current-state summary
- root `SPEC.md`: current-state product contract for `agent47`
- `specs/spec.yml`: task-specific spec/plan artifact when work needs one
