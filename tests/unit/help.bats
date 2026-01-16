#!/usr/bin/env bats

load ../helpers/common

setup() {
  setup_workdir
}

teardown() {
  teardown_workdir
}

@test "a47 help prints core commands" {
  run a47 help
  assert_success
  assert_contains "$output" "Core commands:"
  assert_contains "$output" "a47 help"
}
