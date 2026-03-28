# SPEC

This root `SPEC.md` describes the current product contract and technical behavior of `agent47` itself. It is not the default place to draft a new feature spec or implementation plan; use `specs/spec.yml` for task-scoped planning work.

## 1. Project Identity

- **Name:** `agent47`
- **Type:** Go-first CLI plus template payload for bootstrapping agent-oriented repository scaffolding
- **Public command:** `afs`
- **Primary purpose:** install a local CLI runtime and scaffold a portable repository contract for agent workflows without coupling target repos to vendor-specific agent config

## 2. Product Summary

`agent47` has two layers:

1. A local CLI runtime installed on the machine and exposed through `afs`.
2. A template payload copied into target repositories to establish:
   - `AGENTS.md`
   - `rules/*.yaml`
   - `skills/*`
   - `skills/AVAILABLE_SKILLS.xml`
   - prompt helpers
   - `specs/spec.yml` as an available template artifact

The product is intentionally conservative:

- bootstrap is explicit
- refresh is opt-in
- `--force` is required for destructive reconciliation
- preserved project docs remain untouched during refresh
- policy is centralized in `AGENTS.md`

## 3. Goals

- Keep policy, stack guidance, security rules, skills, and prompts versioned as normal files inside the target repo.
- Make managed vs preserved ownership explicit and machine-checkable through `templates/manifest.txt`.
- Support agent CLIs and IDE agents through repo conventions instead of vendor-specific config.
- Keep install, refresh, uninstall, and doctor flows safe and testable.
- Support both installed use and checkout-based development without maintaining separate products.

## 4. Non-Goals

- Full project generation beyond the agent-workflow scaffold.
- Hidden remote orchestration, cloud sync, or network dependency for normal operation.
- Automatic creation of project-specific business specs beyond the optional `specs/spec.yml` template.
- Enforcement of package-manager dependency manifests in target repos.
- Vendor-specific repo config by default.

## 5. Core Workflows

### 5.1 Local install

Public entrypoints:

- `./install.sh [--force] [--non-interactive]`
- `.\install.ps1 [-Force] [-NonInteractive]`

Expected outcome:

- install managed templates under the agent47 home directory
- install a managed `afs` launcher
- publish helper entrypoints (`add-agent`, `add-agent-prompt`, `add-ss-prompt`)
- expose `afs` on the user PATH

Default install locations:

- Unix-like systems:
  - home: `~/.agent47`
  - managed launcher: `~/.agent47/bin/afs`
  - user entry: `~/bin/afs`
- Windows:
  - home: `%LOCALAPPDATA%\agent47` by default
  - managed launcher: `%LOCALAPPDATA%\agent47\bin\afs.exe`
  - user PATH target: `%LOCALAPPDATA%\agent47\bin`

### 5.2 Verify local installation

Public command:

- `afs doctor [--check-update|--check-update-force|--fail-on-warn]`

Expected outcome:

- confirm managed launcher and helper commands are available
- confirm templates and prompt assets exist
- validate template integrity and required `AGENTS.md` sections
- optionally perform an update check
- allow combining `--fail-on-warn` with update-check flags in one invocation

Update-check behavior:

- if `AGENT47_VERSION_URL` is configured, compare local version with remote `VERSION`
- otherwise, when running from a git checkout with an upstream branch, compare `HEAD` against the upstream branch
- regular git checks compare against the local tracking ref
- `--check-update-force` performs `git fetch --quiet` before comparing
- remote checks may be cached; git tracking checks are not cached
- cached remote checks are scoped to the configured update source

### 5.3 Bootstrap a target repository

Public command:

- `afs add-agent`

Expected outcome:

- write `AGENTS.md` if absent
- write `rules/*.yaml` if absent
- create `skills/*` and `skills/AVAILABLE_SKILLS.xml`
- create an empty `README.md` if absent
- create `specs/spec.yml` if absent
- preserve existing managed files unless `--force` is used

### 5.4 Fresh reinstall of the managed scaffold

Public command:

- `afs add-agent --force`

Expected outcome:

- replace `AGENTS.md`
- reconcile `rules/*.yaml` against current templates
- replace `skills/*`
- regenerate `skills/AVAILABLE_SKILLS.xml`
- remove stale managed rules and skills no longer shipped by current templates
- preserve:
  - `README.md`
  - `specs/spec.yml`
  - `SNAPSHOT.md`
  - `SPEC.md`

Implication:

- local custom files placed under managed paths such as `rules/` or `skills/` can be removed during `--force`

### 5.5 Skills-only refresh

Public commands:

