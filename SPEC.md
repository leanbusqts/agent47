# SPEC — agent47 Lite

## 1. Product direction

`agent47` is a lightweight, vendor-neutral repository harness for coding agents. Its job is to make repository instructions and structural context discoverable, bounded, and verifiable without becoming an orchestration framework or a large prompt distribution system.

The product optimizes for:

1. the smallest practical public command surface;
2. explicit repository policy;
3. safe and deterministic inspection;
4. conservative, reversible initialization;
5. minimal context added to an agent session.

## 2. Repository contract

The generated contract contains:

- root `AGENTS.md`;
- applicable known `rules/*.yaml` templates;
- generated `.agent47/context.md`, a bounded CodeGraph Lite artifact derived from repository evidence.

`AGENTS.md` and rules are stable policy. `.agent47/context.md` is repository-specific evidence, is safe to version, and is never authoritative over source or policy. It is generated rather than copied from a template.

The following are outside the generated contract:

- `README.md`, `SNAPSHOT.md`, and root `SPEC.md`;
- `.agents/`, except the legacy `.agents/specs/spec.yml` migration target;
- `specs/`, except the legacy `specs/spec.yml` migration target;
- any path unrelated to the legacy `rules/`, `skills/`, and `prompts/` namespaces.

Normal initialization preserves all existing out-of-contract content. The explicit `--force` migration removes the legacy Agent47 namespaces and exact task files listed above before installing the current minimal contract.

Agent policy requires the strongest internal reasoning language available, defaults technical work to English internally, and preserves the output language requested by the user.

## 3. Public CLI

The complete public surface is:

```text
afs help
afs version
afs analyze [--json] [--verbose] [--evidence] [--deep]
afs map [--force]
afs init [--bundle NAME ...] [--exclude-bundle NAME ...] [--force] [--preview]
afs doctor [--json] [--check-update|--check-update-force|--fail-on-warn]
afs uninstall
```

`afs init --dry-run` is a hidden alias for `--preview`.

Public exit codes:

- `0`: successful execution, including analysis that reports findings;
- `1`: operational failure;
- `2`: invalid usage or unknown command.

Primary results use stdout; diagnostics use stderr.

## 4. Analyze

`afs analyze` is read-only and deterministic. It detects repository types and technologies, resolves bundles and rule templates, and may emit raw or classification evidence.

`--deep` adds an agent-readiness audit. It recognizes project-local, scoped policy for common agent ecosystems while remaining vendor-neutral. Detection of a vendor file is evidence, not permission to create or modify vendor configuration.

Deep analysis evaluates at least:

- instruction discoverability and nested scope;
- verification commands and test/build documentation;
- contradictions, duplication, and oversized policy;
- repository documentation and structure relevant to autonomous work;
- common noise sources and unsafe guidance.

Every finding has a stable rule ID, severity, title, evidence, explanation, and recommendation. The audit obeys explicit bounds for traversal depth, files, bytes, and findings, reports truncation, ignores common generated/vendor directories, never executes repository code, and never prints secret values.

Historical migration/rejection material is not a live contract, and glob or placeholder paths are not interpreted as exact stale references. Any applied inspection truncation prevents the overall result from being `ready`; only prompt-surface uncertainty affects the `policy.consistency` claim.

The deterministic V1 audit rules are:

- `PA-001`: exact or safely normalized duplicate instructions;
- `PA-002`: same-authority structured conflicts;
- `PA-007`: explicit vendor/model coupling inside a vendor-neutral contract;
- `PA-008`: deterministically stale paths, commands, or managed references.

## 5. Repository map and initialization

### 5.1 Map

`afs map` scans the current repository without executing its code and synchronizes `.agent47/context.md`. The map is intentionally bounded: it summarizes components, package/workspace roots, entrypoints, direct local relationships with confidence and evidence, tests, manifests/configuration, applicable policy, and allow-listed verification commands. It excludes complete symbol/call graphs, external dependencies, embeddings, databases, daemons, watchers, LSP/MCP services, and repository-code execution.

The first line contains machine-readable `afs-context` metadata with `schema_version`, `source_fingerprint`, and `body_sha256`. The source fingerprint covers normalized structural observations and a renderer version; the body hash detects manual edits. `.agent47/` and common generated/vendor directories are excluded from the scan so the output cannot fingerprint itself.

Rules:

