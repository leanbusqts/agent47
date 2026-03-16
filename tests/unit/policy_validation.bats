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

@test "template manifest exists with required sections" {
  assert_file_exists "$ROOT_DIR/templates/manifest.txt"
  run grep -F "[rule_templates]" "$ROOT_DIR/templates/manifest.txt"
  assert_success
  run grep -F "[managed_targets]" "$ROOT_DIR/templates/manifest.txt"
  assert_success
  run grep -F "[preserved_targets]" "$ROOT_DIR/templates/manifest.txt"
  assert_success
  run grep -F "[required_template_files]" "$ROOT_DIR/templates/manifest.txt"
  assert_success
  run grep -F "[required_template_dirs]" "$ROOT_DIR/templates/manifest.txt"
  assert_success
}

@test "manifest rule templates all exist in templates/rules" {
  while IFS= read -r rule_file; do
    [ -n "$rule_file" ] || continue
    assert_file_exists "$ROOT_DIR/templates/rules/$rule_file"
  done < <(awk '
    $0 == "[rule_templates]" { in_section=1; next }
    /^\[/ && in_section { exit }
    in_section && NF { print }
  ' "$ROOT_DIR/templates/manifest.txt")
}

@test "manifest managed and preserved targets do not overlap exactly" {
  managed="$(awk '
    $0 == "[managed_targets]" { in_section=1; next }
    /^\[/ && in_section { exit }
    in_section && NF { print }
  ' "$ROOT_DIR/templates/manifest.txt" | sort)"
  preserved="$(awk '
    $0 == "[preserved_targets]" { in_section=1; next }
    /^\[/ && in_section { exit }
    in_section && NF { print }
  ' "$ROOT_DIR/templates/manifest.txt" | sort)"

  run bash -c "comm -12 <(printf '%s\n' \"$managed\") <(printf '%s\n' \"$preserved\")"
  assert_success
  [ -z "$output" ]
}

@test "manifest managed and preserved targets match runtime contract" {
  for target in AGENTS.md 'rules/*.yaml' 'skills/*' 'skills/AVAILABLE_SKILLS.xml'; do
    run grep -Fx "$target" "$ROOT_DIR/templates/manifest.txt"
    assert_success
  done

  for target in README.md specs/spec.yml SNAPSHOT.md; do
    run grep -Fx "$target" "$ROOT_DIR/templates/manifest.txt"
    assert_success
  done
}

@test "runtime managed and preserved target helpers expose the canonical contract" {
  run bash -c "source '$ROOT_DIR/scripts/lib/managed-files.sh'; project_managed_targets"
  assert_success
  assert_contains "$output" "AGENTS.md"
  assert_contains "$output" "rules/*.yaml"
  assert_contains "$output" "skills/*"
  assert_contains "$output" "skills/AVAILABLE_SKILLS.xml"

  run bash -c "source '$ROOT_DIR/scripts/lib/managed-files.sh'; project_preserved_files"
  assert_success
  assert_contains "$output" "README.md"
  assert_contains "$output" "specs/spec.yml"
  assert_contains "$output" "SNAPSHOT.md"
}

@test "manifest required template files all exist" {
  while IFS= read -r rel_path; do
    [ -n "$rel_path" ] || continue
    assert_file_exists "$ROOT_DIR/templates/$rel_path"
  done < <(awk '
    $0 == "[required_template_files]" { in_section=1; next }
    /^\[/ && in_section { exit }
    in_section && NF { print }
  ' "$ROOT_DIR/templates/manifest.txt")
}

@test "manifest required template dirs all exist" {
  while IFS= read -r rel_path; do
    [ -n "$rel_path" ] || continue
    assert_dir_exists "$ROOT_DIR/templates/$rel_path"
  done < <(awk '
    $0 == "[required_template_dirs]" { in_section=1; next }
    /^\[/ && in_section { exit }
    in_section && NF { print }
  ' "$ROOT_DIR/templates/manifest.txt")
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
  run grep -F 'applies_to: "shell"' "$ROOT_DIR/templates/rules/security-shell.yaml"
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

@test "templates payload does not include macOS system artifacts" {
  run find "$ROOT_DIR/templates" -name '.DS_Store' -print
  assert_success
  [ -z "$output" ]
}