- `afs add-agent --only-skills`
- `afs add-agent --only-skills --force`

Expected outcome:

- manage only `skills/*` and `skills/AVAILABLE_SKILLS.xml`
- do not touch `AGENTS.md` or `rules/*.yaml`
- without `--force`, preserve existing invalid skill files and omit them from `AVAILABLE_SKILLS.xml`
- with `--force`, replace the managed skills directory with the template set

### 5.6 Prompt helpers

Public commands:

- `afs add-agent-prompt [--force]`
- `afs add-ss-prompt`

Expected outcome:

- create or refresh `prompts/agent-prompt.txt`
- copy the snapshot/spec helper prompt to a supported clipboard tool when available
- otherwise print the snapshot/spec helper prompt from installed templates

### 5.7 Uninstall

Public command:

- `afs uninstall`

Expected outcome:

- remove published commands
- remove managed runtime assets
- remove managed template backups

## 6. Public Command Surface

Supported user-facing commands:

- `afs help`
- `afs uninstall`
- `afs doctor [--check-update|--check-update-force|--fail-on-warn]`
- `afs add-agent [--force] [--only-skills]`
- `afs add-agent-prompt [--force]`
- `afs add-ss-prompt`
- `./install.sh [--force] [--non-interactive]`
- `.\install.ps1 [-Force] [-NonInteractive]`

Contributor validation commands:

- `make test`
- `make go-test`
- `make go-build`
- `make lint-shell`
- `make smoke-install`

Commands intentionally outside the supported public surface:

- `afs install`
- `afs upgrade`
- `afs add-spec`
- `afs add-cli-prompt`
- `afs templates`
- `afs check-update`

## 7. Repository Structure Contract

### 7.1 Repo-owned source

- `bin/`
- `cmd/`
- `internal/`
- `scripts/`
- `docs/`
- `tests/`
- root metadata such as `README.md`, `VERSION`, `SNAPSHOT.md`, `SPEC.md`, `PLAN.md`

### 7.2 Template payload

- `templates/AGENTS.md`
- `templates/manifest.txt`
- `templates/rules/*.yaml`
- `templates/skills/*/SKILL.md`
- `templates/prompts/*.txt`
- `templates/specs/spec.yml`

### 7.3 Managed target contract

Defined by `templates/manifest.txt`:

- managed targets:
  - `AGENTS.md`
  - `rules/*.yaml`
  - `skills/*`
  - `skills/AVAILABLE_SKILLS.xml`
- preserved targets:
  - `README.md`
  - `specs/spec.yml`
  - `SNAPSHOT.md`
  - `SPEC.md`

## 8. Architecture

### 8.1 Repo launcher

`bin/afs` is the checkout launcher for repository-local development. It:

- prefers an explicit compiled Go CLI via `AGENT47_GO_CLI`
- otherwise prefers an explicit repo CLI via `AGENT47_REPO_CLI`
- otherwise runs `go run ./cmd/afs` when Go is available
- does not execute a legacy shell runtime

### 8.2 Native runtime

`cmd/afs` plus `internal/*` implement the product runtime in Go. Major responsibilities include:

- command routing
- runtime path detection
- template loading from filesystem or embedded sources
- manifest parsing
- skill validation and XML generation
- install/uninstall
- bootstrap staging, commit, and rollback
- doctor and update checks
- prompt helpers

### 8.3 Install wrappers

`install.sh` and `install.ps1` are thin wrappers that invoke the native install service through the repo launcher. They are part of the public install surface, but they do not contain the core product logic.

### 8.4 Shell surface

The repo still contains a small shell layer for:

- `bin/afs` checkout launching
- `scripts/lint-shell`
- contributor-oriented shell verification around tests and install flows

The product runtime itself is no longer Bash-first.

## 9. Testing And Validation

Current repo validation is centered on:

- Go tests: `make go-test`
- full checkout + installed-artifact validation: `make test`
- shell lint: `make lint-shell`
- isolated install smoke check: `make smoke-install`

The test suite covers:

- install and uninstall flows
- bootstrap and rollback behavior
- prompt helpers
- template integrity
- skills validation and XML generation
- doctor and update-check behavior
- helper publication and launcher behavior

## 10. Constraints And Risks

- `afs add-agent --force` is intentionally destructive within managed paths.
- Regular git-based update checks reflect the local tracking ref; `--check-update-force` refreshes that state with `git fetch`.
- Checkout-based development still depends on Go unless an explicit compiled CLI is provided.
- The runtime contains Windows-aware paths and a PowerShell installer, but day-to-day repo validation remains strongest on Unix-like systems.
