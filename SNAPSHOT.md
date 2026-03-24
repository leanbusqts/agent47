# SNAPSHOT

## 1. Project Overview
- **Name:** agent47
- **Purpose:** Bash CLI plus templates for setting up agent-driven development workflows with explicit policy, rules, skills, prompts, and optional specs.

## 2. Current Status
- **Implemented:** the `agent47` CLI, invoked via `afs`, supports uninstall, doctor, opt-in update checks, and project bootstrap commands, plus `./install.sh` as the public installation entrypoint; curated templates for `AGENTS.md`, stack rules, security rules, skills, prompts, and `specs/spec.yml`; Bats-based unit test suite and Make targets.
- **Architecture:** the `agent47` CLI entrypoint at `bin/afs` is now mostly a router and sources shared logic from `scripts/lib/`, including dedicated runtime bootstrap, install, bootstrap, test-runtime, and skill helper modules plus a declarative `templates/manifest.txt`.
- **Documentation:** `README.md` is now intentionally short; usage and structure details live in `docs/usage.md` and `docs/architecture.md`.
- **Stable workflow:** `afs add-agent` bootstraps `AGENTS.md` and all `rules/*.yaml` plus skills; `afs add-agent-prompt` and `afs add-ss-prompt` are available as focused helpers.
- **Managed refresh behavior:** during `afs add-agent --force`, `rules/*.yaml`, `skills/*`, and `skills/AVAILABLE_SKILLS.xml` are reconciled against the current template set, so local custom files in those paths may be replaced or removed.
- **Preserved project docs:** during `afs add-agent --force`, `README.md`, `specs/spec.yml`, `SNAPSHOT.md`, and root `SPEC.md` stay untouched.
- **Dynamic skills:** the curated skill set is discovered from installed `templates/skills/*/SKILL.md` entries instead of a hardcoded list, so template additions flow into bootstrap automatically.
- **Operational hardening:** core helper scripts now run with strict shell mode, install preflights core assets, and bootstrap commands reject unexpected arguments to reduce silent partial failures.
- **Install hardening:** `./install.sh` now installs the `afs` launcher under `~/.agent47/bin/` and links `~/bin/afs` to that managed copy, avoiding dependence on the original repo checkout path.
- **Not automated by CLI:** `SNAPSHOT.md` and `SPEC.md` creation/update remain manual-agent workflows, alongside vendor-specific configs, Windows/PowerShell support, and dependency enforcement against concrete package manifests.

## 3. Current Commands
- `./install.sh [--force] [--non-interactive]`
- `afs uninstall`
- `afs doctor [--check-update|--check-update-force]`
- `afs add-agent [--force]`
- `afs add-agent --only-skills [--force]`
- `afs add-agent-prompt`
- `afs add-ss-prompt`

## 4. Key Repository Structure
- `AGENTS.md` – live root policy file for this repo.
- `docs/usage.md` – operational guide for install, bootstrap, and refresh.
- `docs/architecture.md` – architecture notes and ASCII project map.
- `bin/afs` – public `afs` command entrypoint for the `agent47` CLI.
- `install.sh` – installer that writes the managed launcher to `~/.agent47/bin/afs` and links `~/bin/afs`.
- `scripts/`
  - `add-agent`
  - `add-agent-prompt`
  - `add-ss-prompt`
  - `lint-shell`
  - `smoke-install`
  - `test`
  - `lib/*.sh`
- `templates/manifest.txt` – declarative scaffold manifest for managed and preserved targets.
- `templates/`
  - `AGENTS.md`
  - `prompts/agent-prompt.txt`
  - `prompts/ss-prompt.txt`
  - `rules/rules-frontend.yaml`
  - `rules/rules-backend.yaml`
  - `rules/rules-mobile.yaml`
  - `rules/security-*.yaml`
  - `specs/spec.yml`
  - `skills/<name>/SKILL.md`
- `tests/unit/` – Bats unit coverage for CLI, prompts, policy, skills, backups, and snapshot/spec helper flows.

## 5. Policy And Rules Model
- Authority order: user > nearest `AGENTS.md` > security rules > stack rules > spec > code/tests > memories.
- `AGENTS.md` is the single source of policy; prompts and README should reference it rather than duplicating policy.
- In template-source repositories such as `agent47` itself, policy reads that normally point to `rules/` should use `templates/rules/`.
- `AGENTS.md` now encourages an implement-then-review flow for non-trivial work and recommends multi-agent or sub-agent execution for complex or multi-domain tasks when supported by the runtime.
- Vendor-specific agent config files such as `claude.md`, `.cursorrules`, and `/.codex/config.toml` require explicit prior user authorization before creation or modification.
- Security rules live directly under `templates/rules/` as `security-*.yaml`.
- Java/Kotlin rules apply to backend and mobile; Swift applies to mobile; C# applies to backend and MAUI/Xamarin-style mobile work.
- `SNAPSHOT.md` and `SPEC.md` remain manual, agent-assisted preserved project documents outside the default scaffold flow; their canonical role definitions live in `README.md`.

## 6. Testing And Validation
- `make test` and `./scripts/test` pass.
- `make lint-shell` and `./scripts/lint-shell` are available as optional maintainer checks when `shellcheck` exists on `PATH`; end users do not need it.
- `make smoke-install` and `./scripts/smoke-install` provide an isolated install + `doctor` smoke check for release confidence.
- `scripts/test` prefers vendored Bats under `tests/vendor/bats` and falls back to system `bats`.
- Current tests verify:
  - AGENTS sections and root/template alignment
  - skills validation and `AVAILABLE_SKILLS.xml`
  - prompt generation without policy duplication
  - security rule IDs and required fields
  - install/uninstall flows
  - regression guards for removed legacy entrypoints and flags
  - snapshot/spec helper behavior
  - legacy prompt-script detection in `doctor`

## 7. Constraints And Risks
- Bash-centric and Unix-focused; no Windows support.
- Dependency governance is expressed as policy and tests, not as hard CLI enforcement on package files.
- Update checks depend on git/curl/network availability, but failures degrade to warnings.
- Templates are copied into projects; existing files are skipped unless force/restore paths are used.
- local custom files under `rules/` or `skills/` can be removed by `afs add-agent --force` while it reconciles the managed scaffold.
- On macOS, downloaded files may still inherit host OS restrictions outside `com.apple.quarantine`; the installer mitigates common quarantine cases but cannot override system-level execution policy.

## 8. Last Updated
- March 24, 2026 (workspace state after release v1.0.22; unreleased changes present)
