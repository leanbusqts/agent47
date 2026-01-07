---
name: implement
description: Deliver a scoped change that meets the stated requirements with minimal surface area.
---

# Implement

## Objective
- Apply the approved change precisely.
- Minimize blast radius and keep behavior aligned to requirements.

## When to use
- Requirements or spec are clear and approved.
- A behavior change is expected.

## Inputs
- User instructions, spec, and acceptance criteria.
- AGENTS.md, rules, and relevant code/tests.
- Any clarifications already answered.

## Process
1) Confirm scope and acceptance criteria; note exclusions.
2) Plan minimal changes; avoid refactors outside scope.
3) Implement with small, focused edits.
4) Update/add tests if behavior changes.
5) Self-review for correctness and unintended impact.

## Outputs
- Files changed and intent.
- Notes on tests added/updated.
- Any residual risks or follow-ups.

## Edge cases
- If scope is unclear, pause and ask before coding.
- Avoid drive-by refactors; defer to refactor skill if needed.
