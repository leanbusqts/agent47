#!/usr/bin/env bats

load ../helpers/common

setup() {
  setup_workdir
}

teardown() {
  teardown_workdir
}

@test "add-prompt creates prompt" {
  run "$ROOT_DIR/scripts/add-prompt"
  assert_success
  assert_file_exists "prompts/agent-prompt.txt"
  run cat "prompts/agent-prompt.txt"
  assert_success
  assert_contains "$output" 'Use `AGENTS.md` as the single source of policy.'
  assert_contains "$output" "skills/AVAILABLE_SKILLS.xml"
  assert_contains "$output" "specs/spec.yml"
  assert_contains "$output" 'If `SNAPSHOT.md` exists and the task made it stale, explicitly suggest updating it.'
  assert_not_contains "$output" "Authoritative order:"
}

@test "add-prompt --force overwrites existing prompt" {
  mkdir -p prompts
  echo "custom prompt" > prompts/agent-prompt.txt

  run "$ROOT_DIR/scripts/add-prompt" --force
  assert_success
  run grep -F 'Use `AGENTS.md` as the single source of policy.' prompts/agent-prompt.txt
  assert_success
}
