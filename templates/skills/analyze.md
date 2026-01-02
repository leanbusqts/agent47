# Skill: analyze

## Objective
Read-only analysis of existing code, specs, or architecture to surface findings and recommendations.

## When to Use
- Clarify current behavior or constraints before changes.
- Assess risks or unknowns prior to implementation.

## Inputs
- AGENTS.md and applicable `rules-*.yaml`
- Relevant specs (if present)
- Relevant source files and tests
- User instructions

## Process
1. Read authoritative sources (AGENTS, rules, specs, code/tests).
2. Identify facts, gaps, and risks.
3. Summarize findings and recommended next steps (no code).

## Outputs
- Findings and observations
- Risks and assumptions
- Suggested next steps (no code or fixes)

## Boundaries
- Allowed: Read-only analysis; cross-module reasoning.
- Forbidden: Code generation, code modification, refactors.

## Expected Quality
- Evidence of reading rules/specs/code.
- Clear risks and unknowns called out explicitly.
