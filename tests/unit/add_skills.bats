#!/usr/bin/env bats

load ../helpers/common

setup() {
  setup_workdir
}

teardown() {
  teardown_workdir
}

@test "add-skills creates all skills and AVAILABLE_SKILLS.xml" {
  run "$ROOT_DIR/scripts/add-skills"
  assert_success
  assert_dir_exists "skills/analyze"
  assert_file_exists "skills/analyze/SKILL.md"
  assert_file_exists "skills/AVAILABLE_SKILLS.xml"
  run grep -q "<name>troubleshoot</name>" "skills/AVAILABLE_SKILLS.xml"
  assert_success
}

@test "add-skills restores missing SKILL.md" {
  run "$ROOT_DIR/scripts/add-skills"
  rm "skills/analyze/SKILL.md"
  run "$ROOT_DIR/scripts/add-skills"
  assert_success
  assert_file_exists "skills/analyze/SKILL.md"
}

@test "add-skills skips missing template and excludes it from AVAILABLE_SKILLS.xml" {
  rm -rf "$AGENT47_HOME/templates/skills/analyze"

  run "$ROOT_DIR/scripts/add-skills"
  assert_success
  [ ! -d "skills/analyze" ]
  assert_file_exists "skills/AVAILABLE_SKILLS.xml"
  run grep -q "<name>analyze</name>" "skills/AVAILABLE_SKILLS.xml"
  [ "$status" -ne 0 ]
  run grep -q "<name>implement</name>" "skills/AVAILABLE_SKILLS.xml"
  assert_success

  # Restaurar templates
  rm -rf "$AGENT47_HOME/templates"
  cp -R "$ROOT_DIR/templates" "$AGENT47_HOME/"
}

@test "add-skills without --force preserves existing SKILL.md contents" {
  run "$ROOT_DIR/scripts/add-skills"
  echo "custom" > skills/analyze/SKILL.md

  run "$ROOT_DIR/scripts/add-skills"
  assert_success
  run grep -q "custom" skills/analyze/SKILL.md
  assert_success
}

@test "add-skills with --force overwrites existing skills from templates" {
  run "$ROOT_DIR/scripts/add-skills"
  echo "custom" > skills/analyze/SKILL.md

  run "$ROOT_DIR/scripts/add-skills" --force
  assert_success
  run grep -q "custom" skills/analyze/SKILL.md
  [ "$status" -ne 0 ]
}

@test "add-skills fails on invalid SKILL.md template" {
  mkdir -p "$AGENT47_HOME/templates/skills/analyze"
  cat > "$AGENT47_HOME/templates/skills/analyze/SKILL.md" <<'EOF'
---
name:
description:
---
EOF

  run "$ROOT_DIR/scripts/add-skills" --force
  [ "$status" -ne 0 ]
  assert_contains "$output" "Invalid skill template"
  [ ! -f "skills/AVAILABLE_SKILLS.xml" ]

  # Restore templates for subsequent tests
  rm -rf "$AGENT47_HOME/templates/skills"
  cp -R "$ROOT_DIR/templates/skills" "$AGENT47_HOME/templates/"
}
