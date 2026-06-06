# Approval Process

Several rules use the phrase "unless explicitly approved." This file defines what
"explicitly approved" means and how to record approvals.

## Required for

- `SEC-global-005` — sending repository code, secrets, or user data to external systems
- `SEC-js-ts-002` — storing tokens in browser-accessible storage
- `X-deps-001` — adding, removing, or upgrading any dependency

## Process

1. **Open a PR** describing the change.
2. **Add an `APPROVAL.md` block** to the PR description with:
   - **What**: the action requiring approval
   - **Why**: rationale and alternatives considered
   - **Risk**: identified risks and compensating controls
   - **Owner**: the engineer accountable
   - **Reviewer**: the security/architecture reviewer who approved
3. **Get review** from the relevant code owner (security, architecture, or stack lead).
4. **Reference the approval** in code with a comment containing the PR URL when
   the rule has long-term implications.

## Sections referenced from rules

### `#external-data-flow`
For `SEC-global-005`. The PR description must list the external destination, the
data classes being sent, retention at the destination, and the legal basis.

### `#dependencies`
For `X-deps-001`. The PR description must include: alternatives considered,
license, last-release date, open-issue count, pinned version + hash.
