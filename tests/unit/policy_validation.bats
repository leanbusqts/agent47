#!/usr/bin/env bats

load ../helpers/common

setup() { setup_workdir; }
teardown() { teardown_workdir; }

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
