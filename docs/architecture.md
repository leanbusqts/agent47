# Architecture

`agent47` is a Bash-first CLI built around a simple model:

1. install CLI helpers and templates into `~/.agent47`
2. run project commands from `a47`
3. copy managed scaffolding into the target repo

## Project structure

```text
agent47/
|
+-- bin/
|   `-- a47
|       Router and user-facing CLI entrypoint
|
+-- scripts/
|   +-- add-agent
|   +-- add-cli-prompt
|   +-- add-agent-prompt
|   +-- add-skills
|   +-- add-spec
|   +-- add-snapshot-prompt
|   +-- reload-skills
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
|   |   +-- cli-prompt.txt
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
  v
a47
  |
  +--> scripts/lib/*.sh
  |     Shared routing, install, doctor, template logic
  |
  `--> scripts/add-*
        Concrete project actions
              |
              v
         templates/*
              |
              v
      target project files
```

## Responsibility split

- `bin/a47`
  - command routing
  - high-level help
  - delegates to shared libs and concrete scripts

- `scripts/lib/`
  - reusable CLI internals
  - constants, installation, doctor, templates, update checks

- `scripts/add-*`
  - user-invoked write operations
  - bootstrap or refresh project scaffolding

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
- project-specific files such as `README.md`, `specs/spec.yml`, and `SNAPSHOT.md` stay preserved during forced refresh
