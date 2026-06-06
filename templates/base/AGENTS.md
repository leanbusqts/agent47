# AGENTS.md

<!--
agents-md: { version: 2, last_updated: "2026-06-06", schema_version: 1, owner: platform-policy, review_cadence: quarterly, applies_to: "/Users/leanbusqts/Develops/agent47/", mirror: "/Users/leanbusqts/Develops/agent47/templates/base/AGENTS.md", max_lines: 200, target_lines: 162 }
-->

## Purpose

`AGENTS.md` is the agent-operating policy for this repo. `README.md`, `RUNBOOK.md`, `SPEC.md`, `specs/spec.yml`, `skills/`, `rules/*.yaml` reference this file instead of restating policy. Conflicts resolve per Authority Order below. [AG-001]

## Scope

This contract applies to all work in the repo. Project-type bundles defined in `SPEC.md` §5.3: `frontend, backend, mobile, cli, scripts, infra, monorepo-tooling, desktop, plugin`. Repos with no detected type receive the base bundle (`AGENTS.md` + global/shell security rules + universal skills). [AG-003]

## Glossary

| Term | Definition | Threshold | ID |
|---|---|---|---|
| **Trivial task** | Change ≤ 2 files, no new tests required, no deps/network/migration impact. Non-trivial = the complement (triggers spec/plan/review). | ≤ 2 files | [AG-005] |
| **In scope** | (a) named by the user, (b) referenced from a file in scope, or (c) the natural test/doc of a file in scope | — | [AG-007] |
| **Approval** | Explicit confirmation in the current conversation OR a written note in `specs/spec.yml` / a referenced ticket | current conversation | [AG-008] |
| **Broad set** | Operation over ≥ 5 files or a `**/*` glob crossing directories | ≥ 5 files | [AG-009] |
| **Long/expensive** | Wall time ≥ 60s, API calls ≥ 50, or use of a billable resource | ≥ 60s | [AG-010] |
| **Documented command** | Appears in `RUNBOOK.md`, `README.md`, `SPEC.md`, `Makefile`, `package.json`, `pyproject.toml`, or `docs/*.md` | — | [AG-011] |
| **Vendor agent config** | `claude.md`, `.cursorrules`, `.codex/config.toml`, `.windsurf*`, undocumented `mcp.json` | — | [AG-017] |

## Authority Order

When sources conflict, resolve in this order (highest wins). If two items at the same level conflict, stop and ask. [AG-040]

1. User instructions in the current conversation. [AG-041]
2. The closest `AGENTS.md` in the directory tree. In this repo: the root one. [AG-042]
3. Security rules by ID: `security-global.yaml` -> `security-<lang>.yaml` -> `security-shell.yaml` when applicable; see `rules/SEVERITY.md` for severity semantics. [AG-043]
4. Stack rules: `rules-<stack>.yaml` for the file under edit. [AG-044]
5. Skill instructions loaded via `skills/AVAILABLE_SKILLS.xml`. Skills extend, never override sources above. [AG-045]
6. `specs/spec.yml` for the current task. [AG-046]
7. Code and tests as evidence of intended behavior. [AG-047]
8. Memories, hints, prior-session notes. Suggestions, not contracts. [AG-048]

## Required Inputs

Load only what the task requires. [AG-018]

| Condition | Files | ID |
|---|---|---|
| Always | `AGENTS.md`, `rules/security-global.yaml`, `rules/security-<lang>.yaml` for files in scope; add `security-shell.yaml` + `rules-scripts.yaml` when shell-heavy | [AG-019] |
| File in a known stack | `rules/rules-<stack>.yaml`; in this template-source repo also `templates/base/rules/*.yaml`, `templates/bundles/<bundle>/rules/*.yaml`, `templates/manifest.txt` | [AG-020] |
| Non-trivial task | `specs/spec.yml`, `SPEC.md` (if product contract changes), `SNAPSHOT.md`, `CHANGELOG.md` (public-surface change) | [AG-023] |
| Skills in use / CLI work | `skills/AVAILABLE_SKILLS.xml` + the selected `skills/<name>/SKILL.md`; for CLI/maintenance work `RUNBOOK.md` and `README.md` §"Public commands" | [AG-024] |
| Never required | files outside scope, build artifacts, `.git/`, `node_modules/`, vendored deps | [AG-026] |

## Executable Commands

`AGENTS.md` does not enumerate universal commands. In this repo: `RUNBOOK.md`, `README.md` §"Public commands", `Makefile`, `SPEC.md` §6. Use only a Documented command (see Glossary). [AG-027]

Forbidden to assume an `afs` subcommand not listed there. `SPEC.md` §6 lists "intentionally not public" — authoritative. [AG-029]

## Context Efficiency

