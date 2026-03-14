# agent47

`agent47` is a small CLI for bootstrapping agent-oriented project scaffolding:

- `AGENTS.md` as the policy contract
- `rules/*.yaml` for stack and security guidance
- `skills/*` plus `AVAILABLE_SKILLS.xml`
- `specs/spec.yml` for non-trivial work
- optional helper prompts for agent-driven workflows

It is intentionally simple: copy templates into a project, keep them versioned, and refresh them explicitly when needed.

## Quickstart

```bash
./install.sh

cd /path/to/project
a47 add-agent
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
a47 add-agent
a47 add-agent --force
a47 add-agent --only-skills
a47 add-agent --only-skills --force
a47 add-agent-prompt [--force]
a47 add-snapshot-prompt
```

## Documentation

- [Usage Guide](docs/usage.md)
- [Architecture](docs/architecture.md)
- [AGENTS.md](AGENTS.md)
- [SNAPSHOT.md](SNAPSHOT.md)

The usage guide covers:

- installation and refresh flows
- managed vs project-owned files
- use with agent CLIs such as Codex or Claude Code
- use with IDEs such as VS Code or Cursor

## Notes

- `a47 add-agent` is the default bootstrap path.
- `./install.sh` now installs a stable launcher at `~/.agent47/bin/a47` and links `~/bin/a47` to that copy.
- `./install.sh --no-prompt` skips shell rc edits and is safe for automation.
- `a47` prefers its managed helper scripts over same-named commands found earlier in `PATH`.
- `a47 add-agent --only-skills` refreshes the curated skills set from whatever `templates/skills/*/SKILL.md` entries ship with the installed templates.
- `a47 add-agent-prompt` and `a47 add-snapshot-prompt` are focused helpers.
- `./scripts/test` auto-installs a temporary `bats` copy from `tests/vendor/bats` when needed and cleans it up after the run.
- Template backups keep only the latest backup when reinstalling with `--force`.
- Core scripts use strict shell mode and fail fast on copy/bootstrap errors.
