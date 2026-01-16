#!/usr/bin/env bats

load ../helpers/common

setup() {
  setup_workdir
}

teardown() {
  teardown_workdir
}

@test "add-agent copies core files and creates README" {
  run "$ROOT_DIR/scripts/add-agent"
  assert_success
  assert_file_exists "AGENTS.md"
  assert_file_exists "rules/rules-mobile.yaml"
  assert_file_exists "rules/rules-frontend.yaml"
  assert_file_exists "rules/rules-backend.yaml"
  assert_file_exists "README.md"
}

@test "add-agent with prompt base adds prompt file" {
  run "$ROOT_DIR/scripts/add-agent" --prompt base
  assert_success
  assert_file_exists "prompts/agent-prompt-base.txt"
}

@test "add-agent with skills copies skills and AVAILABLE_SKILLS.xml" {
  run "$ROOT_DIR/scripts/add-agent" --with-skills
  assert_success
  assert_dir_exists "skills/analyze"
  assert_file_exists "skills/AVAILABLE_SKILLS.xml"
  run grep -q "<name>analyze</name>" "skills/AVAILABLE_SKILLS.xml"
  assert_success
}

@test "add-agent warns on invalid prompt choice" {
  run "$ROOT_DIR/scripts/add-agent" --prompt invalid
  [ "$status" -ne 0 ]
  assert_contains "$output" "Invalid prompt choice"
}

@test "add-agent fails if a required template is missing" {
  rm "$AGENT47_HOME/templates/AGENTS.md"

  run "$ROOT_DIR/scripts/add-agent"
  [ "$status" -ne 0 ]
  assert_contains "$output" "Template not found"
  assert_contains "$output" "required templates missing"
  [ ! -f "AGENTS.md" ]
  [ ! -f "rules/rules-backend.yaml" ]
  [ ! -f "rules/rules-frontend.yaml" ]
  [ ! -f "rules/rules-mobile.yaml" ]

  # Restaurar templates
  rm -rf "$AGENT47_HOME/templates"
  cp -R "$ROOT_DIR/templates" "$AGENT47_HOME/"
}