- Do not reread a file already loaded unless its contents may have changed. [AG-030]
- Do not rerun a command unless its output may have changed. [AG-031]
- Prefer `grep` or `Read offset+limit` over loading > 500-line files in full. [AG-032]
- Quote ≤ 20 contiguous lines; for longer regions -> summarize + cite line ranges. [AG-033]
- Run independent reads/searches in parallel when the runtime supports it. [AG-034]

## Execution

- Trivial: no spec/plan. Implement, test, report. [AG-200]
- Non-trivial: use `specs/spec.yml` for spec/plan/tasks/log; create through conversation, no CLI scaffold (none exists). [AG-201]
- Drafting spec/plan: clarify questions first, user review before implementation. [AG-202]
- Update `SNAPSHOT.md` and `SPEC.md` before commit if the scoped work materially changes them. [AG-203]
- Tests in order: happy, failure, edge (see `rules/shared-testing.yaml` SHT-001). [AG-204]
- Implement-then-review by default; plan-then-implement-then-review for work > 8h or multi-session. [AG-206]
- Single-agent is the default for trivial tasks. Use multi-agent only when the task is complex, multi-file, ambiguous, or clearly multi-domain. [AG-210]
- Multi-agent for tasks that are complex, multi-file, ambiguous, or multi-domain (backend+frontend, security+docs, Android+iOS). Roles: implementer, reviewer, tester, security reviewer, doc editor. One agent owns the final synthesis. [AG-212]
- Review is an independent quality check, not a restatement. Without a multi-agent runtime, emulate the implement-phase / review-phase split. [AG-215]
- Do not use multi-agent when the overhead does not improve outcome quality. [AG-217]

## Filesystem And Approval Boundaries

| Bucket | Actions |
|---|---|
| Always | Read scoped files [AG-051]; edit existing in scope [AG-052]; add/update tests [AG-053]; create small support files (≤ 3 files, ≤ 200 lines combined) [AG-054]; run Documented commands [AG-055] |
| Ask | Deps add/remove/upgrade — see Dependency Policy [AG-061]; Broad set ops [AG-062]; network / external downloads [AG-063]; Long/expensive jobs [AG-064]; template restore over user edits [AG-065]; run scripts from `skills/scripts/` [AG-066]; new top-level dirs at repo root [AG-067]; modify CI/CD workflows (`.github/workflows/`, `.gitlab-ci.yml`) [AG-068] |
| Never | Hardcode secrets/tokens/passwords/personal data — SEC-global-001 [AG-071]; exfiltrate source/secrets/user data — SEC-global-005 [AG-072]; bypass approval with hidden side effects [AG-073]; delete unrelated files or revert user changes without approval [AG-074]; rewrite pushed git history (`amend` pushed, `push --force` shared) [AG-075]; write Vendor agent config without authorization [AG-076]; skip pre-commit hooks/signing (`--no-verify`, `--no-gpg-sign`) without authorization [AG-077]; destructive ops (`rm -rf`, `git reset --hard`, `git clean -fd`) outside an authorized scope [AG-078] |

## Security Expectations

Authoritative: `rules/security-*.yaml`. Load order: `security-global.yaml` -> `security-<lang>.yaml` -> `security-shell.yaml` when shell-related. [AG-080] Principles (canonical in the YAMLs, by ID): secrets out of source/logs/prompts/screenshots/tests (SEC-global-001, 002); untrusted input validated at the boundary (SEC-global-003); no execution without an allow-list (SEC-global-004); no exfiltration without Approval (SEC-global-005). Deduplication: security guidance lives ONLY in `security-*.yaml`; cross-ref by ID, do not restate. [AG-082]

## Dependency Policy

Dependency changes are governed by `X-deps-001` in `rules/rules-cross.yaml` plus `rules/APPROVALS.md#dependencies`. A change is add, remove, upgrade, or pinning shift. [AG-090]

Required justification (in the PR or `specs/spec.yml`): benefit vs cost; acceptable license (default-deny AGPL/SSPL/BUSL); pinning + integrity hash committed (lockfile); wrapper when used in ≥ 3 places. Approval covers the named version; later upgrades require new approval. [AG-091] Optional guidance: prefer existing tooling first, a single package manager, and a small local utility over a new dependency when the utility is genuinely small and stable. [AG-093]

## Stack Notes

Authoritative: `rules/rules-<stack>.yaml`. Hooks only — full rule text and rule IDs (FE-*, BE-*, MO-*, …) live in the YAMLs. [AG-100]

| Stack | Canonical YAML |
|---|---|
| Frontend | `rules-frontend.yaml` |
| Backend | `rules-backend.yaml` |
| Mobile | `rules-mobile.yaml` |
| CLI | `rules-cli.yaml` |
| Scripts | `rules-scripts.yaml` |
| Infra | `rules-infra.yaml` |
| Desktop | `rules-desktop.yaml` |
| Plugin | `rules-plugin.yaml` |
| Monorepo | `rules-monorepo-tooling.yaml` |

