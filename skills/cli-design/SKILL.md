---
name: cli-design
description: Design or refine command surfaces, output contracts, and preview flows for CLI tools.
compatibility: Designed for skills-compatible coding agents.
metadata:
  category: cli
  tags: [cli, ux, output-contracts]
  applies_to: [cli, scripts]
  priority: suggested
  agents: [universal, codex]
  repo_shapes: [cli, scripts, monorepo]
  stack_signals: [cobra, urfave-cli, shell]
---

# CLI Design

## Objective
- Keep command behavior easy to explain, automate, and verify.
- Treat help text, exit codes, and output shape as product contracts.

## When to use
- Adding or changing commands, flags, preview flows, or mutation UX.
- Reviewing whether CLI behavior is stable in both TTY and non-TTY environments.

## Process
1) Confirm the supported command and flag surface.
2) Define the success output, warning output, and failure behavior.
3) Check non-interactive and automation compatibility.
4) Ensure tests cover the advertised contract.