1. A missing context is created.
2. A recognized, unchanged context is left untouched when the candidate is identical.
3. Structural or renderer changes regenerate an unmodified context.
4. A manually modified or unrecognized context is rejected unless `--force` is supplied.
5. `--force` replaces only `.agent47/context.md`; it does not broaden repository ownership.
6. Output is deterministic, size-bounded, secret-safe, and treats discovered paths and commands as evidence rather than instructions.
7. Paths and command names are allow-listed before rendering to prevent generated Markdown from carrying repository-controlled instructions.
8. There is no `--preview`, separate freshness command, background watcher, daemon, or Git hook. Idempotence, hashes, and the narrow single-file target provide the safety boundary.

The generated `AGENTS.md` makes synchronization automatic from the user's perspective: for non-trivial code tasks an agent runs `afs map` from the repository root before broad exploration, then reads `.agent47/context.md`; after structural changes it runs `afs map` again. Trivial tasks skip it. If terminal access or `afs` is unavailable, the agent may use the existing context but must report that it may be stale. If synchronization fails because the context was edited, the agent must not force or trust it without explicit user authorization.

Together, `AGENTS.md`, applicable `rules/*.yaml`, and `.agent47/context.md` form the declarative Harness Lite: stable operating policy, stack constraints, and fresh repository evidence. The harness does not execute agents or verification commands.

### 5.2 Init

`afs init` analyzes the current directory, resolves the selected bundles, prints a deterministic plan, and optionally applies it.

Plan groups are `create`, `update`, `keep`, and `remove`. The removal group is empty unless `--force` is supplied and legacy Agent47 content is present.

Rules:

1. Missing managed targets and `.agent47/context.md` are created in one rollback-capable operation.
2. Existing managed targets are kept by default. Existing `.agent47/context.md` is always preserved by init, including `init --force`.
3. `--force` replaces `rules/` with exactly the resolved current rule set and removes the complete legacy `skills/` and `prompts/` namespaces.
4. `--force` also removes the known legacy task files `specs/spec.yml` and `.agents/specs/spec.yml`, pruning their parent directories only when empty.
5. Repository documentation, other `.agents/` and `specs/` content, vendor configuration, source, and all unrelated paths are preserved.
6. `--preview` and `--dry-run` never write.
7. Initialization is non-interactive; absence of `--preview` is the explicit request to apply the displayed plan.
8. Unsafe roots, symlinked cleanup parents, and path-escape conditions are rejected.
9. Writes and legacy removals are staged; a partial operation is rolled back on failure or cancellation.
10. Preview reports relevant nested-policy, vendor-policy, traversal, and bundle-conflict warnings.
11. Commit, cleanup, and rollback revalidate path identity and content so concurrent user edits are never silently overwritten.

Supported project bundles are `frontend`, `backend`, `mobile`, `cli`, `scripts`, `infra`, `monorepo-tooling`, `desktop`, and `plugin`, plus shared dependencies and the base bundle. An unresolved automatic composition falls back to the base bundle; incompatible explicit compositions fail.

## 6. Installation and maintenance

`install.sh` and `install.ps1` are the public installation entrypoints. The installed payload includes one `afs` executable and the policy/rule templates. It does not publish helper commands.

`afs doctor` validates the executable, installed template manifest, required files/directories, rule/security templates, policy sections, PATH integration, and optional update state. `--json` emits a versioned machine-readable health report.

The runtime home is a dedicated non-symlinked directory with an explicit ownership marker. `afs uninstall` removes only managed runtime artifacts. Legacy helper names may be cleaned up only when ownership can be proven; unmanaged entries are preserved. Template backups are removed only while both their ownership marker and recorded digest remain valid; modified or unverified backups are preserved.

## 7. Explicit non-goals

The Lite core does not provide:

- skills or skill indexes;
- task-spec scaffolding;
- prompt or clipboard helpers;
- vendor-specific config generation;
- subagent orchestration, memory, queues, daemons, or a dashboard;
- speculative “spec grill” workflows;
- repository-code execution during analysis;
- dependency installation in a target repository.

Deep analysis is diagnostics, not a second framework. Future extensions must justify their command, dependency, context, and maintenance cost.

## 8. Verification contract

Maintainer checks are:

```bash
make agents-check
make rules-check
make rules-drift-check
make go-test
make go-build
make lint-shell
make smoke-install
make test
```

Policy mirrors and security-rule mirrors must remain byte-identical. Public behavior must be covered at unit, checkout, and installed-artifact levels.
