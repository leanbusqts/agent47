# CHANGELOG
## [1.0.17] - 2026-03-13
### Changed
- Updated the root and template `AGENTS.md` policy so vendor-specific agent config files such as `claude.md`, `.cursorrules`, `/.codex/config.toml`, and similar files for tools like Codex, Claude Code, Cursor, or Windsurf require explicit prior user authorization instead of being categorically forbidden.

## [1.0.16] - 2026-03-13
### Fixed
- Replaced absolute local filesystem links in `README.md` documentation with repository-relative links so they work outside the original author machine.
- Changed `./install.sh` to link `~/bin/a47` to an installed launcher in `~/.agent47/bin/a47` instead of symlinking directly into the git checkout.
- Cleared macOS quarantine attributes from installed launchers and helper scripts when possible to reduce `Operation not permitted` failures after downloading the repo on another Mac.

### Changed
- Updated install/doctor test coverage to validate the installed launcher layout and `~/bin/a47` symlink behavior.

## [1.0.15] - 2026-03-11
### Added
- `a47 add-cli-prompt`, a terminal-first helper that copies a one-line prompt to the clipboard when available.

### Changed
- Renamed `add-prompt` to `add-agent-prompt` to keep prompt-related commands under a consistent `add-*-prompt` naming scheme.
- Stopped `a47 add-agent` from copying the general agent prompt by default; prompt files are now opt-in helpers.
- Removed `SNAPSHOT.md` references from the public agent contract and default prompt flow.

### Removed
- The ambiguous `add-prompt` command name.

## [1.0.14] - 2026-03-11
### Added
- Project bootstrap around `AGENTS.md`, layered `rules/*.yaml`, curated skills, `skills/AVAILABLE_SKILLS.xml`, `specs/spec.yml`, and a single general `agent-prompt.txt`.
- CLI maintenance flows for install, upgrade, uninstall, doctor checks, template backups, update checks, and focused project commands such as `add-spec`, `add-agent-prompt`, and `add-snapshot-prompt`.
- Security rule coverage split by global, JavaScript/TypeScript, Python, Java/Kotlin, Swift, and C# concerns, with mobile applicability for Android and MAUI/Xamarin-oriented stacks.
- Shared CLI internals under `scripts/lib/` and Bats-based unit coverage for bootstrap, refresh, prompts, templates, policy validation, and snapshot helper behavior.
- Dedicated documentation pages in `docs/usage.md` and `docs/architecture.md`, including an ASCII diagram of the repository structure and execution flow.

### Changed
- Simplified project bootstrap so `a47 add-agent` now installs the full managed scaffolding by default instead of requiring `--with-skills` and `--prompt`.
- Added `a47 add-agent --force` as the refresh path for older projects while preserving `README.md`, `specs/spec.yml`, and `SNAPSHOT.md`.
- Reduced policy duplication by consolidating prompt usage around one general prompt and keeping `AGENTS.md` as the authoritative contract.
- Refactored `bin/a47` into a thinner router backed by shared modules and hardened bootstrap scripts with strict shell mode to fail fast on copy/update errors.
- Shortened `README.md` into an entry document and moved detailed operational and structural guidance into `docs/`.

### Removed
- Legacy prompt variants and the old `add-agent --with-skills --prompt` bootstrap flow.
- Redundant documentation sprawl from the main `README.md` in favor of focused docs pages.
