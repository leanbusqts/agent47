#!/usr/bin/env bats

load ../helpers/common

setup() {
  setup_workdir
}

teardown() {
  teardown_workdir
}

@test "add-agent-prompt creates prompt" {
  run "$ROOT_DIR/bin/afs" add-agent-prompt
  assert_success
  assert_file_exists "prompts/agent-prompt.txt"
  run cat "prompts/agent-prompt.txt"
  assert_success
  assert_contains "$output" "Use \`AGENTS.md\` as the single source of policy."
  assert_contains "$output" "skills/AVAILABLE_SKILLS.xml"
  assert_contains "$output" "specs/spec.yml"
  assert_contains "$output" "suggest that the user review it"
  assert_contains "$output" "another agent or sub-agent"
  assert_not_contains "$output" "Authoritative order:"
}

@test "add-agent-prompt --force overwrites existing prompt" {
  mkdir -p prompts
  echo "custom prompt" > prompts/agent-prompt.txt

  run "$ROOT_DIR/bin/afs" add-agent-prompt --force
  assert_success
  run grep -F "Use \`AGENTS.md\` as the single source of policy." prompts/agent-prompt.txt
  assert_success
}

@test "add-agent-prompt rejects unexpected arguments" {
  run "$ROOT_DIR/bin/afs" add-agent-prompt unexpected
  [ "$status" -ne 0 ]
  assert_contains "$output" "Usage: add-agent-prompt [--force]"
}

@test "add-agent-prompt does not create prompts dir when template is missing" {
  temp_repo="$(make_test_repo_copy)"
  run bash -c '
    set -euo pipefail
    mv "$1/templates/prompts/agent-prompt.txt" "$1/templates/prompts/agent-prompt.txt.bak"
    trap '"'"'mv "$1/templates/prompts/agent-prompt.txt.bak" "$1/templates/prompts/agent-prompt.txt"'"'"' EXIT
    cd "$2"
    AGENT47_REPO_ROOT="$1" "$3/bin/afs" add-agent-prompt
  ' _ "$temp_repo" "$PWD" "$ROOT_DIR"
  [ "$status" -ne 0 ]
  assert_contains "$output" "Template not found: agent-prompt.txt"
  [ ! -d "prompts" ]
}

@test "bin/afs delegates add-agent-prompt to configured go cli bridge" {
  cat > "$TEST_WORKDIR/fake-go-cli" <<'EOF'
#!/bin/bash
echo "prompt-go-cli:$*"
exit 0
EOF
  chmod +x "$TEST_WORKDIR/fake-go-cli"

  run env AGENT47_GO_CLI="$TEST_WORKDIR/fake-go-cli" "$ROOT_DIR/bin/afs" add-agent-prompt --force
  assert_success
  assert_contains "$output" "prompt-go-cli:add-agent-prompt --force"
}
