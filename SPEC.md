# SPEC

This root `SPEC.md` describes the current product contract and current-state technical behavior of `agent47` itself. It is not a forward-looking feature spec or implementation plan for new work; use `specs/spec.yml` for task-specific spec/plan work when needed.

## 1. Project Identity

- **Name:** `agent47`
- **Type:** Bash-first CLI plus template payload for bootstrapping agent-oriented repository scaffolding.
- **Primary purpose:** install a local CLI runtime and scaffold portable repository conventions for agent workflows without tying the target repository to a vendor-specific agent product.

## 2. Product Summary

`agent47` provides two layers:

1. A local machine CLI runtime installed under `~/.agent47` and exposed through `~/bin/afs`.
2. A template payload copied into target repositories to establish a standard agent contract based on:
   - `AGENTS.md`
   - `rules/*.yaml`
   - `skills/*`
   - `skills/AVAILABLE_SKILLS.xml`
   - prompt helpers
   - an optional `specs/spec.yml` workflow artifact

The project is intentionally conservative:

- bootstrap is explicit
- refresh is opt-in
- `--force` is required for managed-file reconciliation
- project-owned files are preserved during refresh
- policy is centralized in `AGENTS.md`

## 3. Goals

- Provide a small, inspectable, low-dependency bootstrap tool for agent-driven repository workflows.
- Keep policy, stack guidance, security rules, skills, and prompts versioned as normal files inside the target repo.
- Separate repo-owned implementation from template-owned scaffold content.
- Make managed vs preserved ownership explicit and machine-checkable.
- Support agent CLIs and IDE agents through repo conventions instead of vendor-specific config.
- Keep install, refresh, and uninstall flows safe, predictable, and testable.

## 4. Non-Goals

- Full project generation beyond the agent-workflow scaffold.
- Enforcement of package-manager dependencies or lockfiles in target repos.
- Windows or PowerShell-first support.
- Tight coupling to a single agent runtime.
- Hidden remote orchestration, cloud sync, or external service dependency for normal operation.
- Automatic creation of project-specific business specs beyond the optional `specs/spec.yml` workflow pattern.

## 5. Core User Workflows

### 5.1 Install `agent47` locally

Entry point:

- `./install.sh [--force] [--non-interactive]`

Expected outcome:

- install managed runtime under `~/.agent47`
- install launcher at `~/.agent47/bin/afs`
- install helper commands under `~/.agent47/scripts/`
- install template payload under `~/.agent47/templates/`
- link `~/bin/afs` to the managed launcher
- optionally update shell rc configuration to include `~/bin`

### 5.2 Verify local installation

Entry point:

- `afs doctor`
- `afs doctor --check-update`
- `afs doctor --check-update-force`

Expected outcome:

- confirm managed launcher and helpers are on `PATH`
- confirm templates and prompt assets exist
- confirm policy and template integrity checks
- optionally check whether a newer version is available

### 5.3 Bootstrap a target repository

Entry point:

- `afs add-agent`

Expected outcome:

- bootstrap the managed agent scaffold in the current directory
- write `AGENTS.md` if absent
- write `rules/*.yaml` if absent
- create `skills/*` and `skills/AVAILABLE_SKILLS.xml`
- create an empty `README.md` if absent
- preserve existing managed files unless `--force` is used

### 5.4 Refresh a managed scaffold

Entry point:

- `afs add-agent --force`

Expected outcome:

- replace managed files with current template versions
- remove stale managed rules and skills no longer shipped by templates
- preserve project-owned files such as:
  - `README.md`
  - `specs/spec.yml`
  - `SNAPSHOT.md`
  - `SPEC.md`

### 5.5 Refresh only curated skills

Entry point:

- `afs add-agent --only-skills`
- `afs add-agent --only-skills --force`

Expected outcome:

- manage only `skills/*` and `skills/AVAILABLE_SKILLS.xml`
- do not touch `AGENTS.md` or `rules/*.yaml`

### 5.6 Create prompt helpers

Entry point:

- `afs add-agent-prompt [--force]`
- `afs add-ss-prompt`

Expected outcome:

- install the general agent prompt file into `prompts/agent-prompt.txt`
- print or copy the snapshot/spec helper prompt from installed templates

### 5.7 Remove local installation

Entry point:

- `afs uninstall`

Expected outcome:

- remove published commands in `~/bin`
- remove managed runtime assets under `~/.agent47`

## 6. Public Command Surface

Supported user-facing CLI commands:

- `afs help`
- `afs uninstall`
- `afs doctor [--check-update|--check-update-force]`
- `afs add-agent [--force] [--only-skills]`
- `afs add-agent-prompt [--force]`
- `afs add-ss-prompt`
- `./install.sh [--force] [--non-interactive]`

Repository validation and maintenance commands:

- `./scripts/test`
- `./scripts/lint-shell`
- `./scripts/smoke-install`

Commands intentionally not part of the supported public install surface:

- `afs install`
- `afs upgrade`
- `afs add-spec`
- `afs add-cli-prompt`
- `afs templates`
- `afs check-update`

