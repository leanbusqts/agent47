# Architecture

`agent47` is a Go-first CLI with a small shell layer around checkout launching and shell linting.

The shipped model is:

1. install the managed runtime with `./install.sh` or `.\install.ps1`
2. run project commands through `afs`
3. bootstrap or refresh scaffolded repo files from `templates/`

## Project structure

```text
agent47/
|
+-- bin/
|   `-- afs
|       Repo launcher for checkout-based development
|
+-- cmd/
|   `-- afs/
|       `-- main.go
|           Native CLI entrypoint
|
+-- internal/
|   +-- app/
|   +-- bootstrap/
|   +-- doctor/
|   +-- fsx/
|   +-- install/
|   +-- manifest/
|   +-- platform/
|   +-- prompts/
|   +-- runtime/
|   +-- skills/
|   +-- templates/
|   +-- update/
|   `-- version/
|       Native runtime packages
|
+-- install.sh
+-- install.ps1
|   Public install wrappers
|
+-- scripts/
|   `-- lint-shell
|       Maintainer shell lint entrypoint
|
+-- templates/
|   +-- AGENTS.md
|   +-- manifest.txt
|   +-- prompts/
|   +-- rules/
|   +-- skills/
|   `-- specs/
|       Canonical scaffold payload
|
+-- tests/
|   +-- unit/
|   `-- vendor/
|       Bats-based repo verification
|
+-- docs/
+-- README.md
+-- SNAPSHOT.md
+-- SPEC.md
`-- PLAN.md
```

## Execution flow

```text
user
  |
  +--> ./install.sh or .\install.ps1
  |      |
  |      v
  |   bin/afs
  |      |
  |      v
  |   cmd/afs + internal/install
  |      |
  |      v
  |   managed runtime + templates
  |
  `--> afs
         |
         +--> doctor [--check-update]
         |      install and template health checks
         |
         +--> add-agent
         |      bootstrap managed scaffold
         |
         +--> add-agent --force
         |      fresh install of managed scaffold
         |
         +--> add-agent --only-skills [--force]
         |      manage only the skills tree
         |
         `--> add-agent-prompt / add-ss-prompt
                prompt helpers
```

## Responsibility split

- `bin/afs`
  - repo-local launcher
  - prefers `AGENT47_GO_CLI`
  - otherwise prefers `AGENT47_REPO_CLI`
  - otherwise runs `go run ./cmd/afs`
  - avoids self-recursion when `AGENT47_REPO_CLI` points back to itself

- `cmd/afs` and `internal/*`
  - command routing
  - runtime path detection
  - template loading
  - manifest parsing
  - bootstrap staging, commit, and rollback
  - install/uninstall
  - doctor and update checks
  - prompt helpers
  - skill validation and `AVAILABLE_SKILLS.xml` generation

- `install.sh` and `install.ps1`
  - public local install entrypoints
  - forward to the native install service
  - handle wrapper-specific PATH setup behavior

- `templates/manifest.txt`
  - declarative ownership contract
  - lists rule templates
  - defines managed targets
  - defines preserved targets
  - defines required install assets

- `templates/`
  - canonical source for scaffolded files
  - copied into target repos or installed into the managed template directory

- `scripts/lint-shell`
  - maintainer shell lint entrypoint
  - validates Bash sources and Bats tests when `shellcheck` is available

## Ownership model

Managed project targets:

- `AGENTS.md`
- `rules/*.yaml`
- `skills/*`
- `skills/AVAILABLE_SKILLS.xml`

Preserved project targets:

- `README.md`
- `specs/spec.yml`
- `SNAPSHOT.md`
- `SPEC.md`

`afs add-agent --force` is intentionally destructive within managed paths. It reconciles the managed scaffold to the current template set, so local custom files under `rules/` or `skills/` can be removed.

## Design choices

- Go-first runtime with templates remaining as plain files
- checkout-friendly launcher for local development
- explicit `--force` for destructive refresh
- manifest-driven ownership and install validation
- preserved project docs stay outside the default managed scaffold
- shell is limited to thin wrappers and maintainer tooling rather than product runtime logic
