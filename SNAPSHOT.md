# SNAPSHOT

## Product

- **Version:** 2.0.4
- **Direction:** lightweight, vendor-neutral harness for agent-ready repositories
- **Generated surface:** `AGENTS.md`, applicable known `rules/*.yaml`, and `.agent47/context.md`
- **Public command count:** seven (`help`, `version`, `analyze`, `map`, `init`, `doctor`, `uninstall`)

## Current behavior

- `afs analyze` detects project types, technologies, bundles, and rules without writing.
- Generated policy combines pragmatic engineering principles with proportionate analysis, reversible decision-making, and an explicit completion boundary.
- `afs analyze --deep` audits project-local agent policy and repository readiness with stable findings and bounded, secret-safe traversal.
- Deep drift checks ignore historical/rejected contracts and non-literal path patterns while retaining explicit truncation evidence.
- `afs map` maintains a deterministic, bounded CodeGraph Lite with components, entrypoints, direct local relationships, tests, policies, manifests, and allow-listed verification commands.
- Structural fingerprints provide freshness; body hashes protect manual changes, and only `afs map --force` replaces modified or unrecognized context.
- `afs init` replaces `add-agent` with deterministic create/update/keep/remove planning.
- Init creates missing project context inside its policy transaction and preserves existing context even during forced migration.
- `afs init --force` migrates older scaffolds by replacing `rules/`, deleting `skills/` and `prompts/`, and removing the two known legacy task-spec files.
- Init commits are confined to the opened repository and reject concurrent path/content changes; rollback never overwrites a later user edit.
- Initialization is non-interactive: `--preview` emits a `Preview`, while its absence emits and applies a `Plan`.
- `afs doctor` verifies the installed executable and Lite template contract, supports versioned JSON output, and keeps update checks opt-in.
- Installers publish only `afs` and clean up old helpers only when they are identifiable as managed artifacts.
- Runtime ownership is explicit; unsafe homes are rejected and modified/unverified template backups survive uninstall.
- Windows self-uninstall defers only the locked executable and final runtime-directory removal to a temporary owned helper after the parent exits.

## Removed from the core

- built-in skills and skill indexes;
- `.agents/specs/spec.yml` scaffolding;
- prompt and clipboard helpers;
- the `add-agent` command and helper executable aliases;
- separate cleanup commands or migration flags beyond `--force`.

Normal init preserves existing legacy content. Explicit `--force` removes the old Agent47-owned namespaces while preserving repository documentation, unrelated paths, and other content inside `.agents/` or `specs/`.

## Source layout

- `cmd/afs` — native CLI entrypoint
- `internal/analyze` — repository detection and readiness analysis
- `internal/contextmap` — CodeGraph Lite generation, metadata, and freshness decisions
- `internal/initrepo` — safe repository initialization
- `internal/install`, `internal/doctor`, `internal/update` — lifecycle support
- `templates/base` and `templates/bundles` — policy and rule payload

## Verification

The supported checks are `make test`, `make agents-check`, `make rules-check`, `make rules-drift-check`, `make go-test`, `make go-build`, `make coverage`, `make lint-shell`, and `make smoke-install`. Source builds require Go 1.26 or newer, and fresh maintainer clones initialize the pinned Bats submodule.
