---
name: analyze
description: Understand the current state, flows, and issues before making changes.
---

# Analyze

## Objective
- Map current behavior and constraints.
- Identify problems, risks, and missing context.

## When to use
- Before proposing fixes or features.
- When debugging or evaluating existing code/flows.

## Inputs
- User request and acceptance criteria.
- AGENTS.md, rules, specs (if any), and relevant code/tests/logs.
- Observed symptoms or failures.

## Process
1) Summarize the goal and observed behavior.
2) Locate relevant code/tests and note gaps or contradictions.
3) Identify probable causes or risk areas.
4) List what evidence or data is missing.
5) Propose next actions (investigation or plan) without changing code.

## Outputs
- Findings (what is known).
- Risks/unknowns (what is missing).
- Proposed next steps or plan.

## Edge cases
- If information is insufficient, stop and ask for specific inputs.
- Do not change code in this skill; stay diagnostic.
