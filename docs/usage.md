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

Non-interactive install:

```bash
./install.sh --non-interactive
```

Verify:

```bash
afs doctor
afs doctor --check-update
./scripts/lint-shell
./scripts/smoke-install
```

`./install.sh` is the only public installation entrypoint.
It installs the managed launcher under `~/.agent47/bin/afs` and links `~/bin/afs` to that copy.
`--non-interactive` skips interactive shell rc edits and is intended for automation.

There is no supported `afs install` or `afs upgrade` command.

## First steps

1. Install the tool locally:

```bash
./install.sh
```

2. Verify the local setup:

```bash
afs doctor
```

3. Enter the target project and bootstrap it:

```bash
cd /path/to/project
afs add-agent
```

## Bootstrap a project

Inside the target project:

```bash
afs add-agent
```

This copies:

- `AGENTS.md`
- all `rules/*.yaml`
- all curated `skills/*` discovered from the installed template tree
- `skills/AVAILABLE_SKILLS.xml`
- and creates an empty `README.md` if missing

## Update an older project

If the project already has an older agent47 setup:

```bash
afs add-agent --force
```

This reconciles managed files against the current template set:

- `AGENTS.md`
- `rules/*.yaml`
- `skills/*`
- `skills/AVAILABLE_SKILLS.xml`

Managed files no longer shipped by the current templates are removed during `--force`.
That includes custom files you may have added under managed paths such as `rules/` or `skills/`.

This preserves project-owned files:

- `README.md`
- `specs/spec.yml`
- `SNAPSHOT.md`
- `SPEC.md`

## Managed files

By default, `agent47` manages these project files:

- `AGENTS.md`
- `rules/*.yaml`
- `skills/*`
- `skills/AVAILABLE_SKILLS.xml`

`afs add-agent --force` refreshes those managed files.
During `--force`, those paths are reconciled against the current template set.

`agent47` does not overwrite these project-owned files during the normal refresh flow:

- `README.md`
- `specs/spec.yml`
- `SNAPSHOT.md`
- `SPEC.md`

If you keep project-specific extensions under `rules/` or `skills/`, expect to reapply them after a forced refresh when necessary.

## Other commands

Refresh only skills:

```bash
afs add-agent --only-skills
afs add-agent --only-skills --force
```

This mode only updates `skills/*` and `skills/AVAILABLE_SKILLS.xml`.
The skill set is derived from whichever `templates/skills/*/SKILL.md` entries are installed under `~/.agent47`.
It does not touch `AGENTS.md` or `rules/*.yaml`.
With `--force`, local custom files under `skills/` are still replaceable because the directory is managed as a whole.

Refresh only the general prompt:

```bash
afs add-agent-prompt [--force]
```

Get the helper prompt for manually generating or updating project state documents such as `SNAPSHOT.md` and `SPEC.md`:

```bash
afs add-ss-prompt
```

Canonical document roles live in `README.md`:

- `SNAPSHOT.md`
- root `SPEC.md`
- `specs/spec.yml`

## Which command to use

- Install the tool on your machine: `./install.sh`
- Verify the local install: `afs doctor`
- Bootstrap a repo with the default scaffolding: `afs add-agent`
- Refresh an existing repo-managed setup: `afs add-agent --force`
- Add or refresh only the default curated skills: `afs add-agent --only-skills [--force]`
- Create or refine a spec/plan through the agent in `specs/spec.yml`
- Treat root `SPEC.md` as the current-state product spec for `agent47`, not as the default file for a new feature spec or plan

## Operational notes

- `./install.sh` writes to `~/.agent47` and `~/bin`
- `./install.sh --non-interactive` avoids interactive shell rc prompts when `~/bin` is not yet on `PATH`; the installer also behaves this way automatically when no TTY is available
- interactive installs write the PATH export to the preferred shell rc file for the active shell, using `~/.bash_profile` for Bash on macOS/login-style setups
- `add-*` commands write to the current project directory
- `doctor` checks installed commands, templates, prompt layout, and policy structure
- `doctor` skips update checks by default; use `afs doctor --check-update` to include them
- `./scripts/test` auto-installs a temporary `bats` copy from `tests/vendor/bats` when needed
- `./scripts/lint-shell` runs `shellcheck` against repo Bash sources as an optional maintainer/contributor check; users of `agent47` do not need it
- `./scripts/smoke-install` runs an isolated install plus `afs doctor` as a smoke/release check
- reinstalling without `--force` preserves existing installed commands and launcher links; use `--force` when you intend to refresh managed runtime files
- template backups keep only the latest backup when reinstalling with `--force`
- if you need to recover templates manually, copy the latest `~/.agent47/templates.bak.*` back over `~/.agent47/templates`
- `afs` resolves managed helper scripts before falling back to same-named commands on `PATH`
- `afs uninstall` removes both the published commands in `~/bin` and the managed runtime assets under `~/.agent47`

## Use with agent CLIs

`agent47` does not depend on a specific CLI runtime.
It provides a repository contract that CLIs such as Codex or Claude Code can follow.

In practice:

- tell the agent to read `AGENTS.md` first if it does not discover it automatically
- let `AGENTS.md` drive the next reads; it already defines authority order and required inputs
- for template-source repositories such as `agent47` itself, read `templates/rules/` when the policy points to `rules/`
- use `specs/spec.yml` for non-trivial work when the workflow needs a written spec, plan, or task log
- if the user asks for a spec or plan, let the agent build or refine `specs/spec.yml` through conversation
- once the draft exists, suggest that the user review it before implementation starts
- if the runtime supports multi-agent or sub-agent execution and the task matters enough, prefer an independent spec/plan review
- use `skills/AVAILABLE_SKILLS.xml` only when the workflow is actually using skills

Recommended terminal-first workflow:

1. Open the repository in the CLI.
2. Ask the agent to read `AGENTS.md`.
3. Ask it to inspect the relevant `rules/*.yaml` and code/tests for the task.
4. Use `specs/spec.yml` only when the task is non-trivial, plan-driven, or spec-driven.

The important point is that `AGENTS.md` remains the authority.
The CLI should adapt to the repository contract, not the other way around.

Minimal text for tools that support persistent instructions:

```text
Read AGENTS.md first and follow the applicable rules before making changes.
```

Use that only when the tool does not reliably discover `AGENTS.md` on its own.
Prefer local user settings or explicitly approved vendor-specific files over adding extra repo-level config by default.

## Use with IDEs

`agent47` is a repository convention, not an IDE integration layer.
That applies to VS Code, Cursor, Windsurf, and similar editors with embedded agents.

Recommended usage:

- open the repository root, not an isolated subfolder
- make `AGENTS.md` visible early in the session
- tell the IDE agent to use `AGENTS.md` as the repository policy if it does not do so automatically
- keep vendor-specific agent config files out of the repo unless they were explicitly requested

This is the intended mental model:

- `agent47` defines portable repository rules
- the IDE agent should be pointed at those rules
- the repo should not depend on vendor-specific config to remain usable
