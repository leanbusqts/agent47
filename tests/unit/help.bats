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
  assert_not_contains "$output" "a47 install [--force]"
  assert_not_contains "$output" "a47 upgrade [--force]"
  assert_not_contains "$output" "a47 add-spec"
  assert_not_contains "$output" "a47 check-update"
  assert_not_contains "$output" "a47 templates"
  assert_not_contains "$output" "a47 add-default-skills"
  assert_contains "$output" "a47 add-agent                 bootstrap project scaffolding"
  assert_contains "$output" "a47 add-agent --only-skills   install only skills"
  assert_contains "$output" "a47 add-agent-prompt [--force]"
}
