# Ownership

`agent47` has two distinct layers:

1. the source repository that builds and verifies `agent47` itself
2. the scaffold payload under `templates/` that gets copied into target projects

## Source repo vs target project

Source repo files exist to implement and maintain `agent47`:

- `bin/`
- `cmd/`
- `internal/`
- `scripts/`
- `docs/`
- `tests/`
- root metadata such as `README.md`, `VERSION`, `AGENTS.md`, `SNAPSHOT.md`, `SPEC.md`, and `PLAN.md`

Target project files are managed from the template payload:

- `AGENTS.md`
- `rules/*.yaml`
- `skills/*`
- `skills/AVAILABLE_SKILLS.xml`

Target project files preserved during refresh:

- `README.md`
- `specs/spec.yml`
- `SNAPSHOT.md`
- `SPEC.md`

During `afs add-agent --force`, the managed target set is reconciled to the current manifest and template payload. That means stale managed rules or skills are removed, while preserved targets stay untouched.

## Ownership contract

The authoritative ownership contract lives in `templates/manifest.txt`.

It defines:

- which rule templates ship with the scaffold
- which targets are managed
- which targets are preserved
- which template files and directories are required for install/bootstrap

The runtime enforces that contract through:

- `internal/manifest/` for parsing and validation
- `internal/bootstrap/` for project staging, commit, rollback, and destructive refresh behavior
- `internal/install/` for managed runtime/template installation
- `internal/skills/` for skill validation and XML generation

## Practical implications

- Files under managed paths belong to the scaffold contract, not to ad hoc project customizations.
- `afs add-agent` is conservative and preserves existing managed files.
- `afs add-agent --force` is intentionally destructive inside managed paths.
- Local custom files under `rules/` or `skills/` can be replaced or removed during `--force`.
- Project-owned docs such as `README.md` and root `SPEC.md` are intentionally outside the default managed scaffold.

## Repo shell surface

The old shell library split is no longer the product boundary. The remaining repo shell surface is small:

- `bin/afs` as the checkout launcher
- `scripts/lint-shell` as a maintainer check
- shell and Bats coverage under `tests/`

The product runtime and ownership enforcement now live primarily in the Go code.
