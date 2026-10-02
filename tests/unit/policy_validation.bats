#!/usr/bin/env bats

load ../helpers/common

setup() { setup_workdir; }
teardown() { teardown_workdir; }

make_policy_fixture() {
  local fixture="$TEST_WORKDIR/policy-fixture"
  mkdir -p "$fixture/scripts" "$fixture/templates/base"
  cp "$ROOT_DIR/AGENTS.md" "$fixture/AGENTS.md"
  cp "$ROOT_DIR/templates/base/AGENTS.md" "$fixture/templates/base/AGENTS.md"
  cp "$ROOT_DIR/scripts/check-agents-md.sh" "$fixture/scripts/check-agents-md.sh"
  printf '%s\n' "$fixture"
}

@test "root and base AGENTS policies stay identical" {
  run diff -q "$ROOT_DIR/AGENTS.md" "$ROOT_DIR/templates/base/AGENTS.md"
  assert_success
}

@test "root and base global security rules stay identical" {
  run diff -q "$ROOT_DIR/rules/security-global.yaml" "$ROOT_DIR/templates/base/rules/security-global.yaml"
  assert_success
}

@test "Lite policy includes the approved language rule" {
  run grep -F "Use the strongest internal reasoning language available. Default to English for technical tasks. Preserve output language as requested by the user." "$ROOT_DIR/AGENTS.md"
  assert_success
}

@test "Lite policy declares pragmatic engineering principles" {
  for principle in \
    "## Engineering Principles" \
    "**KISS / YAGNI:**" \
    "**DRY:**" \
    "**SOLID:**" \
    "composition over inheritance" \
    "least surprise" \
    "optimize only with evidence" \
    "**Proportionality / JEDUF:**" \
    "**Two-way doors / Last Responsible Moment:**" \
    "**Definition of Done:**"; do
    run grep -F "$principle" "$ROOT_DIR/AGENTS.md"
    assert_success
  done
}

@test "removed skills prompts and task-spec templates are absent" {
  [ ! -d "$ROOT_DIR/skills" ]
  [ ! -d "$ROOT_DIR/templates/base/skills" ]
  [ ! -d "$ROOT_DIR/templates/base/prompts" ]
  [ ! -e "$ROOT_DIR/templates/base/.agents/specs/spec.yml" ]
  [ ! -d "$ROOT_DIR/templates/bundles/project-cli/skills" ]
}

@test "manifest manages only exact AGENTS and base rule paths" {
  manifest="$ROOT_DIR/templates/manifest.txt"
  run grep -A5 '^\[managed_targets\]' "$manifest"
  assert_success
  assert_contains "$output" "AGENTS.md"
  assert_contains "$output" "rules/security-global.yaml"
  assert_contains "$output" "rules/security-shell.yaml"
  assert_contains "$output" "rules/rules-cross.yaml"
  assert_not_contains "$output" "rules/*.yaml"
  assert_not_contains "$output" "skills"

  run grep -A2 '^\[generated_targets\]' "$manifest"
  assert_success
  assert_contains "$output" ".agent47/context.md"

  run grep -A8 '^\[preserved_targets\]' "$manifest"
  assert_success
  assert_contains "$output" ".agents/"
  assert_contains "$output" "skills/"
  assert_contains "$output" "prompts/"

  run grep -A6 '^\[force_cleanup_targets\]' "$manifest"
  assert_success
  assert_contains "$output" "rules/"
  assert_contains "$output" "skills/"
  assert_contains "$output" "prompts/"
  assert_contains "$output" "specs/spec.yml"
  assert_contains "$output" ".agents/specs/spec.yml"
}

@test "public docs expose the Lite command surface" {
  for file in README.md RUNBOOK.md SPEC.md SNAPSHOT.md; do
    run grep -F "afs init" "$ROOT_DIR/$file"
    assert_success
    run grep -F "afs map" "$ROOT_DIR/$file"
    assert_success
  done

  run grep -F "afs add-agent                 bootstrap" "$ROOT_DIR/README.md"
  [ "$status" -ne 0 ]
}

@test "policy checker passes" {
  run bash -c 'cd "$1" && bash scripts/check-agents-md.sh' _ "$ROOT_DIR"
  assert_success
}

@test "policy checker rejects a missing engineering principles section" {
  fixture="$(make_policy_fixture)"
  for policy in "$fixture/AGENTS.md" "$fixture/templates/base/AGENTS.md"; do
    awk '
      /^## Engineering Principles$/ { skip = 1 }
      /^## Filesystem And Approval Boundaries$/ { skip = 0 }
      !skip { print }
    ' "$policy" > "$policy.tmp"
    mv "$policy.tmp" "$policy"
  done

  run bash -c 'cd "$1" && bash scripts/check-agents-md.sh' _ "$fixture"
  [ "$status" -ne 0 ]
  assert_contains "$output" "required section '## Engineering Principles' missing"
}

@test "policy checker rejects an unexpected section heading" {
  fixture="$(make_policy_fixture)"
  printf '\n## Unexpected Section\n' >> "$fixture/AGENTS.md"
  printf '\n## Unexpected Section\n' >> "$fixture/templates/base/AGENTS.md"

  run bash -c 'cd "$1" && bash scripts/check-agents-md.sh' _ "$fixture"
  [ "$status" -ne 0 ]
  assert_contains "$output" "expected 20 section headings, found 21"
}
