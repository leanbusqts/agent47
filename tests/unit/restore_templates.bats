#!/usr/bin/env bats

load ../helpers/common

setup() {
  setup_workdir
}

teardown() {
  teardown_workdir
}

@test "templates --restore-latest restores from latest backup" {
  run "$ROOT_DIR/bin/a47" install
  assert_success
  run "$ROOT_DIR/bin/a47" install
  assert_success

  rm -rf "$AGENT47_HOME/templates"
  mkdir -p "$AGENT47_HOME/templates"
  echo "broken" > "$AGENT47_HOME/templates/AGENTS.md"

  run "$ROOT_DIR/bin/a47" templates --restore-latest
  assert_success
  run grep -q "AGENTS" "$AGENT47_HOME/templates/AGENTS.md"
  assert_success
}

@test "templates --list shows backups" {
  run "$ROOT_DIR/bin/a47" install
  run "$ROOT_DIR/bin/a47" install
  run "$ROOT_DIR/bin/a47" templates --list
  assert_success
  assert_contains "$output" "templates.bak."
}

@test "templates --clear-backups removes backups" {
  run "$ROOT_DIR/bin/a47" install
  run "$ROOT_DIR/bin/a47" install
  run "$ROOT_DIR/bin/a47" templates --clear-backups
  assert_success
  run "$ROOT_DIR/bin/a47" templates --list
  assert_success
  assert_contains "$output" "No backups"
}

@test "templates --restore-latest fails without backups" {
  rm -rf "$AGENT47_HOME"/templates.bak.*

  run "$ROOT_DIR/bin/a47" templates --restore-latest
  [ "$status" -ne 0 ]
  assert_contains "$output" "No template backups found"
}
