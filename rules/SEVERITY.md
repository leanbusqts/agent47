# Severity Levels

This file defines the severity scale used across all `rules/*.yaml` files.

## Scale

### `critical`
- **Definition**: Material risk to security, data integrity, or compliance.
- **Detection**: SHOULD be detectable by automated rules (linter, CI, IDE).
- **Operational consequence**: **Blocks merge.** Must be fixed before continuing.
- **Examples**: Hardcoded secrets (`SEC-global-001`); SQL/command injection
  (`SEC-py-002`, `SEC-shell-002`); auth bypass (`BE-006`); force-unwrap in
  production paths (`MO-005`).

### `high`
- **Definition**: Significant defect of architecture, reliability, or operability.
- **Detection**: Often automated, sometimes review-detected.
- **Operational consequence**: **Requires explicit review approval.** May merge
  with documented compensating control.
- **Examples**: External calls without timeouts (`BE-007`); missing observability
  around fallbacks (`X-obs-001`); disabled TLS verification with justification
  (`SEC-global-008`).

### `medium`
- **Definition**: Convention or consistency issue affecting maintainability.
- **Detection**: Usually review-detected; some automation possible.
- **Operational consequence**: **Review comment.** Does not block.
- **Examples**: Naming conventions (`MO-002`); state management style (`FE-012`).

### `low`
- **Definition**: Preference or style.
- **Detection**: Auto-formatter / linter ideal.
- **Operational consequence**: **Optional.** Not enforced.
- **Examples**: `any` vs explicit types (`FE-007`); comment style (`X-doc-001`).

## Conflict resolution

When two rules at the same severity disagree, escalate per AGENTS.md
"Authority Order":
1. User instructions
2. Nearest `AGENTS.md`
3. Security: global → language → stack
4. Stack rules
5. Specs
6. Code/tests
7. Memories

When a `medium` rule like `mobile:consistency` (MO-004) conflicts with a `high`
rule like `mobile:architecture-default` (MO-008), the `high` rule wins; the
medium rule yields. Such yielding must be documented in the PR description.
