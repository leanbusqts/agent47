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
5. Specs such as `specs/spec.yml`
6. Code and tests
7. Memories and other hints

If two items at the same level conflict, stop and ask.

## Required Inputs

Read what applies before acting:
- `AGENTS.md`
- Relevant security and stack rules under `rules/`
- `specs/spec.yml` when the task is non-trivial or spec-driven
- Relevant code and tests
- `skills/AVAILABLE_SKILLS.xml` and the selected `skills/*/SKILL.md` when skills are in use

## Executable Commands

Project-maintained commands:
- Test: `make test` or `./scripts/test`
- Install CLI locally: `./install.sh`
- Health check: `a47 doctor`
- Bootstrap project scaffolding: `a47 add-agent`
- Create spec scaffold: `a47 add-spec`
- Refresh skills metadata: `a47 reload-skills`

There is no build, lint, or deploy command managed by this repository.

## Context Efficiency

- Do not reread files that have not changed unless needed to resolve uncertainty.
- Do not rerun the same command without a reason.
- Prefer targeted reads/searches over loading large files repeatedly.
- Keep quoted source text short; summarize instead of copying.

## Execution Model

- Use `specs/spec.yml` as the optional plan/tasks/log location for non-trivial work.
- Trivial tasks do not require a spec or written plan.
- Prefer tests in this order: happy path, error handling, edge cases.
- Finish scoped work end-to-end: implementation, tests, and verification.

## Filesystem And Approval Boundaries

### Always

Actions that are safe to do without asking:
- Read repository files needed for the task
- Edit existing files already in scope
- Add or update tests that support the requested change
- Create small supporting files clearly required by the task inside the repo
- Run local project commands such as `make test` or `./scripts/test`

Examples:
- Updating `templates/AGENTS.md`
- Adding a missing unit test under `tests/unit/`
- Regenerating `skills/AVAILABLE_SKILLS.xml`

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
- Write vendor-specific agent config files such as `claude.md`, `.cursorrules`, or Codex-only config files

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
