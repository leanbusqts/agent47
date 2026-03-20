#!/usr/bin/env bats

load ../helpers/common

setup() {
  setup_workdir
}

teardown() {
  teardown_workdir
}

@test "afs help prints core commands" {
  run afs help
  assert_success
  assert_contains "$output" "Core commands:"
  assert_contains "$output" "afs help"
  assert_not_contains "$output" "afs install [--force]"
  assert_not_contains "$output" "afs upgrade [--force]"
  assert_not_contains "$output" "afs add-spec"
  assert_not_contains "$output" "afs check-update"
  assert_not_contains "$output" "afs templates"
  assert_not_contains "$output" "afs add-default-skills"
  assert_contains "$output" "afs add-agent                 bootstrap project scaffolding"
  assert_contains "$output" "afs add-agent --only-skills   install only skills"
  assert_contains "$output" "afs add-agent-prompt [--force]"
}
