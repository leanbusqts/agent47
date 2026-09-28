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
  assert_contains "$output" "afs version"
  assert_not_contains "$output" "afs install [--force]"
  assert_not_contains "$output" "afs upgrade [--force]"
  assert_not_contains "$output" "afs add-spec"
  assert_not_contains "$output" "afs check-update"
  assert_not_contains "$output" "afs templates"
  assert_contains "$output" "afs analyze [--json] [--verbose] [--evidence] [--deep]"
  assert_contains "$output" "afs map [--force]"
  assert_contains "$output" "afs doctor [--json]"
  assert_contains "$output" "afs init [--bundle <name> ...] [--exclude-bundle <name> ...]"
  assert_contains "$output" "[--force] [--preview]"
  assert_not_contains "$output" "--yes"
  assert_not_contains "$output" "add-agent"
  assert_not_contains "$output" "--dry-run"
}
