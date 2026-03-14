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
  assert_file_exists "$AGENT47_HOME/bin/a47"
  assert_file_exists "$AGENT47_HOME/templates/AGENTS.md"
  assert_file_exists "$AGENT47_HOME/templates/specs/spec.yml"
  assert_file_exists "$AGENT47_HOME/scripts/skill-utils.sh"
  assert_file_exists "$AGENT47_HOME/scripts/add-cli-prompt"
  assert_file_exists "$AGENT47_HOME/scripts/add-agent-prompt"
  assert_file_exists "$AGENT47_HOME/scripts/add-snapshot-prompt"
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

  ln -sf "$AGENT47_HOME/bin/a47" "$HOME/bin/a47"

  run "$ROOT_DIR/bin/a47" uninstall
  assert_success
  [ ! -f "$HOME/bin/add-agent" ]
  [ ! -f "$HOME/bin/reload-skills" ]
  [ ! -f "$HOME/bin/add-cli-prompt" ]
  [ ! -f "$HOME/bin/add-agent-prompt" ]
  [ ! -f "$HOME/bin/add-snapshot-prompt" ]
  [ ! -L "$HOME/bin/a47" ]
}

@test "install.sh rejects unexpected arguments" {
  run "$ROOT_DIR/install.sh" unexpected-arg
  [ "$status" -ne 0 ]
  assert_contains "$output" "Usage: ./install.sh [--force]"
}

@test "install fails fast when launcher cannot be written" {
  mkdir -p "$AGENT47_HOME/bin"
  chmod 500 "$AGENT47_HOME/bin"

  run "$ROOT_DIR/bin/a47" install
  [ "$status" -ne 0 ]
  assert_not_contains "$output" "[OK] Installed a47 launcher"

  chmod 700 "$AGENT47_HOME/bin"
}

@test "install fails fast when a core install asset is missing" {
  mv "$ROOT_DIR/templates/AGENTS.md" "$ROOT_DIR/templates/AGENTS.md.bak"

  run "$ROOT_DIR/bin/a47" install
  [ "$status" -ne 0 ]
  assert_contains "$output" "Required install asset missing"

  mv "$ROOT_DIR/templates/AGENTS.md.bak" "$ROOT_DIR/templates/AGENTS.md"
}
