#!/usr/bin/env bats

load ../helpers/common

setup() {
  setup_workdir
}

teardown() {
  teardown_workdir
}

@test "add-agent-prompt creates prompt" {
  run "$ROOT_DIR/scripts/add-agent-prompt"
  assert_success
  assert_file_exists "prompts/agent-prompt.txt"
  run cat "prompts/agent-prompt.txt"
  assert_success
  assert_contains "$output" 'Use `AGENTS.md` as the single source of policy.'
  assert_contains "$output" "skills/AVAILABLE_SKILLS.xml"
  assert_contains "$output" "specs/spec.yml"
  assert_not_contains "$output" "Authoritative order:"
}

@test "add-agent-prompt --force overwrites existing prompt" {
  mkdir -p prompts
  echo "custom prompt" > prompts/agent-prompt.txt

  run "$ROOT_DIR/scripts/add-agent-prompt" --force
  assert_success
  run grep -F 'Use `AGENTS.md` as the single source of policy.' prompts/agent-prompt.txt
  assert_success
}

@test "add-cli-prompt prints minimal prompt when clipboard tool is unavailable" {
  export AGENT47_HOME
  PATH="/usr/bin:/bin"
  run "$ROOT_DIR/scripts/add-cli-prompt"
  assert_success
  [ "$output" = "Read AGENTS.md first and follow the applicable rules before making changes." ]
}

@test "add-agent-prompt rejects unexpected arguments" {
  run "$ROOT_DIR/scripts/add-agent-prompt" unexpected
  [ "$status" -ne 0 ]
  assert_contains "$output" "Usage: add-agent-prompt [--force]"
}

@test "add-cli-prompt rejects unexpected arguments" {
  run "$ROOT_DIR/scripts/add-cli-prompt" unexpected
  [ "$status" -ne 0 ]
  assert_contains "$output" "Usage: add-cli-prompt"
}
