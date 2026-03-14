#!/usr/bin/env bats

load ../helpers/common

setup() {
  setup_workdir
}

teardown() {
  teardown_workdir
}

@test "add-snapshot-prompt prints prompt when clipboard tool is unavailable" {
  export AGENT47_HOME
  PATH="/usr/bin:/bin"
  run "$ROOT_DIR/scripts/add-snapshot-prompt"
  assert_success
  assert_contains "$output" 'snapshot or summary file'
}

@test "add-snapshot-prompt rejects unexpected arguments" {
  run "$ROOT_DIR/scripts/add-snapshot-prompt" unexpected
  [ "$status" -ne 0 ]
  assert_contains "$output" "Usage: add-snapshot-prompt"
}
