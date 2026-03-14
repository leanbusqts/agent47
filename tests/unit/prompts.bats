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
  assert_contains "$output" "suggest that the user review it"
  assert_contains "$output" "another agent or sub-agent"
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

@test "add-agent-prompt rejects unexpected arguments" {
  run "$ROOT_DIR/scripts/add-agent-prompt" unexpected
  [ "$status" -ne 0 ]
  assert_contains "$output" "Usage: add-agent-prompt [--force]"
}

@test "add-agent-prompt does not create prompts dir when template is missing" {
  rm -f "$AGENT47_HOME/templates/prompts/agent-prompt.txt"

  run "$ROOT_DIR/scripts/add-agent-prompt"
  [ "$status" -ne 0 ]
  assert_contains "$output" "Template not found: agent-prompt.txt"
  [ ! -d "prompts" ]

  cp "$ROOT_DIR/templates/prompts/agent-prompt.txt" "$AGENT47_HOME/templates/prompts/agent-prompt.txt"
}

@test "add-agent-prompt --force preserves existing file if replace fails" {
  mkdir -p prompts "$TEST_WORKDIR/fake-bin"
  echo "custom prompt" > prompts/agent-prompt.txt

  cat > "$TEST_WORKDIR/fake-bin/mv" <<'EOF'
#!/bin/bash
if [ "${!#}" = "prompts/agent-prompt.txt" ]; then
  exit 1
fi
exec /bin/mv "$@"
EOF
  chmod +x "$TEST_WORKDIR/fake-bin/mv"

  PATH="$TEST_WORKDIR/fake-bin:/usr/bin:/bin" run "$ROOT_DIR/scripts/add-agent-prompt" --force
  [ "$status" -ne 0 ]
  run cat prompts/agent-prompt.txt
  assert_success
  [ "$output" = "custom prompt" ]
}
