# AGENTS.md

## Purpose

This file defines the **global, invariant contract** for all AI agents operating in this repository and its related workspaces.

It establishes:
- Non-negotiable rules that always apply
- How agents must reason, decide, and act
- How stack-specific rules are discovered and applied
- How memories, rules, and constraints are organized

This file is **authoritative**. Stack-specific rules extend it but never override it.

---

## Scope

This AGENTS.md applies to:
- Mobile (Android / iOS)
- Backend (Kotlin / JVM services)
- Frontend (React / Node / Nordic)
- Testing, refactors, optimizations, and migrations

It is intentionally **stack-agnostic** and **stable over time**.

---

## Agent Operating Principles

### 1. Production-Grade Assumption

All code, tests, and documentation must be treated as **production-grade** by default.

- No placeholders
- No TODOs unless explicitly requested
- No partial implementations
- No speculative changes

If requirements are unclear, the agent must **stop and ask**.

---

### 2. Minimal Invasiveness

Agents must:
- Respect existing architecture, patterns, and conventions
- Prefer extension over modification
- Avoid unnecessary refactors
- Avoid stylistic rewrites unless requested

Existing code style always wins over personal preference.

---

### 3. No Silent Assumptions

Agents must not:
- Infer missing business rules
- Invent APIs, flags, or configurations
- Add dependencies, versions, or tools implicitly

When information is missing, the agent must ask before proceeding.

---

### 4. Explicit Reasoning

For non-trivial changes, agents must:
- Think step-by-step
- Validate against existing flows
- Compare new behavior with current behavior

But **must not** add explanatory comments inside production code unless explicitly requested.

---

## Engineering Principles

All code analysis, generation, refactors, and modifications must follow these engineering principles.
They are applied **pragmatically**, not dogmatically.

### SOLID (Pragmatic Application)

- Prefer small, focused units (classes, functions, modules) with a single responsibility.
- Introduce abstractions **only** when there are multiple concrete implementations or clear extension needs.
- Do not introduce inheritance unless polymorphism is required.
- Avoid deep inheritance hierarchies.
- Depend on abstractions only when they already exist or are clearly justified.

### DRY (Avoid Premature Abstraction)

- Avoid duplication **only when the abstraction is stable and obvious**.
- Do not extract shared code prematurely.
- Prefer localized duplication over incorrect or speculative abstractions.

### KISS (Simplicity First)

- Prefer the simplest solution that fully satisfies the spec.
- Avoid unnecessary patterns, layers, or indirection.
- Clarity and readability always win over cleverness.

### YAGNI (No Speculative Design)

- Do not add extensibility points unless explicitly required by the spec.
- Do not prepare for hypothetical future requirements.
- Do not add configuration flags, feature toggles, or hooks unless requested.

### Refactoring Discipline

- Refactor incrementally and safely.
- Never refactor unrelated code unless explicitly requested.
- Refactors must improve **clarity, safety, or correctness**, not personal preference.
- If a refactor risks changing behavior, the agent must stop and ask.

### Code Quality Priorities

When trade-offs exist, prioritize in this order:
1. Correctness
2. Security
3. Clarity
4. Maintainability
5. Performance (only with evidence)

Premature optimization is forbidden.

---

## Refactor Triggers

Agents may suggest or perform **limited refactors** only when one or more of the following conditions are met:

### Allowed Refactor Scenarios

- A change required by the spec would significantly degrade readability without refactoring.
- The current implementation clearly violates existing project conventions.
- There is duplicated logic **within the same module or feature** and the abstraction is obvious and stable.
- A small refactor reduces complexity, nesting, or cognitive load without altering behavior.
- A refactor is necessary to fix a bug, improve testability, or ensure correctness.
- Performance issues are observed and supported by evidence (profiling, metrics, benchmarks).

### Refactor Boundaries

- Refactors must be **minimal, scoped, and incremental**.
- Refactors must not change public APIs unless explicitly requested.
- Refactors must not introduce new architectural patterns.
- Refactors must not affect unrelated features or modules.
- Refactors must preserve existing behavior unless explicitly stated otherwise.

### Mandatory Disclosure

When a refactor is performed, the agent must:
- Explicitly state that a refactor was applied
- Explain the reason briefly
- Confirm that behavior remains unchanged

---

## Red Flags

If any of the following conditions are encountered, the agent **must stop and ask for explicit confirmation before proceeding**:

### Architectural Risks

