# CHANGELOG
## [Unreleased]

## [1.2.0] - 2026-04-29
### Added
- Added `afs analyze` with concise, verbose, evidence, and JSON output backed by the new repository analysis and install-resolution services.
- Added bundle-native template assembly under `templates/base/` and `templates/bundles/`, including project bundles for frontend, backend, mobile, cli, scripts, infra, monorepo-tooling, desktop, and plugin, plus shared bundles for CLI behavior and testing.
- Added generated skills indexes at `skills/AVAILABLE_SKILLS.json` and `skills/SUMMARY.md` alongside the existing XML catalog.

### Changed
- Made `afs add-agent` analysis-driven, with preview/dry-run flows, `--yes`, explicit bundle selection, adaptive bundle resolution, and assembled-contract bootstrap behavior.
- Migrated the scaffold payload from the legacy flat `templates/` tree to the bundle-native layout while preserving the manifest-driven ownership contract.
- Expanded `afs doctor` and install verification to validate bundle assembly, assembled template payload, and the newer managed skills indexes.
- Updated `README.md`, `RUNBOOK.md`, `SNAPSHOT.md`, and `SPEC.md` to describe the bundle-native runtime and current command surface.

### Fixed
- Hardened `--force` migration semantics so stale managed rules and skills are removed according to the resolved assembled contract while preserved project files remain untouched.
- Improved bootstrap and prompt error handling with clearer typed template and bundle-manifest failures.
- Tightened policy and installed-artifact verification so docs, manifests, skills indexes, and bundle-owned templates stay aligned with runtime behavior.

### Removed
- Removed the legacy flat template payload files in favor of `templates/base/` plus `templates/bundles/`.
- Removed deprecated historical planning files `PLAN.md` and `SPEC_IMPROVEMENTS.md` from the active repository surface.

## [1.1.1] - 2026-03-27
### Fixed
- Hardened `afs add-agent` so it no longer risks deleting a preexisting non-directory `skills` path on bootstrap failure, and added regression coverage for that collision case.
- Aligned bootstrap behavior with the product contract by creating `specs/spec.yml` when missing and preserving it across refreshes.
- Strengthened `afs doctor` so it validates the full installed template payload contract, including stack rule templates, `prompts/ss-prompt.txt`, `templates/specs/spec.yml`, and exact manifest sections for required files and directories.

### Changed
- Updated README, SPEC, usage docs, and doctor/bootstrap test fixtures to reflect the current scaffold contract and stricter doctor output.

## [1.1.0] - 2026-03-26
### Changed
- Migrated the public product runtime fully to the Go CLI under `cmd/afs` and `internal/*`, with `install.sh` and `install.ps1` acting as thin wrappers around the native install service.
- Reworked the repo-local launcher so `bin/afs` now prefers explicit compiled CLI overrides, resolves symlinks safely, and rejects invalid explicit launcher paths instead of falling back silently.
- Expanded installed-artifact verification to cover `doctor --check-update`, `add-agent --force`, and `add-agent --only-skills --force`, and tightened verification isolation so it depends less on host PATH state.
- Strengthened cross-platform runtime behavior for path matching, Windows file replacement rollback, context-aware install/bootstrap cancellation, and stricter PowerShell launcher override handling.
- Updated `README.md`, `SNAPSHOT.md`, `SPEC.md`, `docs/`, and `PLAN.md` to reflect the completed Bash-to-Go migration and current public command surface.
- Promoted root `SPEC.md` to a preserved project document during `afs add-agent --force`, aligning the manifest, tests, and documentation with the product contract.
- Renamed the non-interactive install flag from `--no-prompt` to `--non-interactive` across runtime, tests, smoke install, and documentation.
- Added a compact document-role taxonomy across docs to distinguish `SNAPSHOT.md`, root `SPEC.md`, and `specs/spec.yml`.

### Removed
- Removed the legacy `init-agent` compatibility path from `bin/afs` and its dedicated test coverage.

## [1.0.22] - 2026-03-20
### Changed
- Renamed the snapshot helper flow to `afs add-ss-prompt`, renamed the installed prompt template to `templates/prompts/ss-prompt.txt`, and updated docs/tests/runtime references accordingly.
- Expanded the `add-ss-prompt` template so it guides agents to generate or update both `SNAPSHOT.md` and `SPEC.md` instead of only snapshot-style documentation.
- Clarified across `README.md`, `SPEC.md`, `SNAPSHOT.md`, `docs/`, and `AGENTS.md` that root `SPEC.md` is the current-state product spec for `agent47` itself, while task-specific planning work should live in `specs/spec.yml`.
- Added guidance in root and template `AGENTS.md` suggesting that existing `SNAPSHOT.md` and `SPEC.md` files be updated before commit when scoped work materially changes them.

## [1.0.21] - 2026-03-15
### Changed
- Clarified the managed-ownership model across `README.md`, `SNAPSHOT.md`, and docs so `afs add-agent --force` is explicit about `rules/` and `skills/` being fully managed refresh targets rather than mixed-ownership directories.

