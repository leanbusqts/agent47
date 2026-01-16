#!/usr/bin/env bats

load ../helpers/common

setup() {
  setup_workdir
}

teardown() {
  teardown_workdir
}

@test "install copies templates and skill-utils into AGENT47_HOME" {
  run "$ROOT_DIR/bin/a47" install --force
  assert_success
  assert_file_exists "$AGENT47_HOME/templates/AGENTS.md"
  assert_file_exists "$AGENT47_HOME/templates/specs/spec.yml"
  assert_file_exists "$AGENT47_HOME/scripts/skill-utils.sh"
}

@test "install creates a backup when overwriting templates" {
  run "$ROOT_DIR/bin/a47" install --force
  assert_success

  # Segunda instalación genera backup rotativo
  run "$ROOT_DIR/bin/a47" install --force
  assert_success
  run find "$AGENT47_HOME" -maxdepth 1 -type d -name "templates.bak.*"
  assert_success
  assert_contains "$output" "templates.bak."
}

@test "upgrade succeeds and refreshes scripts" {
  run "$ROOT_DIR/bin/a47" upgrade
  assert_success
}

@test "uninstall removes scripts from ~/bin" {
  run "$ROOT_DIR/bin/a47" install
  assert_success

  run "$ROOT_DIR/bin/a47" uninstall
  assert_success
  [ ! -f "$HOME/bin/add-agent" ]
  [ ! -f "$HOME/bin/reload-skills" ]
  [ ! -f "$HOME/bin/add-agent-prompt-base" ]
  [ ! -L "$HOME/bin/a47" ]
}
