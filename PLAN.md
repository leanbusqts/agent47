# PLAN

> DEPRECATED
>
> This file is no longer the active plan or source of truth.
> It is kept only as a historical migration record for the Bash-to-Go transition completed in `agent47`.
> For current behavior and product contract, use `README.md`, `SNAPSHOT.md`, `SPEC.md`, and `docs/`.

## 1. Purpose

This file records what was delivered during the migration from the legacy Bash-first runtime to the current Go-first runtime.

It exists to answer:

- what changed
- what was preserved
- what compatibility choices were made
- what residual limitations remain after the migration

## 2. Migration Outcome

The migration was completed with these end-state properties:

- `afs` remains the public command surface
- the shipped product runtime is implemented in Go under `cmd/afs` and `internal/*`
- `install.sh` and `install.ps1` are thin install wrappers, not the product runtime
- `bin/afs` remains a repo-local development launcher
- the scaffold payload continues to live in `templates/`
- ownership remains manifest-driven through `templates/manifest.txt`
- helper commands remain published:
  - `add-agent`
  - `add-agent-prompt`
  - `add-ss-prompt`

## 3. What Was Preserved

The migration intentionally preserved:

- the `afs` command name
- the template-driven scaffold model
- managed vs preserved ownership semantics
- `--force` as the explicit destructive reconciliation switch
- prompt helpers
- local install flows through `install.sh` and `install.ps1`
- checkout-based development through `bin/afs`

Preserved target behavior:

- managed:
  - `AGENTS.md`
  - `rules/*.yaml`
  - `skills/*`
  - `skills/AVAILABLE_SKILLS.xml`
- preserved:
  - `README.md`
  - `specs/spec.yml`
  - `SNAPSHOT.md`
  - `SPEC.md`

## 4. What Changed

Major migration changes:

- the Bash-first runtime was replaced by a Go-first runtime
- core install/bootstrap/doctor/update/prompt logic moved into `internal/*`
- releases use Go-native runtime behavior
- repo-local development uses the checkout launcher rather than the old shell runtime
- the old shell library split under `scripts/lib/` is no longer the product boundary
- Windows-aware install and runtime paths became first-class concerns

Shell surface intentionally left in place:

- `bin/afs`
- `install.sh`
- `install.ps1`
- `scripts/lint-shell`
- shell and Bats-based test coverage

## 5. Delivered Work

The migration delivered:

- native CLI routing in Go
- runtime path detection in Go
- filesystem and template loading services
- manifest parsing and ownership enforcement
- bootstrap staging, commit, rollback, and `--force` reconciliation
- install and uninstall flows in Go
- update checks and doctor validation in Go
- prompt helper support in Go
- skill validation and `AVAILABLE_SKILLS.xml` generation in Go
- embedded/filesystem template source support
- Go unit/integration coverage plus installed-artifact verification

## 6. Post-Migration Hardening Completed

After the core migration, additional hardening was completed:

- strict launcher override validation in `bin/afs`
- strict launcher override validation in `install.ps1`
- improved symlink and path handling for portability
- stronger installed-artifact verification in `cmd/afsverify`
- coverage for:
  - `doctor --check-update`
  - `add-agent --force`
  - `add-agent --only-skills --force`
- better cross-platform path comparison in `doctor`
- safer Windows file replacement behavior with rollback in `fsx`
- context cancellation checks in install/bootstrap flows
- policy tests for command-surface and documentation consistency

## 7. Public Surface After Migration

Supported public commands:

- `afs help`
- `afs uninstall`
- `afs doctor [--check-update|--check-update-force|--fail-on-warn]`
- `afs add-agent [--force] [--only-skills]`
- `afs add-agent-prompt [--force]`
- `afs add-ss-prompt`
- `./install.sh [--force] [--non-interactive]`
- `.\install.ps1 [-Force] [-NonInteractive]`

Intentionally unsupported public commands:

- `afs install`
- `afs upgrade`
- `afs add-spec`
- `afs add-cli-prompt`
- `afs templates`
- `afs check-update`
- `afs add-default-skills`
- `afs init-agent`

## 8. Validation State

The migration and follow-up hardening are validated through:

- `go test ./...`
- checkout test runner via `cmd/afstest`
- installed-artifact verification via `cmd/afsverify`
- isolated smoke validation via `cmd/afssmoke`
- shell lint via `scripts/lint-shell`

Contributor entrypoints:

- `make test`
- `make go-test`
- `make go-build`
- `make lint-shell`
- `make smoke-install`

## 9. Residual Limitations

The main residual limitations after migration are:

- Windows confidence is improved but still less battle-tested than Unix-like environments
- Windows integration is not as deeply exercised end-to-end in day-to-day local validation
- some cross-platform filesystem behavior is safer than before but not as strong as ideal Unix rename semantics
- checkout-based development still depends on Go unless an explicit compiled CLI is supplied

## 10. Status

Migration status: completed

Plan status: deprecated historical archive

Last reviewed: March 26, 2026