## [1.0.20] - 2026-03-14
### Fixed
- Hardened installation so it now fails fast on permission errors and on missing core install assets from the source checkout instead of reporting a false success.
- Made `add-agent` abort before writing any project files when required skills helper dependencies are missing, preventing partial bootstrap state.
- Rejected unexpected arguments across scaffolding scripts such as `add-agent`, `add-agent-prompt`, and the snapshot/spec prompt helper flow.
- Ensured unknown `afs` commands return a non-zero exit code.
- Fixed template restore behavior under strict shell mode so `templates --restore-latest` now reports missing backups correctly and restore tests are isolated.
- Made skill installation derive from the actual template tree instead of a hardcoded list, so new curated skills are picked up automatically.
- Prevented `add-agent-prompt` from creating `prompts/` when the source template is missing.
- Hardened update-cache decoding so corrupted cache entries now fall back cleanly instead of poisoning later checks.

### Changed
- Simplified the public install surface so `./install.sh` is now the only supported installation entrypoint; `afs install` and `afs upgrade` are no longer exposed.
- Removed `afs add-spec` from the public CLI; spec creation is now an agent-driven workflow around `specs/spec.yml`, with an explicit user review step and optional multi-agent review when supported.
- Removed `afs add-cli-prompt`; the minimal bootstrap text for CLIs and IDEs now lives in documentation instead of the public CLI.
- Removed `afs templates` from the public CLI; template backups remain automatic and recovery is now documented as a manual maintenance step.
- Removed `afs check-update` from the public CLI; update checks now live only under `afs doctor --check-update`.
- Folded `add-default-skills` into `afs add-agent --only-skills [--force]` and updated help text to explain each `add-agent` variant inline.
- Updated `doctor` usage to make update checks opt-in via `--check-update` and `--check-update-force`.
- Switched the update cache filename from `update.json` to `update.cache` to reflect the on-disk format.
- Improved the test runner to install a temporary `bats` copy automatically when needed.
- Expanded unit coverage for CLI error handling, install preflight, argument parsing, dynamic skill discovery, update cache round-tripping, and restore flows.
- Clarified the root and template `AGENTS.md` policy for template-source repositories so `agent47` can point to `templates/rules/` without contradicting its own contract.

## [1.0.18] - 2026-03-13
### Changed
- Added `Execution Strategy` guidance to the root and template `AGENTS.md` files to encourage multi-agent or sub-agent workflows for non-trivial work, including an implement-then-review default for higher-risk changes.
- Expanded the escalation guidance to explicitly call out multi-domain tasks across backend, frontend, security, documentation, tests, Android, and iOS.

## [1.0.17] - 2026-03-13
### Changed
- Updated the root and template `AGENTS.md` policy so vendor-specific agent config files such as `claude.md`, `.cursorrules`, `/.codex/config.toml`, and similar files for tools like Codex, Claude Code, Cursor, or Windsurf require explicit prior user authorization instead of being categorically forbidden.

## [1.0.16] - 2026-03-13
### Fixed
- Replaced absolute local filesystem links in `README.md` documentation with repository-relative links so they work outside the original author machine.
- Changed `./install.sh` to link `~/bin/afs` to an installed launcher in `~/.agent47/bin/afs` instead of symlinking directly into the git checkout.
- Cleared macOS quarantine attributes from installed launchers and helper scripts when possible to reduce `Operation not permitted` failures after downloading the repo on another Mac.

### Changed
- Updated install/doctor test coverage to validate the installed launcher layout and `~/bin/afs` symlink behavior.

## [1.0.15] - 2026-03-11
### Changed
- Renamed `add-prompt` to `add-agent-prompt` to keep prompt-related commands under a consistent `add-*-prompt` naming scheme.
- Stopped `afs add-agent` from copying the general agent prompt by default; prompt files are now opt-in helpers.
- Removed `SNAPSHOT.md` references from the public agent contract and default prompt flow.

### Removed
- The ambiguous `add-prompt` command name.

## [1.0.14] - 2026-03-11
### Added
- Project bootstrap around `AGENTS.md`, layered `rules/*.yaml`, curated skills, `skills/AVAILABLE_SKILLS.xml`, `specs/spec.yml`, and a single general `agent-prompt.txt`.
- CLI maintenance flows for install, upgrade, uninstall, doctor checks, template backups, update checks, and focused project commands such as `add-spec`, `add-agent-prompt`, and the snapshot/spec prompt helper flow.
- Security rule coverage split by global, JavaScript/TypeScript, Python, Java/Kotlin, Swift, and C# concerns, with mobile applicability for Android and MAUI/Xamarin-oriented stacks.
- Shared CLI internals under `scripts/lib/` and Bats-based unit coverage for bootstrap, refresh, prompts, templates, policy validation, and snapshot/spec helper behavior.
- Dedicated documentation pages in `docs/usage.md` and `docs/architecture.md`, including an ASCII diagram of the repository structure and execution flow.

### Changed
- Simplified project bootstrap so `afs add-agent` now installs the full managed scaffolding by default instead of requiring `--with-skills` and `--prompt`.
- Added `afs add-agent --force` as the refresh path for older projects while preserving `README.md`, `specs/spec.yml`, and `SNAPSHOT.md`.
- Reduced policy duplication by consolidating prompt usage around one general prompt and keeping `AGENTS.md` as the authoritative contract.
- Refactored `bin/afs` into a thinner router backed by shared modules and hardened bootstrap scripts with strict shell mode to fail fast on copy/update errors.
- Shortened `README.md` into an entry document and moved detailed operational and structural guidance into `docs/` at that time.

### Removed
- Legacy prompt variants and the old `add-agent --with-skills --prompt` bootstrap flow.
- Redundant documentation sprawl from the main `README.md` in favor of focused docs pages.
