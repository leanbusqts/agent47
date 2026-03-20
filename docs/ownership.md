# Ownership

`agent47` has two distinct layers that used to blur together in the implementation:

- repo-owned source
  - the CLI, installer, tests, and documentation that make `agent47` work

- project scaffold payload
  - the files under `templates/` that are copied into a target repository

## Source repo vs target project

Source repo files live here to build and maintain `agent47` itself:

- `bin/`
- `scripts/`
- `docs/`
- `tests/`
- root metadata such as `README.md`, `VERSION`, and this repo's own `AGENTS.md`

Target project files are managed from the template payload:

- `AGENTS.md`
- `rules/*.yaml`
- `skills/*`
- `skills/AVAILABLE_SKILLS.xml`

Target project files preserved during refresh:

- `README.md`
- `specs/spec.yml`
- `SNAPSHOT.md`

During `afs add-agent --force`, the managed target set is reconciled to the current manifest and template payload.
That means stale managed rules or skills are removed, while preserved targets stay untouched.
Custom project files can exist under `rules/` and `skills/`, but they are at risk of replacement or removal during `afs add-agent --force`.

## Code boundaries

The ownership boundary is now encoded in shell modules instead of being implicit inside one large script:

- `templates/manifest.txt`
  - declarative contract for rule templates, required assets, and the managed/preserved target lists that runtime validates against

- `scripts/lib/managed-files.sh`
  - canonical source for managed and preserved file definitions, plus manifest contract validation

- `scripts/lib/bootstrap.sh`
  - project bootstrap transaction, staging, commit, and rollback logic

- `scripts/lib/skill-utils.sh`
  - skill validation and `AVAILABLE_SKILLS.xml` generation helpers

- `scripts/lib/runtime-env.sh`
  - shared runtime bootstrap for installed and repo-local command execution

- `scripts/lib/test-runtime.sh`
  - isolated test-environment bootstrap for the repo test runner

- `scripts/add-agent`
  - thin command entrypoint

- `scripts/test`
  - executable repo test runner, intentionally not treated as a library

This keeps the product code separate from the scaffold content and makes it clearer which rules apply to the repo source versus installed project files.
