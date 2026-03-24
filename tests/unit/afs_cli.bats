#!/usr/bin/env bats

load ../helpers/common

setup() {
  setup_workdir
}

teardown() {
  teardown_workdir
}

@test "bin/afs add-agent uses local scripts when not in PATH" {
  # PATH sin scripts instalados
  PATH="/usr/bin:/bin"

  run "$ROOT_DIR/bin/afs" add-agent
  assert_success
  assert_file_exists "AGENTS.md"
}

@test "bin/afs add-agent prefers managed script over PATH shadow" {
  mkdir -p "$TEST_WORKDIR/fake-bin"
  cat > "$TEST_WORKDIR/fake-bin/add-agent" <<'EOF'
#!/bin/bash
echo injected-command
exit 0
EOF
  chmod +x "$TEST_WORKDIR/fake-bin/add-agent"

  PATH="$TEST_WORKDIR/fake-bin:/usr/bin:/bin" run "$ROOT_DIR/bin/afs" add-agent
  assert_success
  assert_not_contains "$output" "injected-command"
  assert_file_exists "AGENTS.md"
}

@test "uninstall removes installed scripts" {
  PATH="$HOME/bin:$PATH" run "$ROOT_DIR/install.sh"
  assert_success
  # Verifica instalados en ~/bin
  assert_file_exists "$HOME/bin/add-agent"

  run "$ROOT_DIR/bin/afs" uninstall
  assert_success
  [ ! -f "$HOME/bin/add-agent" ]
  [ ! -L "$HOME/bin/afs" ]
}

@test "bin/afs exits non-zero for unknown commands" {
  run "$ROOT_DIR/bin/afs" does-not-exist
  [ "$status" -ne 0 ]
  assert_contains "$output" "Unknown command: does-not-exist"
}

@test "bin/afs no expone install como comando publico" {
  run "$ROOT_DIR/bin/afs" install
  [ "$status" -ne 0 ]
  assert_contains "$output" "Unknown command: install"
}

@test "bin/afs no expone upgrade como comando publico" {
  run "$ROOT_DIR/bin/afs" upgrade
  [ "$status" -ne 0 ]
  assert_contains "$output" "Unknown command: upgrade"
}

@test "bin/afs no expone add-spec como comando publico" {
  run "$ROOT_DIR/bin/afs" add-spec
  [ "$status" -ne 0 ]
  assert_contains "$output" "Unknown command: add-spec"
}

@test "bin/afs no expone add-cli-prompt como comando publico" {
  run "$ROOT_DIR/bin/afs" add-cli-prompt
  [ "$status" -ne 0 ]
  assert_contains "$output" "Unknown command: add-cli-prompt"
}

@test "bin/afs no expone templates como comando publico" {
  run "$ROOT_DIR/bin/afs" templates
  [ "$status" -ne 0 ]
  assert_contains "$output" "Unknown command: templates"
}

@test "bin/afs no expone check-update como comando publico" {
  run "$ROOT_DIR/bin/afs" check-update
  [ "$status" -ne 0 ]
  assert_contains "$output" "Unknown command: check-update"
}

@test "bin/afs no expone add-default-skills como comando publico" {
  run "$ROOT_DIR/bin/afs" add-default-skills
  [ "$status" -ne 0 ]
  assert_contains "$output" "Unknown command: add-default-skills"
}

@test "bin/afs no expone init-agent como comando publico" {
  run "$ROOT_DIR/bin/afs" init-agent
  [ "$status" -ne 0 ]
  assert_contains "$output" "Unknown command: init-agent"
}
