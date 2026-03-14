# agent47

`agent47` is a small CLI for bootstrapping agent-oriented project scaffolding:

- `AGENTS.md` as the policy contract
- `rules/*.yaml` for stack and security guidance
- `skills/*` plus `AVAILABLE_SKILLS.xml`
- `specs/spec.yml` for non-trivial work
- optional prompts for full-agent and terminal-first workflows

It is intentionally simple: copy templates into a project, keep them versioned, and refresh them explicitly when needed.

## Quickstart

```bash
./install.sh

cd /path/to/project
a47 add-agent
```

That bootstraps:

- `AGENTS.md`
- `rules/*.yaml`
- `skills/*`
- `skills/AVAILABLE_SKILLS.xml`
- `README.md` if missing

To refresh an older project copy:

```bash
a47 add-agent --force
```

`--force` updates agent47-managed files while preserving `README.md`, `specs/spec.yml`, and any existing project snapshot or summary file such as `SNAPSHOT.md`.

## Common commands

```bash
a47 help
a47 doctor
a47 doctor --check-update
a47 add-agent [--force]
a47 add-spec
a47 add-skills [--force]
a47 reload-skills
a47 add-cli-prompt
a47 add-agent-prompt [--force]
a47 add-snapshot-prompt
```

## Documentation

- [Usage Guide](docs/usage.md)
- [Architecture](docs/architecture.md)
- [AGENTS.md](AGENTS.md)
- [SNAPSHOT.md](SNAPSHOT.md)

## Notes

- `a47 add-agent` is the default bootstrap path.
- `./install.sh` now installs a stable launcher at `~/.agent47/bin/a47` and links `~/bin/a47` to that copy.
- `a47 add-cli-prompt` copies a one-line terminal prompt to the clipboard when possible.
- `a47 add-agent-prompt` and `a47 add-snapshot-prompt` are focused helpers.
- `./scripts/test` auto-installs a temporary `bats` copy from `tests/vendor/bats` when needed and cleans it up after the run.
- Template backups keep only the latest backup when reinstalling with `--force`.
- Core scripts use strict shell mode and fail fast on copy/bootstrap errors.
