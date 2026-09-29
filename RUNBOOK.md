# RUNBOOK

Operational guide for `agent47` 2.0 Lite.

## Install and verify

Unix-like systems:

```bash
./install.sh
./install.sh --force
./install.sh --non-interactive
```

Windows:

```powershell
.\install.ps1
.\install.ps1 -Force
.\install.ps1 -NonInteractive
```

```bash
afs version
afs doctor
```

The supported install entrypoints are `install.sh` and `install.ps1`; installation is not a public `afs` subcommand.
Installing from a source checkout requires Go 1.26 or newer unless `AGENT47_GO_CLI` or `AGENT47_REPO_CLI` points to an explicit precompiled launcher.

## Inspect a repository

The basic scan is fast and read-only:

```bash
afs analyze
afs analyze --evidence
afs analyze --json
```

Use deep mode when evaluating agent readiness or migrating an existing repository:

```bash
afs analyze --deep
afs analyze --deep --verbose
afs analyze --deep --json
```

Deep mode inspects project-local policy and documentation without executing repository code. Findings include stable IDs, severity, evidence, and a concrete recommendation. The scan has file, byte, depth, and finding bounds; truncated results are reported explicitly and cannot produce a `ready` result. Historical/rejected sections and non-literal path patterns are excluded from stale-reference findings.

## Refresh repository context

Synchronize the bounded CodeGraph Lite artifact:

```bash
afs map
```

The command creates or refreshes `.agent47/context.md` from repository structure without executing repository code. A second run is a no-op when both structural fingerprint and rendered body are current. A recognized, unmodified map updates automatically after structural or renderer changes.

If the file was edited manually or is not a recognized generated context, normal refresh stops. Replace it only with explicit authorization:

```bash
afs map --force
```

There is no `--preview`, freshness flag, watcher, daemon, or hook. The target is one generated file; body integrity and the explicit force boundary protect user content. A symlinked `.agent47` or `context.md`, a runtime-home collision, hostile path, or concurrent modification is rejected.

## Initialize policy

Review the exact plan first:

```bash
afs init --preview
```

Apply automatic detection:

```bash
afs init
```

Select or exclude bundles explicitly:

```bash
afs init --bundle cli --bundle scripts --preview
afs init --bundle cli --exclude-bundle scripts
```

Refresh managed files:

```bash
afs init --force --preview
afs init --force
```

Without `--force`, existing `AGENTS.md`, rules, and legacy content are kept. Init creates `.agent47/context.md` when missing and includes it in the same rollback-capable transaction; both normal and forced init preserve any existing copy. `--force` is destructive only toward the legacy contract: it replaces all of `rules/`, removes `skills/` and `prompts/`, and removes `specs/spec.yml` plus `.agents/specs/spec.yml`. Run `afs init --force --preview` first when the repository may contain custom content in those paths. README, `SNAPSHOT.md`, root `SPEC.md`, other `.agents/` or `specs/` files, existing project context, and unrelated paths remain untouched. Preview also reports nested-policy, vendor-policy, and composition warnings. Apply revalidates identity and content; failure or cancellation restores staged paths when safe and retains/reports recovery backups on a concurrent-edit conflict.

Initialization is non-interactive. `--preview` never writes; running without it applies the displayed plan. `--dry-run` is a compatibility alias for `--preview` but is omitted from help.

## Diagnose the installation

```bash
afs doctor
afs doctor --json
afs doctor --fail-on-warn
afs doctor --check-update
afs doctor --check-update-force
afs doctor --check-update --fail-on-warn
```

Normal `doctor` runs do not access the network. `--check-update` opts into update resolution; `--check-update-force` may fetch the tracked git remote.
When combined with an update check, `--fail-on-warn` also treats an unavailable or diverged update source as a warning failure.
Doctor JSON uses `schema_version: 1`, keeps stdout valid JSON, and reports `ok`, `warning`, or `error` plus checks and warnings.

## Uninstall

```bash
afs uninstall
```

Uninstall removes the managed runtime, templates, and the published `afs` entry. Legacy helper entries are removed only when they can be identified as managed agent47 artifacts; unrelated files with the same names are preserved.

The runtime home must be a dedicated, non-symlinked directory and carries an Agent47 ownership marker. Uninstall preserves an unowned runtime directory. Forced-install template backups carry a content digest; any backup changed after creation is preserved rather than removed.

## Exit codes and streams

- `0`: success; analysis findings do not make the command fail
- `1`: operational failure
- `2`: invalid flags, arguments, or unknown command

Primary output goes to stdout. Diagnostics and failures go to stderr.

## Maintainer verification

After a fresh Git clone, initialize the pinned Bats dependency:

```bash
git submodule update --init --recursive
```

```bash
make agents-check
make rules-check
make rules-drift-check
make go-test
make go-build
make coverage
make lint-shell
make smoke-install
make test
```

Run `make test` and `make coverage` before release. The test target includes policy/rule validation, checkout tests, and installed-artifact verification; coverage enforces the same global and per-package floors as CI.
