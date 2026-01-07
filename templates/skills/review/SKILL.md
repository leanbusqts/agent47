---
name: review
description: Inspect changes for correctness, risks, and regressions before accepting them.
---

# Review

## Objective
- Find defects, regressions, and unmet requirements.
- Provide actionable feedback.

## When to use
- Reviewing proposed changes (diffs/PRs).
- Before merging or approving work.

## Inputs
- User request and acceptance criteria.
- AGENTS.md, rules, specs (if any), and the diff or changed files.
- Related tests and CI signals if available.

## Process
1) Restate the intended change and scope.
2) Check for requirements coverage and correctness.
3) Look for regressions, edge cases, and risk areas.
4) Evaluate tests: coverage and missing cases.
5) Summarize issues with severity and recommendations.

## Outputs
- Issues found (with severity) and recommendations.
- Coverage gaps or missing tests.
- Approval status or blockers.

## Edge cases
- If context is missing (spec, requirements), call it out and limit findings.
- Keep feedback concise and prioritized.
