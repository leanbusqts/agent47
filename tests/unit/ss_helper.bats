#!/usr/bin/env bats

load ../helpers/common

setup() {
  setup_workdir
}

teardown() {
  teardown_workdir
}

@test "add-ss-prompt prints snapshot/spec prompt when clipboard tool is unavailable" {
  export AGENT47_HOME
  temp_path="$TEST_WORKDIR/no-clipboard-bin"
  mkdir -p "$temp_path"
  ln -s /bin/bash "$temp_path/bash"
  PATH="$temp_path:/bin"
  run "$ROOT_DIR/bin/afs" add-ss-prompt
  assert_success
  assert_contains "$output" 'SNAPSHOT.md'
  assert_contains "$output" 'SPEC.md'
}

@test "add-ss-prompt rejects unexpected arguments" {
  run "$ROOT_DIR/bin/afs" add-ss-prompt unexpected
  [ "$status" -ne 0 ]
  assert_contains "$output" "Usage: add-ss-prompt"
}
