# CHANGELOG
## [1.0.14] - 2026-03-11
### Added
- Project bootstrap around `AGENTS.md`, layered `rules/*.yaml`, curated skills, `skills/AVAILABLE_SKILLS.xml`, `specs/spec.yml`, and a single general `agent-prompt.txt`.
- CLI maintenance flows for install, upgrade, uninstall, doctor checks, template backups, update checks, and focused project commands such as `add-spec`, `add-prompt`, and `add-snapshot-prompt`.
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
