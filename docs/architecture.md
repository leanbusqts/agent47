# Architecture

`agent47` is a Bash-first CLI built around a simple model:

1. install the managed launcher and templates with `./install.sh`
2. run `agent47` project commands via `afs`
3. bootstrap or refresh managed scaffolding in the target repo

## Project structure

```text
agent47/
|
+-- bin/
|   `-- afs
|       Router and user-facing CLI entrypoint
|
+-- install.sh
|   Public installation entrypoint for the local machine
|
+-- scripts/
|   +-- add-agent
|   +-- add-agent-prompt
|   +-- add-ss-prompt
|   +-- lint-shell
|   +-- smoke-install
|   +-- test
|   `-- lib/
|       +-- bootstrap.sh
|       +-- common.sh
|       +-- constants.sh
|       +-- doctor.sh
|       +-- install-assets.sh
|       +-- install-runtime.sh
|       +-- install.sh
|       +-- managed-files.sh
|       +-- runtime-env.sh
|       +-- skill-utils.sh
|       +-- test-runtime.sh
|       +-- templates.sh
|       `-- update.sh
|           Shared implementation modules by domain
|
+-- templates/
|   +-- AGENTS.md
|   +-- manifest.txt
|   +-- prompts/
|   |   +-- agent-prompt.txt
|   |   `-- ss-prompt.txt
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
+-- SPEC.md
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
  |   ~/.agent47 + ~/bin/afs
  |
  `--> afs
         |
         +--> doctor [--check-update]
         |      Health check plus optional update check
         |
         +--> add-agent
         |      Bootstrap project scaffolding:
         |      AGENTS.md + rules + skills + empty README if missing
         |
         +--> add-agent --force
         |      Refresh managed scaffolding:
         |      managed files only, preserving project-owned files
         |
         +--> add-agent --only-skills [--force]
         |      Refresh only skills:
         |      skills/* + AVAILABLE_SKILLS.xml only
         |
         `--> add-*-prompt
                Helper prompt generation
```

## Responsibility split

- `bin/afs`
  - command routing
  - high-level help
  - runtime bootstrap is delegated to `scripts/lib/runtime-env.sh`
  - delegates to shared libs and concrete scripts

- `install.sh`
  - public local installation entrypoint
  - installs the managed launcher and managed templates under `~/.agent47`
  - interactive PATH persistence targets the active shell's preferred rc file

- `scripts/lib/`
  - reusable CLI internals
  - `managed-files.sh` defines project-owned vs agent47-managed file boundaries
  - `runtime-env.sh` centralizes path resolution, version loading, and shared environment bootstrap
  - `bootstrap.sh` owns project scaffolding transactions, staging, and rollback
  - `install-assets.sh` owns atomic file, directory, and symlink publication helpers
  - `install-runtime.sh` owns installer preflight, managed runtime publication, and uninstall flows
  - `test-runtime.sh` owns temporary test environment setup and Bats resolution
  - `install.sh`, `doctor.sh`, `templates.sh`, and `update.sh` stay focused on orchestration and diagnostics

- `scripts/add-*`
  - user-invoked write operations
  - bootstrap or refresh project scaffolding
  - `add-agent` is now a thin entrypoint that delegates bootstrap behavior to shared modules
  - curated skills are discovered from `templates/skills/*/SKILL.md` instead of a hardcoded list

- `scripts/test`
  - repo-level executable test runner
  - stays outside `lib/` because it is a user-invoked command, not a reusable shell module

- `scripts/smoke-install`
  - isolated install + `doctor` smoke check for releases and maintenance validation

- `templates/manifest.txt`
  - declarative scaffold manifest for managed targets, preserved targets, and rule template membership

- `templates/`
  - canonical source of scaffolded files
  - what gets copied into user projects

- `tests/unit`
  - behavior checks for install, doctor, bootstrap, refresh, prompts, skills, and policy integrity

## Design choices

- Bash-first, no extra runtime dependency
- conservative by default, explicit `--force` for refresh
- forced refresh reconciles manifest-managed targets instead of only overwriting matching filenames
- forced refresh reconciles managed paths such as `rules/` and `skills/` against the current template payload, which can remove local custom files there
- policy lives in `AGENTS.md`, not duplicated in prompts
- security guidance is layered: global, language, stack, including shell-specific rules for Bash-first repos
- template-source repositories such as `agent47` can map policy reads from `rules/` to `templates/rules/`
- project-specific files such as `README.md`, `specs/spec.yml`, and `SNAPSHOT.md` stay preserved during forced refresh

## Ownership model

- repo-owned source files
  - CLI code under `bin/`, `scripts/`, `docs/`, and `tests/`
  - these exist to build, install, and verify `agent47` itself

- template payload
  - canonical scaffold content under `templates/`
  - this is what gets copied into user repositories

- project-managed targets
  - `AGENTS.md`, `rules/*.yaml`, `skills/*`, and `skills/AVAILABLE_SKILLS.xml`
  - ownership is defined in `scripts/lib/managed-files.sh`
  - local custom files under these paths can be replaced or removed during `--force`

- project-owned preserved targets
  - `README.md`, `specs/spec.yml`, and snapshot-style files such as `SNAPSHOT.md`
  - refresh flows keep these intact unless a user changes them manually
