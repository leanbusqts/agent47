# Architecture

`agent47` is a Bash-first CLI built around a simple model:

1. install the managed launcher and templates with `./install.sh`
2. run project commands from `a47`
3. bootstrap or refresh managed scaffolding in the target repo

## Project structure

```text
agent47/
|
+-- bin/
|   `-- a47
|       Router and user-facing CLI entrypoint
|
+-- install.sh
|   Public installation entrypoint for the local machine
|
+-- scripts/
|   +-- add-agent
|   +-- add-agent-prompt
|   +-- add-snapshot-prompt
|   +-- test
|   `-- lib/
|       +-- common.sh
|       +-- constants.sh
|       +-- doctor.sh
|       +-- install.sh
|       +-- templates.sh
|       `-- update.sh
|           Shared implementation modules
|
+-- templates/
|   +-- AGENTS.md
|   +-- prompts/
|   |   +-- agent-prompt.txt
|   |   `-- snapshot-prompt.txt
|   +-- rules/
|   |   +-- rules-backend.yaml
|   |   +-- rules-frontend.yaml
|   |   +-- rules-mobile.yaml
|   |   `-- security-*.yaml
|   `-- skills/
|       `-- <skill>/SKILL.md
|
+-- tests/
|   `-- unit/
|       Bats coverage for CLI behavior and template integrity
|
+-- AGENTS.md
+-- SNAPSHOT.md
`-- README.md
```

## Execution flow

```text
user
  |
  +--> ./install.sh
  |      |
  |      v
  |   scripts/lib/install.sh
  |      |
  |      v
  |   ~/.agent47 + ~/bin/a47
  |
  `--> a47
         |
         +--> doctor [--check-update]
         |      Health check plus optional update check
         |
         +--> add-agent
         |      Bootstrap completo:
         |      AGENTS.md + rules + skills + README if missing
         |
         +--> add-agent --force
         |      Refresh completo:
         |      managed files only, preserving project-owned files
         |
         +--> add-agent --only-skills [--force]
         |      Refresh parcial:
         |      skills/* + AVAILABLE_SKILLS.xml only
         |
         `--> add-*-prompt
                Helper prompt generation
```

## Responsibility split

- `bin/a47`
  - command routing
  - high-level help
  - delegates to shared libs and concrete scripts

- `install.sh`
  - public local installation entrypoint
  - installs the managed launcher and managed templates under `~/.agent47`

- `scripts/lib/`
  - reusable CLI internals
  - constants, installation, doctor, lightweight template checks, update checks

- `scripts/add-*`
  - user-invoked write operations
  - bootstrap or refresh project scaffolding
  - `add-agent` is the main project command and also handles the `--only-skills` path
  - curated skills are discovered from `templates/skills/*/SKILL.md` instead of a hardcoded list

- `templates/`
  - canonical source of scaffolded files
  - what gets copied into user projects

- `tests/unit`
  - behavior checks for install, doctor, bootstrap, refresh, prompts, skills, and policy integrity

## Design choices

- Bash-first, no extra runtime dependency
- conservative by default, explicit `--force` for refresh
- policy lives in `AGENTS.md`, not duplicated in prompts
- security guidance is layered: global, language, stack
- template-source repositories such as `agent47` can map policy reads from `rules/` to `templates/rules/`
- project-specific files such as `README.md`, `specs/spec.yml`, and `SNAPSHOT.md` stay preserved during forced refresh
