# AGENTS.md

## Purpose

This file is the single source of operating policy for agents in this repository.
Prompts, README notes, specs, skills, and stack rules must reference this file instead of copying policy text.

## Scope

This contract applies to frontend, backend, mobile, scripts, tests, docs, and repo maintenance work.

## Authority Order

Use this order whenever instructions conflict:
1. User instructions
2. Nearest `AGENTS.md`
3. Security rules: global -> language -> stack
4. Stack rules
5. Specs and plans such as `specs/spec.yml`
6. Code and tests
7. Memories and other hints

If two items at the same level conflict, stop and ask.

## Required Inputs

Read what applies before acting:
- `AGENTS.md`
- Relevant security and stack rules under `rules/`
  - In template-source repositories such as `agent47` itself, read the equivalent files under `templates/base/rules/` and the relevant project bundle rules under `templates/bundles/*/rules/`
  - For shell-heavy repositories, include the shell security rules when present
- `specs/spec.yml` when the task is non-trivial, plan-driven, or spec-driven
- In this repository, root `SPEC.md` describes the current-state product spec for `agent47` itself; do not treat it as the default place to draft a new feature spec or implementation plan
- Relevant code and tests
- `skills/AVAILABLE_SKILLS.xml` and the selected `skills/*/SKILL.md` when skills are in use

## Executable Commands

This file does not define universal commands for every project that uses `AGENTS.md`.

Use only commands that are documented or clearly discoverable in the current project, such as:
- test commands exposed by the repo
- local bootstrap or setup scripts owned by the repo
- documented health checks, generators, or maintenance tasks

If a command is not defined by the current project, do not assume an `agent47` CLI command exists.

## Context Efficiency

- Do not reread files that have not changed unless needed to resolve uncertainty.
- Do not rerun the same command without a reason.
- Prefer targeted reads/searches over loading large files repeatedly.
- Keep quoted source text short; summarize instead of copying.

## Execution Model

- Use `specs/spec.yml` as the optional spec/plan/tasks/log location for non-trivial work.
- If the user asks for a spec or plan and `specs/spec.yml` does not exist, create or update it through normal agent interaction instead of relying on a dedicated CLI scaffold.
- When drafting or updating a spec or plan through conversation, ask follow-up questions when needed, then suggest that the user review the resulting spec/plan before implementation starts.
- If `SNAPSHOT.md` or `SPEC.md` already exist and the scoped work materially changes them, consider updating them before commit to keep project documentation aligned with the current state.
- Trivial tasks do not require a spec or written plan.
- Prefer an implement-then-review flow for non-trivial changes.
- Prefer tests in this order: happy path, error handling, edge cases.
- Finish scoped work end-to-end: implementation, tests, and verification.

## Execution Strategy

- Prefer a single-agent flow for trivial, low-risk, or narrowly scoped tasks.
- For non-trivial work, prefer a split between implementation and review when the runtime or tool supports multi-agent or sub-agent execution.
- When a spec or plan is important or high-impact, prefer an independent review of the spec/plan itself by another agent or sub-agent when the runtime supports it.
- Escalate to a multi-agent workflow when the task is complex, high-impact, multi-file, ambiguous, or spans multiple domains such as backend, frontend, security, documentation, tests, Android, or iOS.
- Use specialized roles when helpful, such as implementer, reviewer, tester, security reviewer, or documentation editor.
- Keep one agent responsible for final synthesis so the result is coherent, conflicts are resolved, and the user receives one clear outcome.
- Treat review as an independent quality check, not as a restatement of the implementation step.
- If the runtime does not support multi-agent execution, emulate the same separation of responsibilities within a single session.
- Balance depth against cost and latency; do not use multi-agent orchestration when it adds overhead without improving outcome quality.
- Do not delegate secrets or sensitive data more broadly than necessary for the task.

## Filesystem And Approval Boundaries

### Always

Actions that are safe to do without asking:
- Read repository files needed for the task
- Edit existing files already in scope
- Add or update tests that support the requested change
- Create small supporting files clearly required by the task inside the repo
- Run documented local project commands for the current repo, such as its test or verification scripts

Examples:
- Updating an existing policy or rule file already in scope
- Adding a missing unit test under the project's test directory
- Running the project's documented test command

### Ask

Actions that require approval first:
- Adding, removing, or upgrading dependencies
- Creating, deleting, moving, or overwriting broad sets of files
- Network access, external downloads, or calling remote services
- Running long or potentially expensive jobs
- Copying or restoring templates in a way that replaces user-edited files
- Executing scripts from `skills/scripts/` or similar helper locations outside normal repo commands

Examples:
- Adding a new npm, pip, Gradle, CocoaPods, or Maven dependency
- Restoring templates from backup over an existing project copy
- Downloading a new test tool

### Never

Actions that are forbidden:
- Hardcode secrets, tokens, passwords, or personal data
- Exfiltrate source, secrets, or user data to external systems
- Bypass approval requirements with hidden side effects
- Delete unrelated files or revert user changes without explicit approval
- Write vendor-specific agent config files such as `claude.md`, `.cursorrules`, `/.codex/config.toml`, or other vendor-only agent config files for tools like Codex, Claude Code, Cursor, or Windsurf without explicit prior user authorization

Examples:
- Committing `.env` secrets
- Running destructive cleanup outside the requested scope

## Security Expectations

- Always load the applicable security rules first: `security-global.yaml`, then the applicable language file, then the stack file.
- Keep secrets out of source, logs, prompts, and tests.
- Validate untrusted input at the appropriate boundary.
- Avoid unsafe code execution, unsafe deserialization, and user-controlled outbound destinations.
- Keep security guidance deduplicated in security rule files; stack rules may add context or reference IDs, not restate the same rule text.

## Dependency Policy

New dependencies or dependency changes require approval and must be justified by:
- clear benefit vs maintenance cost
- acceptable license
- stable version pinning
- a small interface or wrapper when appropriate

Prefer existing project tooling and libraries.

## Stack Notes

Frontend:
- Keep business logic off the client when it belongs on the server.
- Preserve validation, CSP-conscious behavior, SSR/BFF boundaries, and input handling.

Backend:
- Keep transport, service, and data responsibilities separate.
- Use explicit error contracts and safe external call patterns.

Mobile:
- Keep UI/main thread work light.
- Use safe async patterns; avoid force unwraps and unsafe state sharing.
- Unless the user requests another architecture, prefer a layered mobile architecture with explicit UI, domain, and data layers.
- Use unidirectional data flow, screen-level state holders, repositories as the data boundary, and dependency injection; map these patterns to platform-native equivalents on Android and iOS.

## Skills

- Use a skill when the prompt metadata or user request indicates one.
- Load only the selected `SKILL.md` and any directly needed references.
- Skills extend workflow guidance; they do not override this file.

## Output Expectations

Responses should include:
- what changed or what was found
- tests/verification performed, or why not
- residual risks or assumptions

Keep reports concise and factual.
