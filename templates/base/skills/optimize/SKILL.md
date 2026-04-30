---
name: optimize
description: Improve performance or resource usage with evidence and clear trade-offs.
compatibility: Designed for skills-compatible coding agents.
metadata:
  category: performance
  tags: [performance, efficiency, cost]
  applies_to: [frontend, backend, cli, scripts, mobile]
  priority: suggested
  agents: [universal, codex]
  repo_shapes: [app, library, cli, scripts, monorepo]
---

# Optimize

## Objective
- Reduce latency, resource consumption, or cost.
- Provide evidence and clarify trade-offs.

## When to use
- Performance/cost is a requirement or pain point.
- There is baseline behavior to compare against.

## Inputs
- User goal (what to optimize and target).
- AGENTS.md, rules, and relevant code/tests.
- Baseline metrics or observations; if missing, note it.

## Process
1) Restate the optimization goal and constraints.
2) Identify likely hotspots and quick wins.
3) Propose minimal changes; avoid behavior drift.
4) Measure or reason about impact; note assumptions if unmeasured.
5) Report trade-offs and follow-ups.

## Outputs
- Changes made and why.
- Evidence (metrics or reasoned estimates) and assumptions.
- Trade-offs and next steps.

## Edge cases
- If no baseline exists, state that and avoid speculative claims.
- Do not sacrifice correctness for micro-optimizations without approval.
