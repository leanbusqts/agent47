#!/usr/bin/env bats

load ../helpers/common

setup() {
  setup_workdir
}

teardown() {
  teardown_workdir
}

@test "AGENTS template stays compact and includes required sections" {
  run wc -l "$ROOT_DIR/templates/AGENTS.md"
  assert_success
  lines="$(echo "$output" | awk '{print $1}')"
  [ "$lines" -le 300 ]

  run grep -F "## Filesystem And Approval Boundaries" "$ROOT_DIR/templates/AGENTS.md"
  assert_success
  run grep -F "### Always" "$ROOT_DIR/templates/AGENTS.md"
  assert_success
  run grep -F "### Ask" "$ROOT_DIR/templates/AGENTS.md"
  assert_success
  run grep -F "### Never" "$ROOT_DIR/templates/AGENTS.md"
  assert_success
  run grep -F "## Dependency Policy" "$ROOT_DIR/templates/AGENTS.md"
  assert_success
}

@test "single prompt template exists" {
  assert_file_exists "$ROOT_DIR/templates/prompts/agent-prompt.txt"
  [ ! -f "$ROOT_DIR/templates/prompts/agent-prompt-base.txt" ]
  [ ! -f "$ROOT_DIR/templates/prompts/agent-prompt-skills.txt" ]
  [ ! -f "$ROOT_DIR/templates/prompts/agent-prompt-sdd.txt" ]
}

@test "repo root AGENTS exists and matches the template" {
  assert_file_exists "$ROOT_DIR/AGENTS.md"
  run cmp -s "$ROOT_DIR/AGENTS.md" "$ROOT_DIR/templates/AGENTS.md"
  assert_success
}

@test "security templates expose unique SEC ids" {
  run sh -c "grep -ho 'id:[[:space:]]*\"SEC-[^\"]*\"' '$ROOT_DIR'/templates/rules/security-*.yaml | sed -E 's/.*\"(SEC-[^\"]*)\"/\\1/' | sort | uniq -d"
  assert_success
  [ -z "$output" ]
}

@test "security templates include severity and applies_to" {
  for file in "$ROOT_DIR"/templates/rules/security-*.yaml; do
    run grep -F "severity:" "$file"
    assert_success
    run grep -F "applies_to:" "$file"
    assert_success
  done
}

@test "stack rules reference security ids instead of copying security topics" {
  run grep -F "refs:" "$ROOT_DIR/templates/rules/rules-backend.yaml"
  assert_success
  run grep -F "refs:" "$ROOT_DIR/templates/rules/rules-frontend.yaml"
  assert_success
  run grep -F 'applies_to: "backend|mobile"' "$ROOT_DIR/templates/rules/security-java-kotlin.yaml"
  assert_success
  run grep -F 'applies_to: "backend|mobile"' "$ROOT_DIR/templates/rules/security-csharp.yaml"
  assert_success
}

@test "dependency approval policy is present across AGENTS and stack rules" {
  run grep -F "dependencies:approval" "$ROOT_DIR/templates/rules/rules-backend.yaml"
  assert_success
  run grep -F "dependencies:approval" "$ROOT_DIR/templates/rules/rules-frontend.yaml"
  assert_success
  run grep -F "mobile:dependencies" "$ROOT_DIR/templates/rules/rules-mobile.yaml"
  assert_success
  run grep -F "New dependencies or dependency changes require approval" "$ROOT_DIR/AGENTS.md"
  assert_success
}
