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
- `prompts/agent-prompt.txt`
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
- `prompts/agent-prompt.txt`

This preserves project-owned files:

- `README.md`
- `specs/spec.yml`
- `SNAPSHOT.md`

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

Refresh only the general prompt:

```bash
a47 add-prompt
a47 add-prompt --force
```

Get the helper prompt for manually updating `SNAPSHOT.md`:

```bash
a47 add-snapshot-prompt
```

## Operational notes

- `a47 install/upgrade` write to `~/.agent47` and `~/bin`
- `add-*` commands write to the current project directory
- `doctor` checks installed commands, templates, prompt layout, and policy structure
- templates backups are managed through:

```bash
a47 templates --list
a47 templates --restore-latest
a47 templates --clear-backups
```
