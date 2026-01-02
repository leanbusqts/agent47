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
- Abstract handling of rules and memories (authoritative order: user > AGENTS > stack rules > specs > code/tests > memories/hints)
- Templates inspired by agents.md (contracts), Anthropic/agentskills (skills format), and a simplified Spec Kit pattern (spec.yml)

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

The goal is to make agent-driven workflows:
- repeatable
- inspectable
- versionable

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

# Bootstrap agent scaffolding (copies AGENTS.md and rules-*.yaml)
agent47 init-agent

# Optional: add spec, skills, and prompts
agent47 add-spec
agent47 add-skills
agent47 add-agent-prompt        # general prompt
agent47 add-agent-prompt-ss     # spec + skills prompt
```

What you get:
- `AGENTS.md` and `rules-*.yaml` in the project root
- `specs/spec.yml` (fill it in)
- `skills/*.md` (behavior contracts)
- `prompts/agent-prompt*.txt` (edit before running your agent)

---

## Installation

### Quick install (recommended)

From the repository root:

```bash
chmod +x install.sh
./install.sh
```

This command will:

- Make the CLI executable
- Install helper commands into ~/bin
- Install templates into ~/.agent47
- Link agent47 into your PATH (via ~/bin)
- Verify the installation

After installation, verify:

```bash
agent47 doctor
```

During installation, both `agent47` and `a47` commands are installed.
`a47` is a shorthand wrapper for `agent47`.

Note: install.sh is the recommended entry point.
Manual steps are provided only for troubleshooting or advanced usage.

---

## Usage
All examples below use `agent47`, but `a47` works identically.

### Show help

```bash
agent47 help
```

### Check installation

```bash
agent47 doctor
```

This verifies:

* `agent47` availability
* installed helper commands
* current version

### Check for updates (manual, cached)

```bash
agent47 check-update          # uses cache (24h)
agent47 check-update --force  # bypass cache
```

`agent47 doctor` also runs the check once per invocation and reports the result without updating anything.

### Command Usage

All commands can be executed using either `agent47` or `a47`.
Examples:

```bash
agent47 doctor
a47 doctor

agent47 add-spec
a47 add-spec

agent47 add-agent-prompt
a47 add-agent-prompt
```

Note: `a47` is a real executable installed by the CLI.
It is not a shell alias and requires no shell configuration.

---

## Project Initialization

### Initialize agent environment in a project

From inside a project directory:

```bash
agent47 init-agent
```

This will copy:

* `AGENTS.md`
* `rules-*.yaml`
* create `README.md` (if missing)

No specs, skills, or prompts are created automatically.

---

## Optional Add-ons (Manual, Explicit)

You opt-in to each component.

### Add a base spec

```bash
agent47 add-spec
```

Creates:

```text
specs/spec.yml
```

---

### Add skills

```bash
agent47 add-skills
```

Creates:

```text
skills/
├── analyze.md
├── implement.md
├── review.md
├── refactor.md
└── optimize.md
```

---

### Add a general agent prompt

```bash
agent47 add-agent-prompt
```

Creates:

```text
prompts/agent-prompt.txt
```

---

### Add a spec & skill–driven prompt (SDD flow)

```bash
agent47 add-agent-prompt-ss
```

Creates:

```text
prompts/agent-prompt-ss.txt
```

---

## Upgrade / Repair

If you update the `agent47` repository or want to repair the installation:

```bash
agent47 install
```

This safely reinstalls helper commands and templates.
Run this after pulling repo changes to propagate updated templates (e.g., new `skills/*.md`) into `~/.agent47`.

---

## Uninstall

To remove the installed helper commands from your system:

```bash
agent47 uninstall
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
agent47 help
agent47 doctor
```

---

## Directory Structure (agent47)

```text
agent47/
├── bin/
│   └── agent47
├── scripts/
│   ├── add-agent
│   ├── add-spec
│   ├── add-skills
│   ├── add-agent-prompt
│   └── add-agent-prompt-ss
├── templates/
│   ├── AGENTS.md
│   ├── rules-*.yaml
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
agent47 init-agent
↓
agent47 add-spec (optional)
↓
agent47 add-skills (optional)
↓
agent47 add-agent-prompt / agent47 add-agent-prompt-ss
↓
Use your AI tool of choice
```

---

### Notes

#### Moving the agent47 repository

If you move the `agent47` repository after installation, the CLI command may stop working.
This is expected behavior when using symbolic links.
To fix it, recreate the symlink:

```bash
cd /new/path/to/agent47
ln -sf "$(pwd)/bin/agent47" ~/bin/agent47
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
