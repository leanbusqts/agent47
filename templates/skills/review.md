# Skill: review

## Objective
Read-only review to identify issues, risks, regressions, and missing tests; provide recommendations.

## When to Use
- Code or design changes need evaluation before merge.
- Risk assessment is required without modifying code.

## Inputs
- AGENTS.md and applicable `rules-*.yaml`
- Relevant specs or requirements
- Code/design under review and related tests
- User instructions

## Process
1. Check conformance to AGENTS, rules, and spec/requirements.
2. Identify bugs, risks, regressions, or gaps in tests.
3. Classify severity and recommend actions (no code changes).

## Outputs
- Issues with severity and context/location.
- Risks and assumptions.
- Recommendations (no code patches).

## Boundaries
- Allowed: Read-only review; identifying gaps; recommending tests/fixes.
- Forbidden: Code modification; refactoring; approving unresolved critical risks.

## Expected Quality
- Findings prioritized by severity.
- Clear links between issues and rules/specs.
