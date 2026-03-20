# agent47

`agent47` is a small CLI for bootstrapping agent-oriented project scaffolding:

- `AGENTS.md` as the policy contract
- `rules/*.yaml` for stack and security guidance
- `skills/*` plus `AVAILABLE_SKILLS.xml`
- `specs/spec.yml` for non-trivial spec/plan work
- optional helper prompts for agent-driven workflows

It is intentionally simple: copy templates into a project, keep them versioned, and refresh them explicitly when needed.

The public command is `afs`, short for `Agent Forty-Seven` (`agent47`).

In this repository, root [`SPEC.md`](SPEC.md) is a current-state product spec for `agent47` itself. It is not the place to draft a new feature spec or implementation plan; those should live in `specs/spec.yml` when needed.

## Quickstart

```bash
./install.sh

cd /path/to/project
afs add-agent
```

For non-interactive environments, use:

```bash
./install.sh --no-prompt
```

That bootstraps:

- `AGENTS.md`
- `rules/*.yaml`
- curated `skills/*` discovered from the template tree
- `skills/AVAILABLE_SKILLS.xml`
- an empty `README.md` if missing

To refresh an older project copy:

```bash
afs add-agent --force
```

`--force` reconciles agent47-managed files with the current template set, removing stale managed rules or skills while preserving `README.md`, `specs/spec.yml`, and any existing project snapshot or summary file such as `SNAPSHOT.md`.
Custom files can live under `rules/` or `skills/`, but `afs add-agent --force` may replace or remove them while reconciling the managed scaffold.

## Common commands

```bash
afs help
afs doctor
afs doctor --check-update
afs add-agent
afs add-agent --force
afs add-agent --only-skills
afs add-agent --only-skills --force
afs add-agent-prompt [--force]
afs add-ss-prompt
```

## Documentation

- [Usage Guide](docs/usage.md)
- [Architecture](docs/architecture.md)
- [Ownership](docs/ownership.md)
- [AGENTS.md](AGENTS.md)
- [SNAPSHOT.md](SNAPSHOT.md)
- [SPEC.md](SPEC.md)

The usage guide covers:

- installation and refresh flows
- managed vs project-owned files
- use with agent CLIs such as Codex or Claude Code
- use with IDEs such as VS Code or Cursor
- the difference between root `SPEC.md` and project work in `specs/spec.yml`

## Notes

- `afs add-agent` is the default bootstrap path.
- `./install.sh` now installs a stable launcher at `~/.agent47/bin/afs` and links `~/bin/afs` to that copy.
- `./install.sh --no-prompt` skips shell rc edits and is safe for automation.
- interactive installs update the preferred shell rc file for the active shell, using `~/.bash_profile` for Bash on macOS/login-style setups.
- `afs` prefers its managed helper scripts over same-named commands found earlier in `PATH`.
- `afs add-agent --only-skills` refreshes the curated skills set from whatever `templates/skills/*/SKILL.md` entries ship with the installed templates.
- during `afs add-agent --force`, `rules/*.yaml` and `skills/*` are reconciled against the current template set, so local custom files there may be replaced or removed
- `afs add-agent-prompt` and `afs add-ss-prompt` are focused helpers.
- `./scripts/test` auto-installs a temporary `bats` copy from `tests/vendor/bats` when needed and cleans it up after the run.
- `./scripts/lint-shell` runs `shellcheck` over repo Bash sources as an optional contributor check; it is not required for installing or using `agent47`.
- `./scripts/smoke-install` runs an isolated install + `doctor` pass in a temporary home directory as a release/smoke check.
- shell security guidance now ships with the template rules for Bash-first repositories and scripts.
- skill validation helpers now live under `scripts/lib/`, while `scripts/test` remains a repo-level executable entrypoint.
- Template backups keep only the latest backup when reinstalling with `--force`.
- `afs uninstall` removes the published commands plus the managed runtime assets under `~/.agent47`.
- Core scripts use strict shell mode and fail fast on copy/bootstrap errors.
