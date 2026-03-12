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
  assert_contains "$output" 'update `SNAPSHOT.md`'
}
