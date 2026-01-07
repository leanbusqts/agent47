---
name: refactor
description: Improve structure and clarity without changing externally visible behavior.
---

# Refactor

## Objective
- Simplify and clarify code while keeping behavior identical.
- Reduce technical debt in-scope without altering APIs or outputs.

## When to use
- Behavior is correct but code quality is poor.
- There is approval to clean up without feature changes.

## Inputs
- User intent and acceptance for refactor scope.
- AGENTS.md, rules, and relevant code/tests.
- Existing tests to confirm behavior stability.

## Process
1) Identify pain points (duplication, naming, structure).
2) Plan minimal, safe refactors; avoid tangents.
3) Apply changes incrementally, keeping tests passing.
4) Verify behavior parity via tests or reasoning.
5) Note any residual debt deferred.

## Outputs
- Files changed and rationale.
- Confirmation of unchanged behavior.
- Tests run (or why not) to prove parity.

## Edge cases
- If behavior must change, switch to implement/optimize.
- Avoid large extractions without clear gain and approval.