## 7. Repository Structure Contract

### 7.1 Repo-owned source

- `bin/`
- `scripts/`
- `docs/`
- `tests/`
- root metadata such as `README.md`, `VERSION`, `CHANGELOG.md`, `SNAPSHOT.md`, `SPEC.md`

### 7.2 Template payload

- `templates/AGENTS.md`
- `templates/manifest.txt`
- `templates/rules/*.yaml`
- `templates/skills/*/SKILL.md`
- `templates/prompts/*.txt`
- `templates/specs/spec.yml`

### 7.3 Runtime libraries

- `scripts/lib/runtime-env.sh`
- `scripts/lib/install-runtime.sh`
- `scripts/lib/install-assets.sh`
- `scripts/lib/bootstrap.sh`
- `scripts/lib/managed-files.sh`
- `scripts/lib/doctor.sh`
- `scripts/lib/templates.sh`
- `scripts/lib/skill-utils.sh`
- `scripts/lib/update.sh`
- `scripts/lib/test-runtime.sh`

## 8. Architecture

### 8.1 Router

`bin/afs` is the public CLI entrypoint. The product remains `agent47`; `afs` is the public command, short for `Agent Forty-Seven`. Its responsibilities are:

- resolve the real entrypoint path
- initialize runtime globals
- load shared libraries
- route subcommands
- print help

It is intentionally thin and delegates most operational logic to `scripts/lib/`.

### 8.2 Runtime bootstrap

`scripts/lib/runtime-env.sh` is the shared environment initializer. It resolves:

- runtime root directories
- installed home path
- scripts and library paths
- version location
- update cache paths
- remote version URL

This allows both repo-local execution and installed execution to use the same library model.

### 8.3 Installation subsystem

Installation is handled by:

- `install.sh`
- `scripts/lib/install-runtime.sh`
- `scripts/lib/install-assets.sh`

Responsibilities:

- preflight required install assets
- publish templates
- publish helper scripts
- install launcher
- update `~/bin` symlink
- clean up legacy helper scripts, including older `a47` launchers in managed locations
- clear macOS quarantine attributes where possible
- uninstall managed runtime cleanly

### 8.4 Bootstrap subsystem

Project scaffolding is handled by:

- `scripts/add-agent`
- `scripts/lib/bootstrap.sh`
- `scripts/lib/managed-files.sh`
- `scripts/lib/skill-utils.sh`

Responsibilities:

- validate required template presence
- validate skills helper support
- stage files in a temporary transaction directory
- reconcile managed targets
- roll back on failure
- generate `skills/AVAILABLE_SKILLS.xml`

### 8.5 Diagnostics subsystem

Diagnostics are handled by:

- `scripts/lib/doctor.sh`
- `scripts/lib/templates.sh`
- `scripts/lib/update.sh`

Responsibilities:

- detect whether commands in `PATH` are managed copies
- verify template and prompt presence
- verify rule IDs and AGENTS structure
- perform cached update checks
- degrade safely to warnings when git/curl/network are unavailable

## 9. Manifest Contract

`templates/manifest.txt` is the declarative contract for scaffold ownership and required install assets.

Sections:

- `[rule_templates]`
- `[managed_targets]`
- `[preserved_targets]`
- `[required_template_files]`
- `[required_template_dirs]`

Behavioral meaning:

- `rule_templates` defines canonical rule files to install
- `managed_targets` defines files/directories that `agent47` owns in target repos
- `preserved_targets` defines files protected from replacement during normal refresh
- `required_template_files` and `required_template_dirs` drive install preflight validation

The runtime treats an empty required section as an invalid manifest contract.

## 10. Managed vs Preserved Ownership

### 10.1 Managed targets

By default, the scaffold manages:

- `AGENTS.md`
- `rules/*.yaml`
- `skills/*`
- `skills/AVAILABLE_SKILLS.xml`

### 10.2 Preserved targets

The refresh flow preserves:

- `README.md`
- `specs/spec.yml`
- `SNAPSHOT.md`
- `SPEC.md`

### 10.3 Force semantics

`afs add-agent --force` reconciles managed targets against the current template payload. This means:

- stale managed files may be removed
- local custom files under managed paths may be replaced or removed
- preserved targets remain untouched

## 11. Policy Model

The scaffold is policy-first.

Key contract principles:

- `AGENTS.md` is the single source of operational policy
- prompts and workflow artifacts should align with `AGENTS.md` and avoid unnecessary policy duplication
- security guidance is layered:
  - global
  - language
  - stack
- in template-source repositories such as `agent47` itself, reads that normally target `rules/` should instead use `templates/rules/`

The authority order defined by the scaffold is:

1. User instructions
2. Nearest `AGENTS.md`
3. Security rules
4. Stack rules
5. Specs and plans
6. Code and tests
7. Memories and hints

## 12. Rules and Skills Model

### 12.1 Rules

The template payload includes:

- stack rules:
  - frontend
  - backend
  - mobile
- security rules:
  - global
  - shell
  - JS/TS
  - Python
  - Java/Kotlin
  - Swift
  - C#

Rule design intent:

