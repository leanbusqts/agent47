# agent64

`agent64` is a lightweight CLI to bootstrap and manage **AI Agent–driven development workflows**, with support for:

- AGENTS.md–based agent guidance
- Spec Driven Development (SDD)
- Skills-based agent execution
- Reusable prompts
- Consistent project initialization

It is designed to be **simple, explicit, and composable**, without hidden automation.

---

## Philosophy

- **agent64 does not manage your project**
- **agent64 provides templates and conventions**
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

## Installation

### 1. Clone or download `agent64`

Place the repository anywhere you want, for example:

```bash
~/Documents/agent64
````

### 2. Make the CLI executable

```bash
chmod +x bin/agent64
```

### 3. Install agent64 tools

From inside the repository:

```bash
./bin/agent64 install
```

This installs the following commands into `~/bin`:

* `add-agent`
* `add-spec`
* `add-skills`
* `add-agent-prompt`
* `add-agent-prompt-ss`

### 4. Make agent64 available globally (recommended)

Create a symlink so agent64 can be executed from anywhere:

```bash
ln -s $(pwd)/bin/agent64 ~/bin/agent64
```

Ensure ~/bin is in your $PATH.

You can verify with:
```bash
agent64 doctor
```

---

## Usage

### Show help

```bash
agent64 help
```

### Check installation

```bash
agent64 doctor
```

This verifies:

* `agent64` availability
* installed helper commands
* current version

---

## Project Initialization

### Initialize agent environment in a project

From inside a project directory:

```bash
agent64 init-agent
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
add-spec
```

Creates:

```text
specs/spec.yml
```

---

### Add skills

```bash
add-skills
```

Creates:

```text
skills/
├── analyze.yml
├── implement.yml
├── review.yml
├── refactor.yml
└── optimize.yml
```

---

### Add a general agent prompt

```bash
add-agent-prompt
```

Creates:

```text
prompts/agent-prompt.txt
```

---

### Add a spec & skill–driven prompt (SDD flow)

```bash
add-agent-prompt-ss
```

Creates:

```text
prompts/agent-prompt-ss.txt
```

---

## Upgrade

If you update the `agent64` repository and want to reinstall the scripts:

```bash
agent64 upgrade
```

This safely reinstalls all helper commands.

---

## Uninstall

To remove the installed helper commands from your system:

```bash
agent64 uninstall
```

This does **not** delete:

* projects
* specs
* prompts
* templates
* the `agent64` repository itself

---

## Versioning

`agent64` follows **Semantic Versioning (SemVer)**.

The current version is stored in:

```text
VERSION
```

You can check it with:

```bash
agent64 help
agent64 doctor
```

---

## Directory Structure (agent64)

```text
agent64/
├── bin/
│   └── agent64
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

## What agent64 Does NOT Do

* It does not run agents
* It does not modify code automatically
* It does not hide decisions
* It does not enforce tools or vendors

`agent64` is intentionally minimal.

---

## Intended Workflow (High Level)

```text
agent64 install
↓
cd project
↓
agent64 init-agent
↓
add-spec (optional)
↓
add-skills (optional)
↓
add-agent-prompt / add-agent-prompt-ss
↓
Use your AI tool of choice
```

---

## License / Usage

Internal tooling / personal workflow.
Adapt freely to your needs.

```

---