## Skills

- Discovery: `skills/AVAILABLE_SKILLS.xml` (canonical), `.json`, `SUMMARY.md`. Load the index first; load a specific `SKILL.md` only when the skill is selected. [AG-120]
- Trigger: explicit invocation (`/skill-name`) or a match on `description`. Optional guidance: skip the skill when the task is a direct lookup or the skill would add little value relative to the task size. [AG-121]
- Loading: only the selected `SKILL.md` plus refs it explicitly cites. Do not eagerly load. [AG-122]
- Authority: skills extend, never override (see AG-045). [AG-123]
- Conflict: the skill named by the user wins; otherwise the one with the more specific `description`, and report the conflict. [AG-124]
- Skip when: user opt-out, single-step lookup, prerequisites not satisfied. [AG-125]

## Output Expectations

Structure: (1) what changed/found; (2) verification performed (commands + outcomes, or "not run because X"); (3) residual risks/assumptions. [AG-130] Format: markdown for > 3 lines; code blocks with a language tag for ≥ 2-line snippets; absolute paths for files outside the file under edit. Length: trivial 1–3 sentences; non-trivial 1 paragraph + a flat list; multi-agent synthesis 1 paragraph + a flat list; avoid > 1 page unless requested. Tone: factual, active voice; match the user's language (es/en/pt); switch only on explicit instruction. [AG-131]

## Verification And Rollback

- Detect early: after each destructive step, verify state before the next one. [AG-180]
- Re-read the user request; confirm every point is addressed. [AG-141]
- Every edited file is syntactically valid (compiles/parses). [AG-142]
- Code changes: run the project's test command per Executable Commands; backend/library: run a build (`make go-build`, `tsc --noEmit`). [AG-143]
- Rule/policy edits: `diff -q AGENTS.md templates/base/AGENTS.md` empty; `make agents-check` passes. [AG-145]
- Skipping a check: name the check AND the reason in the response (per AG-130). [AG-146]
- A test failure not expected by the change is a task failure, not a test failure. Do not "fix the test" without root-cause analysis. [AG-181]
- Roll back: uncommitted -> `git restore <file>` per file (never `git restore .`); committed-unpushed -> `git revert <sha>`; pushed feature -> follow-up revert commit; pushed `main` -> ALWAYS ask the user. [AG-182]
- Do not bypass the failed check, delete the failing test without authorization, or catch broadly to silence. [AG-183]
- Report on rollback: what failed, what was rolled back, what remains known-good. [AG-184]

## Git And Commits

Free: `status`, `diff`, `log`, `show`, `branch --list`, `stash list`, creating a local branch, staging specific files (NOT `git add -A` or `.`). [AG-150]

Requires user request: `commit`, `push`, PR create/edit/merge, switching branches with uncommitted changes, `stash`/`stash pop`. [AG-151]

Forbidden without authorization (see AG-075, AG-077): `amend` pushed, `rebase -i`, `reset --hard`, `checkout --`, `clean -fd`, `push --force[-with-lease]` shared, `--no-verify`, `--no-gpg-sign`. [AG-152]

Commit message: subject ≤ 70 chars present-tense imperative; body wrapped at 72, explains the why; reference rule IDs when applicable; trailer block ≤ 5 lines. Branch names: kebab-case `<verb>-<noun>`; stacking `<feature>/01-foundation`. [AG-153]

## Communication Conventions

- Language: mirror the user (es/en/pt). Switch only when the user switches. [AG-170]
- Tone: professional, concise, neutral. No filler. No emojis unless the user uses them. [AG-171]
- Status: pre > 5 steps state the plan in one sentence; mid-operation update only on blocker / direction change / finding; on completion, emit the AG-130 shape. [AG-172]
- Questions: batch related ones (A/B/C); do not ask about Always actions. [AG-173]
- Refusals: name the action + the rule ID; offer the closest acceptable alternative. [AG-174]

## Maintenance Of This File

- Owner: `platform-policy`. Reviewers: 1 platform-security + 1 platform-tooling. PR-only. [AG-190]
- Update when: a new bundle -> Scope + Stack Notes; a new `afs` command -> Executable Commands; rule add/rename/remove -> cross-refs here; `manifest.txt` change -> Glossary if relevant. [AG-191]
- Versioning: bump `version` for every AG-NNN add/remove/modify; bump `schema_version` only when structure changes; `last_updated` ISO-8601 is mandatory. [AG-192]
- Mirror invariant: `AGENTS.md` == `templates/base/AGENTS.md`. Validated by `make agents-check`, wired into `make test` + CI. [AG-193]