- security rules hold normative security requirements
- stack rules add workflow and architecture guidance
- stack rules reference security IDs instead of copying security guidance

### 12.2 Skills

Skills are discovered dynamically from installed templates:

- any `templates/skills/<name>/SKILL.md` is eligible
- `SKILL.md` must contain valid frontmatter
- name must be kebab-case
- description is required
- `skills/AVAILABLE_SKILLS.xml` is regenerated from valid skill templates

Invalid skill behavior:

- during normal bootstrap, invalid skill content is warned about and existing content is preserved where possible
- during force refresh, invalid staged skills abort the operation

## 13. Prompt Model

The project ships two prompt helpers:

- `templates/prompts/agent-prompt.txt`
- `templates/prompts/ss-prompt.txt`

Design intent:

- prompt helpers assist agent workflows
- they are not the primary source of policy
- they must stay aligned with `AGENTS.md`
- prompt generation is explicit, not part of every bootstrap flow
- `add-ss-prompt` is intended to help generate or update `SNAPSHOT.md` and `SPEC.md`

## 14. Data and File Contracts

### 14.1 `AGENTS.md`

- must remain compact enough for agent consumption
- must contain required sections validated by runtime/tests
- defines execution model, authority order, approval boundaries, security expectations, and output expectations

### 14.2 `skills/AVAILABLE_SKILLS.xml`

- generated artifact
- contains valid skills only
- includes skill name, description, and file location

### 14.3 `templates/specs/spec.yml`

- template artifact describing the expected shape of a project spec/plan/tasks/log file
- exists as part of the installed template payload
- target repos may use `specs/spec.yml` when work is non-trivial

### 14.4 Update cache

- stored under `~/.agent47/cache/update.cache`
- cache content is base64-encoded line fields
- cache invalidates on age or local-version mismatch

## 15. Operational Requirements

- Bash must be available.
- Standard Unix tooling is assumed, including utilities such as `cp`, `mv`, `find`, `grep`, `sed`, `awk`, `mktemp`, and `readlink`.
- `git` and `curl` are optional but improve update-check behavior.
- `pbcopy` is optional for `add-ss-prompt` clipboard support.
- `shellcheck` is optional for maintainer linting.
- Bats is provided via vendored source or a local installation.

## 16. Quality and Reliability Requirements

- Core scripts should run in strict shell mode where appropriate.
- Install and bootstrap flows must fail fast on missing required assets.
- Write operations should prefer staging plus atomic publish or swap patterns.
- Bootstrap and install flows should roll back partial state when a mid-flight failure occurs.
- Unknown public commands must return non-zero.
- Diagnostics should warn instead of crashing when optional tools are missing.

## 17. Security Expectations

- Never hardcode secrets or private credentials.
- Do not expose sensitive data in logs.
- Validate untrusted input at boundaries.
- Do not execute shell commands derived from untrusted input.
- Prefer portable shell constructs or guard platform-specific commands.
- Perform destructive filesystem operations only on explicitly scoped managed paths.
- Do not add vendor-specific agent config files without explicit approval.

## 18. Testing and Verification Model

Primary validation commands:

- `./scripts/test`
- `./scripts/lint-shell`
- `./scripts/smoke-install`

Current automated coverage includes:

- CLI routing and public command surface
- bootstrap behavior
- force refresh semantics
- install and uninstall behavior
- update-check caching and fallbacks
- prompt helper behavior
- policy/template integrity
- manifest validation
- rollback behavior under failure injection

## 19. Known Constraints

- Unix/macOS/Linux oriented; Windows support is not a goal.
- The product relies on shell semantics and local filesystem behavior.
- Update checks depend on network access and tooling availability.
- `--force` can remove custom files under managed paths by design.
- Prompt and spec workflows are helper-oriented, not a full project-planning platform.

## 20. Known Product Gaps

- `templates/specs/spec.yml` ships with the installed template payload, but `afs add-agent` does not scaffold `specs/spec.yml` into the target repository by default.

## 21. Success Criteria

The project is considered successful when:

- local installation is reproducible and recoverable
- `afs doctor` can diagnose common setup issues
- `afs add-agent` establishes a consistent agent contract in a target repo
- `afs add-agent --force` safely reconciles managed files
- scaffold ownership boundaries are explicit and test-backed
- templates and runtime stay aligned through manifest validation and policy tests
- the repo remains small, auditable, and dependency-light

## 22. Maintenance Expectations

- Keep root docs concise and push detailed operational behavior into focused docs.
- Keep runtime logic modular under `scripts/lib/`.
- Update tests when command semantics, manifest sections, or template contracts change.
- Prefer documentation and tests that reflect real behavior over aspirational behavior.
- Preserve the distinction between:
  - repo implementation
  - installed runtime
  - target-project scaffold

## 23. Current State

At the time of this spec:

- the repository is organized and test-backed
- the public CLI surface is intentionally narrow
- installation and bootstrap flows are hardened compared to earlier revisions
- the public command is `afs`, while the product and CLI identity remain `agent47`
- the design center is explicit, inspectable repository policy rather than automation magic
