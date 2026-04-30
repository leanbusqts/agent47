---
name: plan
description: Create a concise plan with risks and checkpoints before executing any changes.
compatibility: Designed for skills-compatible coding agents.
metadata:
  category: planning
  tags: [planning, checkpoints, risk]
  applies_to: [frontend, backend, cli, scripts, mobile]
  priority: core
  agents: [universal, codex]
  repo_shapes: [app, library, cli, scripts, monorepo]
---

# Plan

## Objective
- Frame the goal, constraints, and success criteria.
- Identify risks, unknowns, and checkpoints before acting.

## When to use
- Before implementation or risky refactors.
- When scope is ambiguous or there are open questions.

## Inputs
- User request and acceptance criteria.
- AGENTS.md and applicable rules/specs.
- Known constraints (time, tooling, environments).

## Process
1) Restate the goal and constraints.
2) Identify unknowns; call out what must be clarified.
3) Propose 3-7 sequenced steps with clear outcomes.
4) Mark risks and mitigation/rollback ideas.
5) Define checkpoints to pause/confirm before proceeding.

## Outputs
- Short plan (steps with expected outcomes).
- Risks and mitigations.
- Checkpoints or decision gates.

## Edge cases
- If critical inputs are missing, stop and ask.
- Keep the plan lean; avoid over-specifying implementation details.
