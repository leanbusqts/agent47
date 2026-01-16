#!/usr/bin/env bats

load ../helpers/common

setup() {
  setup_workdir
}

teardown() {
  teardown_workdir
}

@test "bin/a47 add-agent uses local scripts when not in PATH" {
  # PATH sin scripts instalados
  PATH="/usr/bin:/bin"

  run "$ROOT_DIR/bin/a47" add-agent
  assert_success
  assert_file_exists "AGENTS.md"
}

@test "uninstall removes installed scripts" {
  run "$ROOT_DIR/bin/a47" install
  assert_success
  # Verifica instalados en ~/bin
  assert_file_exists "$HOME/bin/add-agent"
  assert_file_exists "$HOME/bin/reload-skills"

  run "$ROOT_DIR/bin/a47" uninstall
  assert_success
  [ ! -f "$HOME/bin/add-agent" ]
  [ ! -f "$HOME/bin/reload-skills" ]
  [ ! -L "$HOME/bin/a47" ]
}
