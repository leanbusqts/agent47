# SNAPSHOT

## 1. Project Overview

- **Name:** `agent47`
- **Purpose:** Go-first CLI plus templates for setting up agent-driven development workflows with explicit policy, rules, skills, prompts, and an optional `specs/spec.yml` planning artifact

## 2. Current Status

- **Runtime:** `afs` is implemented by the Go CLI under `cmd/afs` and `internal/*`
- **Repo launcher:** `bin/afs` is a checkout launcher for development; it prefers `AGENT47_GO_CLI`, then `AGENT47_REPO_CLI`, and otherwise uses `go run`
- **Installers:** `install.sh` and `install.ps1` are thin wrappers around the native install service
- **Bootstrap:** `afs add-agent` scaffolds `AGENTS.md`, template rules, curated skills, and `skills/AVAILABLE_SKILLS.xml`
- **Forced refresh:** `afs add-agent --force` performs a fresh install of the managed scaffold and removes stale managed rules and skills
- **Preserved targets:** `README.md`, `specs/spec.yml`, `SNAPSHOT.md`, and root `SPEC.md` stay untouched during forced refresh
- **Skills-only mode:** `afs add-agent --only-skills` manages only the skills tree; without `--force`, invalid existing skill files are preserved but omitted from `AVAILABLE_SKILLS.xml`
- **Update checks:** `doctor --check-update` uses a remote `VERSION` when configured or the local git tracking ref when running from a checkout with an upstream branch; `doctor --check-update-force` performs `git fetch --quiet` first
- **Doctor flags:** update flags and `--fail-on-warn` can be combined in a single invocation
- **Prompt helper:** `add-ss-prompt` copies to a supported clipboard tool when available and otherwise prints to stdout
- **Testing:** `make test`, `make go-test`, `make go-build`, `make lint-shell`, and `make smoke-install` are the current maintainer entrypoints

## 3. Current Commands

- `./install.sh [--force] [--non-interactive]`
- `.\install.ps1 [-Force] [-NonInteractive]`
- `afs help`
- `afs uninstall`
- `afs doctor [--check-update|--check-update-force|--fail-on-warn]`
- `afs add-agent [--force] [--only-skills]`
- `afs add-agent-prompt [--force]`
- `afs add-ss-prompt`

## 4. Key Repository Structure

- `bin/afs` - repo-local launcher
- `cmd/afs` - native CLI entrypoint
- `internal/` - runtime packages for bootstrap, install, doctor, prompts, templates, manifest parsing, update checks, and platform handling
- `templates/manifest.txt` - canonical managed/preserved target contract
- `templates/` - scaffold payload copied into target repositories
- `docs/usage.md` - operational guide
- `docs/architecture.md` - runtime and repo structure summary
- `docs/ownership.md` - ownership model for repo source vs scaffold payload
- `scripts/lint-shell` - remaining shell maintainer script

## 5. Constraints And Risks

- `--force` is intentionally destructive inside managed paths such as `rules/` and `skills/`
- git-based update checks use local tracking refs and can be stale until the user fetches or runs `--check-update-force`
- checkout-based execution still depends on Go unless a compiled CLI is supplied explicitly
- Windows-aware code paths exist, but the strongest day-to-day repo validation remains on Unix-like systems

## 6. Last Updated

- March 26, 2026
