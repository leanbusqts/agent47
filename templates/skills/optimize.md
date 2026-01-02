# Skill: optimize

## Objective
Improve performance based on evidence while preserving behavior.

## When to Use
- Profiling or metrics indicate a bottleneck.
- Performance goals are explicit and approved.

## Inputs
- AGENTS.md and applicable `rules-*.yaml`
- Profiling/metrics evidence
- Relevant source files and tests
- User instructions and performance targets

## Process
1. Validate performance goal and scope; confirm evidence.
2. Apply targeted optimizations with minimal surface area.
3. Re-measure to confirm improvement and unchanged behavior.

## Outputs
- Optimized files with intent noted.
- Evidence of improvement (metrics/benchmarks).
- Trade-offs and risks.

## Boundaries
- Allowed: Targeted, evidence-based optimizations; keeping behavior intact.
- Forbidden: Speculative optimizations; changes without measurement; architectural rewrites.

## Expected Quality
- Demonstrated improvement; clear trade-offs; behavior preserved.
