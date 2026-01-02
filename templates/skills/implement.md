# Skill: implement

## Objective
Implement a new feature or change based on explicit requirements or an approved spec.

## When to Use
- A spec or clear requirements are provided.
- Behavior change is expected and approved.

## Inputs
- AGENTS.md and applicable `rules-*.yaml`
- Relevant spec(s)
- User instructions and acceptance criteria
- Relevant source files and tests

## Process
1. Validate scope against AGENTS, rules, and spec.
2. Plan localized changes; avoid speculative abstractions.
3. Implement with minimal invasiveness.
4. Update/add tests when behavior changes.

## Outputs
- Files created/modified with intent noted.
- Brief implementation notes.
- Tests added/updated (if behavior changed).

## Boundaries
- Allowed: Localized changes; new files; adhering to existing architecture.
- Forbidden: Large refactors; public API changes unless requested; speculative abstractions.

## Expected Quality
- Behavior matches requirements/spec.
- Minimal surface area change; no unrelated edits.
