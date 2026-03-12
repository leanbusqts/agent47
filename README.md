```
 █████╗  ██████╗ ███████╗███╗   ██╗████████╗██╗  ██╗███████╗
██╔══██╗██╔════╝ ██╔════╝████╗  ██║╚══██╔══╝██║  ██║╚════██║
███████║██║  ███╗█████╗  ██╔██╗ ██║   ██║   ███████║    ██╔╝
██╔══██║██║   ██║██╔══╝  ██║╚██╗██║   ██║   ╚════██║   ██╔╝ 
██║  ██║╚██████╔╝███████╗██║ ╚████║   ██║        ██║   ██║  
╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═╝  ╚═══╝   ╚═╝        ╚═╝   ╚═╝  
```

# agent47

`agent47` is a lightweight CLI to bootstrap and manage **AI Agent–driven development workflows**, with support for:

- AGENTS.md–based agent guidance
- Spec Driven Development (SDD)
- Skills-based agent execution
- Reusable prompts
- Consistent project initialization
- Security rules split into global, language, and stack layers
- Templates inspired by agents.md (contracts), Anthropic/agentskills (skills format), a simplified Spec Kit pattern (spec.yml), and aligned with community resources (https://github.com/anthropics/skills, https://agents.md, https://github.com/github/spec-kit/blob/main/spec-driven.md, https://agentskills.io/home)

It is designed to be **simple, explicit, and composable**, without hidden automation.

---

## What agent47 is for

- Bootstrap agent workflows with explicit contracts (AGENTS, stack rules, skills, specs, prompts) so agents act consistently.
- Keep templates and conventions versioned; copy them into projects when you decide.
- Stay language-agnostic: no generators, no hidden automation—just scaffolding and guidance.

---

## Philosophy

- **agent47 does not manage your project**
- **agent47 provides templates and conventions**
- You decide *when* and *how* to apply specs, skills, and prompts

## Context Access Policy (MCP-aligned)

All templates and prompts enforce explicit context boundaries:
- Use only files, resources, and data explicitly provided in the prompt/workspace/approved context.
- Do not assume access to files not listed, undeclared APIs, or external services not explicitly enabled.
- If required context is missing, stop and ask.

This keeps agent runs auditable and prevents accidental access to unintended resources.

---

## Requirements

- Unix-like environment (macOS / Linux)
- Bash
- `~/bin` included in your `$PATH`

---

## Quickstart

```bash
# Install CLI + templates into ~/.agent47 and link to ~/bin
./install.sh

# In your project
cd /path/to/project

# Bootstrap agent scaffolding
a47 add-agent --with-skills --prompt

# Optional: add spec or refresh skills/prompts later
a47 add-spec                    # creates specs/spec.yml if missing
a47 add-skills                  # rerun to refresh skills/AVAILABLE_SKILLS.xml after edits
a47 add-prompt                  # single general prompt
a47 add-snapshot-prompt         # copies/prints a prompt to manually refresh SNAPSHOT.md
```

### What you get
- `AGENTS.md` in the project root
- `rules-*.yaml` under `rules/`
- `rules/security-*.yaml` for shared security policy
  - `security-java-kotlin.yaml` now applies to backend and Android/mobile work
  - `security-swift.yaml` applies to iOS/mobile work
  - `security-csharp.yaml` applies to backend and MAUI/Xamarin-style mobile work
- `specs/spec.yml` (fill it in; includes optional plan/tasks/log scaffold)
- `skills/<name>/SKILL.md` (behavior contracts) and `skills/AVAILABLE_SKILLS.xml`
- `prompts/agent-prompt.txt`

### Prompt workflows
- The CLI ships one general prompt at `templates/prompts/agent-prompt.txt`.
- That prompt references `AGENTS.md`, rules, skills metadata, and `specs/spec.yml` without duplicating policy.
- `snapshot-prompt.txt` is separate and only meant to help manually refresh `SNAPSHOT.md`.

### Spec template (plan before code)
- `specs/spec.yml` follows the Spec Kit format and accepts optional nodes `plan`, `tasks`, `log`.
- Use those nodes to persist decisions, checklist, and log; complete or confirm before implementing.
- No extra files are created; everything lives in the same spec.

---

## Skills overview (curated set)

- analyze: understand current state, flows, and issues before changes.
- implement: deliver scoped changes to requirements/spec with minimal surface.
- review: inspect changes for correctness, risks, regressions.
- refactor: improve structure without changing behavior.
- optimize: improve performance/resources with evidence.
- plan: create concise plans with risks and checkpoints.
- spec-clarify: ask targeted questions to clarify scope and edge cases.
- troubleshoot: isolate root causes and propose targeted fixes.

---

## Installation

### Quick install (recommended)

From the repository root:

```bash
chmod +x install.sh
./install.sh            # safe (does not overwrite existing templates/scripts)
./install.sh --force    # overwrite existing templates/scripts (backs up templates)
```

This command will:

- Make the CLI executable
- Install helper commands into ~/bin
- Install templates into ~/.agent47
- Link a47 into your PATH (via ~/bin)
- Verify the installation

After installation, verify:

```bash
a47 doctor
```

During installation, `a47` is installed as the entrypoint (symlinked into ~/bin).

Note: install.sh is the recommended entry point.
Manual steps are provided only for troubleshooting or advanced usage.

---

## Usage
All examples below use `a47`. Use flags to include skills and prompts in one step if desired.

## Command cheatsheet (common)
- `a47 add-agent [--with-skills] [--prompt]`
- `a47 add-spec`
- `a47 add-skills [--force]` / `a47 reload-skills`
- `a47 install [--force]` / `a47 upgrade [--force]` / `a47 uninstall`
- `a47 add-prompt`
- `a47 add-snapshot-prompt`

### Show help

```bash
a47 help
```

### Templates backups

```bash
a47 templates --restore-latest    # restore templates from latest backup
a47 templates --list              # list available template backups
a47 templates --clear-backups     # remove all template backups
```

Backups live in `~/.agent47/templates.bak.<timestamp>`, created automatically by `a47 install/upgrade` (only the latest is kept).

Notes:
- `a47 install/upgrade` require write access to `$HOME/.agent47`.
- `add-*` commands require write access to the current directory and fail-fast if required templates are missing.
- Core helper scripts use strict shell mode to reduce partial-state failures on copy/bootstrap paths.
- CLI internals are split between `bin/a47` as router and `scripts/lib/` as sourceable implementation modules.

### Check installation

```bash
a47 doctor
```

This verifies:

* `a47` availability
* installed helper commands
* prompt and security templates
* required `AGENTS.md` sections
* current version
* update check (shows warnings if git/curl or network are unavailable; does not block)

### Initialize agent (AGENTS + rules) with optional skills and prompt

```bash
a47 add-agent --with-skills --prompt
```

- `--with-skills` copies the curated skills and generates `skills/AVAILABLE_SKILLS.xml`.
- `--prompt` copies the single general prompt. Omit it to skip prompts.
- Security templates are copied into `rules/` together with stack rules.

### Check for updates (manual, cached)

```bash
a47 check-update          # uses cache (24h)
a47 check-update --force  # bypass cache
```

`a47 doctor` also runs the check once per invocation and reports the result without updating anything.

```bash
a47 doctor

a47 add-spec

a47 add-prompt
```

Note: `a47` is a real executable installed by the CLI.
It is not a shell alias and requires no shell configuration.

---

## Project Initialization

### Initialize agent environment in a project

From inside a project directory:

```bash
a47 init-agent
```

This will copy:

* `AGENTS.md`
* `rules/*.yaml`
* create `README.md` (if missing)

No specs, skills, or prompts are created automatically.

---

## Optional Add-ons (Manual, Explicit)

You opt-in to each component.

### Add a base spec

```bash
a47 add-spec
```

Creates:

```text
specs/spec.yml
```

---

### Add skills

```bash
a47 add-skills [--force]
```

Creates:

```text
skills/
├── analyze/
│   └── SKILL.md
├── implement/
│   └── SKILL.md
├── review/
│   └── SKILL.md
├── refactor/
│   └── SKILL.md
├── optimize/
│   └── SKILL.md
├── plan/
│   └── SKILL.md
├── spec-clarify/
│   └── SKILL.md
└── troubleshoot/
    └── SKILL.md
```

- Without `--force`, existing skills are preserved and only missing skills/`AVAILABLE_SKILLS.xml` are generated. Use `--force` to overwrite skills from templates.

---

### Add the agent prompt

```bash
a47 add-prompt
```

Creates:

```text
prompts/agent-prompt.txt
```

Usage notes:
- Uses skills if present through `skills/AVAILABLE_SKILLS.xml`.
- Uses `specs/spec.yml` for non-trivial work.
- Keeps policy out of the prompt and points back to `AGENTS.md`.

---

### Agent Skills format and validation

- Skills follow the Agent Skills spec (`SKILL.md` with YAML frontmatter `name` + `description`, body in Markdown). Reference: https://agentskills.io/specification
- Optional folders (`scripts/`, `references/`, `assets/`) are not created by default. Add them only if needed, keep files small, and reference them with relative paths from `SKILL.md` (progressive disclosure).
- `a47 add-skills` also generates `skills/AVAILABLE_SKILLS.xml` with the `<available_skills>` block (name, description, location). Prompts instruct the agent to read this file directly for filesystem-based activation; no manual pasting required.
- If you add or edit skills later, rerun `a47 add-skills` to refresh `skills/AVAILABLE_SKILLS.xml`.
- Alternatively, run `a47 reload-skills` to regenerate only `skills/AVAILABLE_SKILLS.xml` without copying templates.
- To validate a skill after editing/creating it: `skills-ref validate skills/<skill>` (optional, requires the `skills-ref` tool).

---

## Upgrade / Repair

If you update the `agent47` repository or want to repair the installation:

```bash
a47 install [--force]
```

This safely reinstalls helper commands and templates.
Run this after pulling repo changes to propagate updated templates (e.g., new `skills/*.md`) into `~/.agent47`.
Use `--force` to overwrite existing templates/scripts (backs up templates before overwriting); without the flag, existing files are left intact.

---

## Uninstall

To remove the installed helper commands from your system:

```bash
a47 uninstall
```

This does **not** delete:

* projects
* specs
* prompts
* templates
* the `agent47` repository itself

---

## Versioning

`agent47` follows **Semantic Versioning (SemVer)**.

The current version is stored in:

```text
VERSION
```

You can check it with:

```bash
a47 help
a47 doctor
```

---

## Directory Structure (agent47)

```text
agent47/
├── bin/
│   └── a47
├── scripts/
│   ├── add-agent
│   ├── add-prompt
│   ├── add-skills
│   ├── add-spec
│   ├── lib/
│   └── add-snapshot-prompt
├── templates/
│   ├── AGENTS.md
│   ├── rules/
│   │   ├── rules-backend.yaml
│   │   ├── rules-frontend.yaml
│   │   └── rules-mobile.yaml
│   ├── prompts/
│   ├── specs/
│   └── skills/
└── VERSION
```

---

## What agent47 Does NOT Do

* It does not run agents
* It does not modify code automatically
* It does not hide decisions
* It does not enforce tools or vendors

`agent47` is intentionally minimal.

---

## Intended Workflow (High Level)

```text
./install.sh
↓
cd project
↓
a47 add-agent --with-skills --prompt
↓
a47 add-spec (optional)
↓
a47 add-skills (optional, to refresh skills/templates)
↓
a47 reload-skills (optional, regenerate skills/AVAILABLE_SKILLS.xml only)
↓
a47 add-prompt (optional, if not added via flags)
↓
Use your AI tool of choice
```

### Testing

- Run the suite: `make test` (uses `scripts/test`; prefers `tests/vendor/bats/bin/bats` and falls back to `bats` on PATH).
- Cleanup: temp dirs live under `/tmp/a47-test-*` and are auto-removed; use `make clean-test` to remove leftovers from interrupted runs.
- Bats install: vendor `bats-core` into `tests/vendor/bats` or install it system-wide and ensure `bats` is on PATH.
- Vendored deps with embedded `.git` (e.g., bats): run `make vendor-clean` to remove the nested repo before committing.

### Notes

#### Moving the agent47 repository

If you move the `agent47` repository after installation, the CLI command may stop working.
This is expected behavior when using symbolic links.
To fix it, recreate the symlink:

```bash
cd /new/path/to/agent47
ln -sf "$(pwd)/bin/a47" ~/bin/a47
```

Alternatively, rerun the installer:

```bash
./install.sh
```

---

## License / Usage

Internal tooling / personal workflow.
Adapt freely to your needs.

---