- A change would alter core architecture or layering.
- A refactor would affect multiple modules or features.
- A change would modify public APIs or contracts.
- The requested change conflicts with existing architectural decisions.

### Behavioral Risks

- Unclear or ambiguous requirements.
- Potential change in observable behavior without explicit approval.
- Removal or modification of existing functionality.
- Changes that could introduce breaking changes.

### Technical Risks

- Introduction of new dependencies or tools.
- Changes to build, CI/CD, or configuration files.
- Migration between frameworks, libraries, or architectural styles.
- Performance optimizations without supporting evidence.

### Safety & Consistency Risks

- Conflicting rules between AGENTS.md, stack rules, or project code.
- Missing or ambiguous stack-specific rules.
- Incomplete test coverage for affected behavior.
- Security implications or exposure of sensitive data.

### Mandatory Action

When a red flag is encountered, the agent must:
1. Stop execution
2. Clearly explain the risk
3. Ask for explicit approval or clarification

Proceeding without confirmation is forbidden.

---

## Language & Communication

- **Code, commits, and documentation**: English only
- **Agent explanations**: English unless user explicitly requests Spanish
- Be concise, precise, and non-verbose

---

## Security Baseline (Global)

Security is **always-on** and **non-optional**.

Agents must:
- Never hardcode secrets, credentials, tokens, or keys
- Never log sensitive data (PII, tokens, headers, payloads)
- Always validate inputs using allow-list strategies
- Avoid reflection, eval, dynamic execution, or unsafe deserialization
- Use secure randomness and cryptography primitives only

If a request conflicts with secure practices, the agent must:
1. Implement the safest alternative
2. Explain why the original approach is unsafe

---

## Dependencies & Configuration

Agents must **never**:
- Add new dependencies
- Change dependency versions
- Modify build configuration files

Unless the user explicitly requests and approves it.

Existing tooling, libraries, and frameworks are constraints, not suggestions.

---

## Testing Philosophy (Global)

When tests are involved:
- Follow existing testing style strictly
- Maintain naming conventions already used
- Cover positive and negative paths
- Do not mix testing frameworks in the same test
- Clean up resources, mocks, and coroutines deterministically

Tests must validate **behavior**, not implementation details.

---

## Code Style & Comments

- Respect existing formatting, naming, and file organization
- Do not add comments to production code unless requested
- In tests, use only `Given / When / Then` labels without descriptions
- Avoid redundant or obvious comments

---

## Memory Model

The agent operates with three conceptual memory layers:

### 1. Global Rules (This File)

- Always applied
- Never overridden
- Defines core behavior and constraints

### 2. Stack-Specific Rules (External)

- Loaded conditionally based on stack detection
- Extend (never contradict) global rules

### 3. Workspace / Contextual Memories

- Observations, known issues, performance notes
- Used for reasoning and validation
- Never treated as executable rules

---

## Stack Detection

Before acting, the agent must identify the active stack(s):

- **Mobile / Android** → Kotlin, Gradle, Robolectric, MockK, Coroutines
- **Mobile / iOS** → Swift, SwiftUI/UIKit, XCTest
- **Backend** → Kotlin/JVM, Spring/Ktor, Arrow, REST APIs
- **Frontend** → React, Node.js, Nordic, Jest, Cypress

Multiple stacks may apply simultaneously.

---

## Dynamic Rule Inclusion

Stack-specific rules are **not duplicated here**.

Instead, the agent must dynamically load them based on context:

```
IF stack == mobile/android  → rules-mobile.yaml
IF stack == backend         → rules-backend.yaml
IF stack == frontend        → rules-frontend.yaml
```

Rules are:
- Additive
- Scoped
- Explicitly referenced

If a required rules file is missing or ambiguous, the agent must stop and ask.

---

## Conflict Resolution

If multiple rules apply:

1. Global rules win over everything
2. Stack-specific rules win over memories
3. Existing project code wins over generic guidance

If conflict remains → **ask the user**.

---

## What the Agent Must Never Do

- Invent requirements
- Ignore security constraints
- Add dependencies silently
- Change architecture implicitly
- Modify unrelated code
- Over-optimize without evidence

---

## Change Management

When making changes, the agent must:
- Keep commits focused
- Preserve backward compatibility unless told otherwise
- Avoid breaking public APIs
- Update tests and documentation when behavior changes

---

## Final Rule

When in doubt:

> **Stop, explain the uncertainty, and ask before proceeding.**

This is always preferable to guessing.
