# Usage Guide

## Install

Recommended:

```bash
./install.sh
```

Force reinstall:

```bash
./install.sh --force
```

Verify:

```bash
a47 doctor
a47 doctor --check-update
```

## Bootstrap a project

Inside the target project:

```bash
a47 add-agent
```

This copies:

- `AGENTS.md`
- all `rules/*.yaml`
- all curated `skills/*`
- `skills/AVAILABLE_SKILLS.xml`
- `README.md` if missing

## Update an older project

If the project already has an older agent47 setup:

```bash
a47 add-agent --force
```

This refreshes managed files:

- `AGENTS.md`
- `rules/*.yaml`
- `skills/*`
- `skills/AVAILABLE_SKILLS.xml`

This preserves project-owned files:

- `README.md`
- `specs/spec.yml`
- any existing project snapshot or summary file such as `SNAPSHOT.md`

## Other commands

Add a spec scaffold:

```bash
a47 add-spec
```

Refresh only skills:

```bash
a47 add-skills
a47 add-skills --force
a47 reload-skills
```

Copy a minimal terminal prompt to the clipboard:

```bash
a47 add-cli-prompt
```

Refresh only the general prompt:

```bash
a47 add-agent-prompt [--force]
```

Get the helper prompt for manually updating a project snapshot or summary file, such as `SNAPSHOT.md`:

```bash
a47 add-snapshot-prompt
```

## Operational notes

- `a47 install/upgrade` write to `~/.agent47` and `~/bin`
- `add-*` commands write to the current project directory
- `doctor` checks installed commands, templates, prompt layout, and policy structure
- `doctor` skips update checks by default; use `a47 doctor --check-update` to include them
- `./scripts/test` auto-installs a temporary `bats` copy from `tests/vendor/bats` when needed
- template backups keep only the latest backup when reinstalling with `--force`
- templates backups are managed through:

```bash
a47 templates --list
a47 templates --restore-latest
a47 templates --clear-backups
```
