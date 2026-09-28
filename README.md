# agent47

`agent47` is a lightweight, vendor-neutral harness for preparing repositories to work well with coding agents. It installs one CLI, `afs`, and keeps the repository contract deliberately small: `AGENTS.md`, applicable `rules/*.yaml`, and a generated `.agent47/context.md` CodeGraph Lite.

It does not install skills, prompts, task-spec templates, memory services, orchestration platforms, or vendor-specific agent configuration.

## Quickstart

```bash
./install.sh
cd /path/to/project
afs analyze --deep
afs init --preview
afs init
afs map
```

On Windows:

```powershell
.\install.ps1
```

Verify the installation with:

```bash
afs version
afs doctor
```

## Public commands

```text
afs help
afs version
afs analyze [--json] [--verbose] [--evidence] [--deep]
afs map [--force]
afs init [--bundle NAME ...] [--exclude-bundle NAME ...] [--force] [--preview]
afs doctor [--json] [--check-update|--check-update-force|--fail-on-warn]
afs uninstall
```

`--dry-run` remains a hidden alias for `afs init --preview`. Unknown commands and invalid usage exit with status `2`; operational failures exit with status `1`.

Commands intentionally removed in 2.0 include `add-agent`, `add-agent-prompt`, and `add-ss-prompt`. The installer no longer publishes helper executables for them.

## Analyze

`afs analyze` is read-only. It detects project types and technologies, resolves the applicable rule bundles, and reports the resulting install set.

`afs analyze --deep` additionally audits whether the repository is ready for effective agent-assisted work. It examines project-local policy files, nested policy scope, useful verification commands, documentation signals, conflicting or duplicated guidance, and common sources of unnecessary agent context. Deep analysis is bounded, deterministic, secret-safe, and never executes repository code. Reported truncation makes the overall result at most `partial`; historical migration text and path patterns such as `rules/*.yaml` are not treated as live stale references.

JSON output is additive within its declared schema version. Incompatible deep-analysis schema changes increment `analysis_version`; doctor JSON uses `schema_version`.

Examples:

```bash
afs analyze
afs analyze --evidence
afs analyze --deep --verbose
afs analyze --deep --json
```

## Map and Harness Lite

`afs map` safely scans repository structure and synchronizes `.agent47/context.md`. The bounded map records components, package/workspace roots, entrypoints, direct local relationships, tests, manifests/configuration, applicable policy, and allow-listed verification commands. It never executes repository code and does not build a complete symbol or call graph.

Freshness is built into the command. Generated metadata stores a structural fingerprint and a body hash: an unchanged map is a no-op, structural or renderer changes update an unmodified map, and manual or unrecognized content requires explicit `afs map --force`. No watcher, daemon, hook, `--preview`, or separate freshness command is involved.

The generated `AGENTS.md` tells capable agents to run `afs map` before broad exploration for non-trivial code tasks and again after structural changes. Trivial tasks skip it. Without terminal access or an available `afs`, agents use the existing map with a stale-context warning. Source and policy always override the generated context.

## Init

`afs init` writes only:

- `AGENTS.md`
- applicable known files under `rules/*.yaml`
- generated `.agent47/context.md` when it is missing

Existing managed files are kept unless `--force` is supplied. Existing `.agent47/context.md` is preserved even by `init --force`; use `afs map --force` for an explicitly authorized replacement. `afs init --force` is the migration path from older Agent47 scaffolds: it replaces the complete `rules/` namespace, removes `skills/` and `prompts/`, and removes the known legacy task files `specs/spec.yml` and `.agents/specs/spec.yml`. It preserves `README.md`, `SNAPSHOT.md`, root `SPEC.md`, other `.agents/` or `specs/` content, and every unrelated repository path.

The command always prints a deterministic `create`/`update`/`keep`/`remove` plan before writing. `afs init` is deliberately non-interactive: use `--preview` to inspect the exact target list, then run the command without `--preview` to apply it. Forced cleanup and managed writes form one rollback flow; an error or cancellation restores staged legacy paths unless a concurrent edit makes restoration unsafe, in which case the recovery path is retained and reported.

```bash
afs init --preview
afs init --bundle cli --bundle scripts --preview
afs init --exclude-bundle scripts
afs init --force
```

## Installation model

The native `afs` binary and template payload are installed under the dedicated `~/.agent47` directory on Unix-like systems or `%LOCALAPPDATA%\agent47` on Windows. Installing directly from a source checkout requires Go unless an explicit precompiled launcher is provided. Unsafe broad or symlinked runtime paths are rejected, and an ownership marker prevents uninstall from claiming an unrelated directory. Unix-like installs publish only `~/bin/afs`; Windows uses the managed bin directory. Forced template backups are deleted only while their ownership marker and content digest still match; modified or unverified backups are preserved.

`templates/manifest.txt` defines ownership:

- managed: `AGENTS.md`, known `rules/*.yaml`
- generated: `.agent47/context.md`
- preserved by normal init: repository documentation and existing `.agents/`, `skills/`, or `prompts/` content
- force cleanup: `rules/`, `skills/`, `prompts/`, `specs/spec.yml`, and `.agents/specs/spec.yml`

## Development

```bash
make test
make agents-check
make rules-check
make rules-drift-check
make go-test
make go-build
make lint-shell
make smoke-install
```

The current product contract lives in [SPEC.md](SPEC.md), operational details are in [RUNBOOK.md](RUNBOOK.md), and the implemented state is summarized in [SNAPSHOT.md](SNAPSHOT.md).
