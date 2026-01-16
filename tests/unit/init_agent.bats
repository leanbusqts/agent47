#!/usr/bin/env bats

load ../helpers/common

setup() {
  setup_workdir
}

teardown() {
  teardown_workdir
}

@test "init_agent uses fallback and copies core files" {
  PATH="/usr/bin:/bin"
  run "$ROOT_DIR/bin/a47" init-agent
  echo "$output"
  assert_success
  assert_file_exists "AGENTS.md"
  assert_file_exists "rules-backend.yaml"
}
