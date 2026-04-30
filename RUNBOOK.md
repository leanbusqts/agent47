# RUNBOOK

`RUNBOOK.md` is the operational guide for using `agent47` in practice. Use `README.md` for the entrypoint and high-level architecture, and `SPEC.md` for the formal product contract.

## Install

Unix-like systems:

```bash
./install.sh
./install.sh --force
./install.sh --non-interactive
```

Windows:

```powershell
.\install.ps1
.\install.ps1 -Force
.\install.ps1 -NonInteractive
```

Verify the local setup:

```bash
afs doctor
afs doctor --check-update
```

Contributor checks:

```bash
make test
make go-test
make go-build
make lint-shell
make smoke-install
```

`install.sh` and `install.ps1` are the supported public install entrypoints. There is no supported public `afs install`, `afs upgrade`, `afs templates`, `afs check-update`, `afs add-spec`, `afs add-cli-prompt`, `afs add-default-skills`, or `afs init-agent` command.

## First Steps

1. Install the tool locally.
2. Verify the local setup with `afs doctor`.
3. Enter the target project.
4. Run `afs analyze` if you want to inspect the resolved install set first.
5. Run `afs add-agent`.

## Bootstrap A Project

Inside the target project:

```bash
afs add-agent
```

`afs analyze` is read-only. It reports detected project types, confidence, and the install set that `add-agent` would use.

```bash
afs analyze
afs analyze --json
afs analyze --verbose
```

`afs add-agent` analyzes the repo first and then bootstraps:

- `AGENTS.md`
- the resolved `rules/*.yaml` set for the detected project type
- the resolved curated `skills/*`
- `skills/AVAILABLE_SKILLS.xml`
- `skills/AVAILABLE_SKILLS.json`
- `skills/SUMMARY.md`
- `prompts/agent-prompt.txt`
- `prompts/ss-prompt.txt`
- an empty `README.md` if missing
- `specs/spec.yml` if missing

Existing managed files are preserved unless you use `--force`.

Useful inspection modes:

```bash
afs add-agent --preview
afs add-agent --dry-run
afs add-agent --bundle cli --bundle scripts --preview
```

In interactive terminals, `afs add-agent` asks for confirmation before writing. Use `--yes` to skip that confirmation.

Common example flows:

```bash
# Empty or low-signal repo
afs analyze
afs add-agent --preview
afs add-agent --yes

# CLI repo with scripts
afs analyze
afs add-agent --preview --bundle cli --bundle scripts
afs add-agent --yes

# Unresolved conflict
afs analyze --verbose
afs add-agent --preview

# Legacy scaffold migration
afs add-agent --force --yes
```

## Refresh An Older Project

If the project already has an older `agent47` scaffold:

```bash
afs add-agent --force
```

This is a fresh install of the managed scaffold in the current project. It:

- replaces `AGENTS.md`
- reconciles the resolved `rules/*.yaml`
- replaces `skills/*`
- regenerates the managed skills indexes
- removes stale managed rules and skills no longer selected for the resolved bundle set

This preserves:

- `README.md`
- `specs/spec.yml`
- `SNAPSHOT.md`
- `SPEC.md`

If you keep project-specific files under `rules/` or `skills/`, expect them to be replaced or removed by `--force`.

## Skills-Only Mode

```bash
afs add-agent --only-skills
afs add-agent --only-skills --force
```

This mode only manages:

- `skills/*`
- `skills/AVAILABLE_SKILLS.xml`
- `skills/AVAILABLE_SKILLS.json`
- `skills/SUMMARY.md`

It does not touch `AGENTS.md` or `rules/*.yaml`.

Behavior differences:

- without `--force`, existing invalid skill files are preserved but omitted from the generated skills indexes
- with `--force`, the managed skills directory is replaced with the current template set
- the same validated skill set is rendered into the XML, JSON, and Markdown indexes

## Prompt Helpers

Refresh or create the general agent prompt:

```bash
afs add-agent-prompt
afs add-agent-prompt --force
```

Print the snapshot/spec helper prompt:

```bash
afs add-ss-prompt
```

When a supported clipboard tool is available, `afs add-ss-prompt` copies the prompt directly. Otherwise it prints the prompt to stdout.

## Managed Vs Preserved Files

Managed targets:

- `AGENTS.md`
- `rules/*.yaml`
- `skills/*`
- `skills/AVAILABLE_SKILLS.xml`
- `skills/AVAILABLE_SKILLS.json`
- `skills/SUMMARY.md`

Preserved targets:

- `README.md`
- `specs/spec.yml`
- `SNAPSHOT.md`
- `SPEC.md`

Ownership is defined by `templates/manifest.txt`.

Practical implications:

- Files under managed paths belong to the scaffold contract, not to ad hoc project customizations.
- `afs add-agent` is conservative and preserves existing managed files.
- `afs add-agent --force` is intentionally destructive inside managed paths.
- Local custom files under `rules/` or `skills/` can be replaced or removed during `--force`.

## Update Checks

`afs doctor` skips update checks by default.

Use:

```bash
afs doctor --check-update
afs doctor --check-update-force
afs doctor --fail-on-warn
afs doctor --check-update --fail-on-warn
```

Behavior:

- if `AGENT47_VERSION_URL` is configured, `agent47` compares local and remote `VERSION`
- otherwise, from a git checkout with an upstream branch, it compares `HEAD` to the upstream branch
- regular git-based checks do not run `git fetch`
- `afs doctor --check-update-force` performs `git fetch --quiet` before comparing
- remote checks may be cached; git-tracking checks are evaluated fresh
- `doctor` flags can be combined, for example `afs doctor --check-update --fail-on-warn`

## Operational Notes

- Unix-like installs write managed assets under `~/.agent47` and publish `afs` plus helper commands into `~/bin`
- Windows installs default to `%LOCALAPPDATA%\agent47` and use the managed bin directory on PATH
- `--non-interactive` avoids interactive shell rc prompts
- repo-local `bin/afs` is for checkout-based development, not the installed runtime path
- checkout-based execution depends on Go unless you provide `AGENT47_GO_CLI` or an explicit `AGENT47_REPO_CLI`
- `afs uninstall` removes published commands and managed runtime assets

## Use With Agent CLIs And IDEs

`agent47` is a repository convention, not a vendor-specific integration.

Recommended workflow:

1. Open the repository root.
2. Ensure the agent reads `AGENTS.md`.
3. Let `AGENTS.md` drive the next reads, including relevant `rules/*.yaml` or, in template-source repos like `agent47`, `templates/base/rules/*.yaml` plus the relevant `templates/bundles/*/rules/*.yaml`.
4. Use `specs/spec.yml` only when work actually needs a written spec or plan.

Minimal instruction text for tools that do not discover repo policy reliably:

```text
Read AGENTS.md first and follow the applicable rules before making changes.
```
