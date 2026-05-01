# SNAPSHOT

## 1. Project Overview

- **Name:** `agent47`
- **Purpose:** Go-first CLI plus templates for setting up agent-driven development workflows with explicit policy, rules, skills, prompts, and an optional `specs/spec.yml` planning artifact

## 2. Current Status

- **Runtime:** `afs` is implemented by the Go CLI under `cmd/afs` and `internal/*`
- **Repo launcher:** `bin/afs` is a checkout launcher for development; it prefers `AGENT47_GO_CLI`, then `AGENT47_REPO_CLI`, and otherwise uses `go run`
- **Repo launcher caches:** when `bin/afs` falls back to `go run`, it now seeds both repo-safe `GOCACHE` and `GOMODCACHE` defaults
- **Installers:** `install.sh` and `install.ps1` are thin wrappers around the native install service
- **Analyze:** `afs analyze` is read-only and reports detected project types, testing stacks, evidence, and the resolved install plan; `--evidence` now includes classification evidence as well as raw scan hits
- **Bootstrap:** `afs add-agent` analyzes first, previews the resolved bundle set, and then scaffolds `AGENTS.md`, template rules, curated skills, `skills/AVAILABLE_SKILLS.xml`, `skills/AVAILABLE_SKILLS.json`, `skills/SUMMARY.md`, and `specs/spec.yml`; prompt helpers are now opt-in via their own commands
- **Skills indexes:** the skills contract also supports JSON and Markdown summary indexes alongside `skills/AVAILABLE_SKILLS.xml`
- **Forced refresh:** `afs add-agent --force` performs a fresh install of the managed scaffold and removes stale managed rules and skills not present in the resolved assembled contract
- **Preserved targets:** `README.md`, `specs/spec.yml`, `SNAPSHOT.md`, and root `SPEC.md` stay untouched during forced refresh
- **Skills-only mode:** `afs add-agent --only-skills` manages only the skills tree; without `--force`, invalid existing skill files are preserved but omitted from the generated skills indexes; in interactive terminals it follows the same confirmation rules as the main scaffold flow
- **Skills-only preview:** `afs add-agent --only-skills --force --preview` now reflects the actual managed `skills/` replacement and pending removals under `skills/`
- **Update checks:** `doctor --check-update` uses a remote `VERSION` when configured or the local git tracking ref when running from a checkout with an upstream branch; `doctor --check-update-force` performs `git fetch --quiet` first
- **Doctor flags:** update flags and `--fail-on-warn` can be combined in a single invocation
- **Doctor validation:** `afs doctor` now validates the installed manifest contract, required template files and directories, stack rule templates, security templates, and required `AGENTS.md` sections
- **Prompt helpers:** `add-agent-prompt` and `add-ss-prompt` remain available as explicit helper commands, with `add-ss-prompt` copying to a supported clipboard tool when available and otherwise printing to stdout
- **Testing:** `make test`, `make go-test`, `make go-build`, `make lint-shell`, and `make smoke-install` are the current maintainer entrypoints

## 3. Current Commands

- `./install.sh [--force] [--non-interactive]`
- `.\install.ps1 [-Force] [-NonInteractive]`
- `afs help`
- `afs version`
- `afs uninstall`
- `afs doctor [--check-update|--check-update-force|--fail-on-warn]`
- `afs analyze [--json|--verbose|--evidence]`
- `afs add-agent [--force] [--only-skills] [--preview|--dry-run] [--yes] [--bundle <name>] [--exclude-bundle <name>]`
- `afs add-agent-prompt [--force]`
- `afs add-ss-prompt`

## 4. Key Repository Structure

- `bin/afs` - repo-local launcher
- `cmd/afs` - native CLI entrypoint
- `internal/` - runtime packages for bootstrap, install, doctor, prompts, templates, manifest parsing, update checks, and platform handling
- `templates/manifest.txt` - managed/preserved target contract for scaffold ownership
- `templates/base/` - shared scaffold payload
- `templates/bundles/` - project-specific bundle payload
- `README.md` - entrypoint, command surface, and high-level architecture
- `RUNBOOK.md` - operational guide for using the CLI in depth
- `SPEC.md` - current-state product contract
- `scripts/lint-shell` - remaining shell maintainer script

## 5. Constraints And Risks

- `--force` is intentionally destructive inside managed paths such as `rules/` and `skills/`
- git-based update checks use local tracking refs and can be stale until the user fetches or runs `--check-update-force`
- checkout-based execution still depends on Go unless a compiled CLI is supplied explicitly
- Windows-aware code paths exist, but the strongest day-to-day repo validation remains on Unix-like systems

## 6. Last Updated

- April 30, 2026

## 7. Verification Notes

- Manual empty-repo verification on April 29, 2026: `afs analyze` reported `type: unknown` and `bundles: base`, then `afs add-agent` installed the expected base scaffold.
- Manual legacy-scaffold verification on April 29, 2026: `afs add-agent --force --yes` removed stale managed rules and stale managed skills, preserved `README.md` and `specs/spec.yml`, and kept prompt helpers as separate opt-in commands
- Manual preview verification on April 30, 2026: `afs add-agent --only-skills --force --preview` reported the managed `skills/` replacement plus the concrete entries that would be removed under `skills/`.
