# SNAPSHOT

## 1. Project Overview
- **Name:** agent47
- **Purpose:** Bash CLI plus templates for setting up agent-driven development workflows with explicit policy, rules, skills, prompts, and optional specs.

## 2. Current Status
- **Implemented:** CLI in `bin/a47` with install/upgrade/uninstall, doctor, update checks, template backup/restore, and project bootstrap commands; curated templates for `AGENTS.md`, stack rules, security rules, skills, prompts, and `specs/spec.yml`; Bats-based unit test suite and Make targets.
- **Architecture:** `bin/a47` is now mostly a router and sources shared logic from `scripts/lib/` (`constants`, `update`, `templates`, `install`, `doctor`, `common`).
- **Documentation:** `README.md` is now intentionally short; usage and structure details live in `docs/usage.md` and `docs/architecture.md`.
- **Stable workflow:** `a47 add-agent` bootstraps `AGENTS.md` and all `rules/*.yaml` plus skills; `a47 add-cli-prompt`, `a47 add-agent-prompt`, and `a47 add-snapshot-prompt` are available as focused helpers.
- **Operational hardening:** core helper scripts now run with strict shell mode to reduce silent partial failures during bootstrap/update flows.
- **Not automated by CLI:** `SNAPSHOT.md` creation/update, vendor-specific configs, Windows/PowerShell support, dependency enforcement against concrete package manifests.

## 3. Current Commands
- `a47 install [--force]`
- `a47 upgrade [--force]`
- `a47 uninstall`
- `a47 doctor`
- `a47 check-update [--force]`
- `a47 templates --restore-latest|--list|--clear-backups`
- `a47 add-agent [--force]`
- `a47 add-spec`
- `a47 add-skills`
- `a47 reload-skills`
- `a47 add-cli-prompt`
- `a47 add-agent-prompt`
- `a47 add-snapshot-prompt`

## 4. Key Repository Structure
- `AGENTS.md` – live root policy file for this repo.
- `docs/usage.md` – operational guide for install, bootstrap, and refresh.
- `docs/architecture.md` – architecture notes and ASCII project map.
- `bin/a47` – main CLI router and maintenance logic.
- `install.sh` – installer that links `a47` into `~/bin`.
- `scripts/`
  - `add-agent`
  - `add-cli-prompt`
  - `add-agent-prompt`
  - `add-snapshot-prompt`
  - `add-skills`
  - `add-spec`
  - `reload-skills`
  - `test`
  - `skill-utils.sh`
  - `lib/*.sh`
- `templates/`
  - `AGENTS.md`
  - `prompts/cli-prompt.txt`
  - `prompts/agent-prompt.txt`
  - `prompts/snapshot-prompt.txt`
  - `rules/rules-frontend.yaml`
  - `rules/rules-backend.yaml`
  - `rules/rules-mobile.yaml`
  - `rules/security-*.yaml`
  - `specs/spec.yml`
  - `skills/<name>/SKILL.md`
- `tests/unit/` – Bats unit coverage for CLI, prompts, policy, skills, backups, and snapshot helper.

## 5. Policy And Rules Model
- Authority order: user > nearest `AGENTS.md` > security rules > stack rules > spec > code/tests > memories.
- `AGENTS.md` is the single source of policy; prompts and README should reference it rather than duplicating policy.
- Security rules live directly under `templates/rules/` as `security-*.yaml`.
- Java/Kotlin rules apply to backend and mobile; Swift applies to mobile; C# applies to backend and MAUI/Xamarin-style mobile work.
- `SNAPSHOT.md` remains a manual project note, outside the public agent contract and default prompt flow.

## 6. Testing And Validation
- `make test` and `./scripts/test` pass.
- `scripts/test` prefers vendored Bats under `tests/vendor/bats` and falls back to system `bats`.
- Current tests verify:
  - AGENTS sections and root/template alignment
  - skills validation and `AVAILABLE_SKILLS.xml`
  - prompt generation without policy duplication
  - security rule IDs and required fields
  - install/upgrade/uninstall and template backup flows
  - snapshot helper behavior
  - legacy prompt-script detection in `doctor`

## 7. Constraints And Risks
- Bash-centric and Unix-focused; no Windows support.
- Dependency governance is expressed as policy and tests, not as hard CLI enforcement on package files.
- Update checks depend on git/curl/network availability, but failures degrade to warnings.
- Templates are copied into projects; existing files are skipped unless force/restore paths are used.

## 8. Last Updated
- March 11, 2026
