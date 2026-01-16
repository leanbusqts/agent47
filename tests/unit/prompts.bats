#!/usr/bin/env bats

load ../helpers/common

setup() {
  setup_workdir
}

teardown() {
  teardown_workdir
}

@test "add-agent-prompt-base creates prompt" {
  run "$ROOT_DIR/scripts/add-agent-prompt-base"
  assert_success
  assert_file_exists "prompts/agent-prompt-base.txt"
}

@test "add-agent-prompt-skills creates prompt" {
  run "$ROOT_DIR/scripts/add-agent-prompt-skills"
  assert_success
  assert_file_exists "prompts/agent-prompt-skills.txt"
}

@test "add-agent-prompt-sdd creates prompt" {
  run "$ROOT_DIR/scripts/add-agent-prompt-sdd"
  assert_success
  assert_file_exists "prompts/agent-prompt-sdd.txt"
}
