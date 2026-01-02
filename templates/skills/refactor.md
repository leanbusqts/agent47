# Skill: refactor

## Objective
Improve code structure, clarity, or safety without changing behavior, within allowed refactor triggers.

## When to Use
- Refactor triggers are met (duplication, clarity, correctness, testability).
- Behavior must remain unchanged.

## Inputs
- AGENTS.md and applicable `rules-*.yaml`
- Relevant specs (if any)
- Target source files and tests
- User instructions

## Process
1. Confirm refactor trigger and scope; ensure no behavior change.
2. Apply minimal, incremental changes to improve structure/readability/safety.
3. Verify behavior is preserved (reasoning or tests).

## Outputs
- Modified files with intent noted.
- Refactor justification.
- Confirmation that behavior is unchanged.

## Boundaries
- Allowed: Scoped refactors tied to triggers; removing duplication in-scope; simplifying logic.
- Forbidden: Behavior changes; architectural changes; unrelated modifications.

## Expected Quality
- Smaller surface area; clearer code; no behavioral drift.
