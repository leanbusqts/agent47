#!/usr/bin/env bats

load ../helpers/common

setup() {
  setup_workdir
}

teardown() {
  teardown_workdir
}

@test "bin/a47 add-agent uses local scripts when not in PATH" {
  # PATH sin scripts instalados
  PATH="/usr/bin:/bin"

  run "$ROOT_DIR/bin/a47" add-agent
  assert_success
  assert_file_exists "AGENTS.md"
}

@test "bin/a47 add-agent prefers managed script over PATH shadow" {
  mkdir -p "$TEST_WORKDIR/fake-bin"
  cat > "$TEST_WORKDIR/fake-bin/add-agent" <<'EOF'
#!/bin/bash
echo injected-command
exit 0
EOF
  chmod +x "$TEST_WORKDIR/fake-bin/add-agent"

  PATH="$TEST_WORKDIR/fake-bin:/usr/bin:/bin" run "$ROOT_DIR/bin/a47" add-agent
  assert_success
  assert_not_contains "$output" "injected-command"
  assert_file_exists "AGENTS.md"
}

@test "uninstall removes installed scripts" {
  PATH="$HOME/bin:$PATH" run "$ROOT_DIR/install.sh"
  assert_success
  # Verifica instalados en ~/bin
  assert_file_exists "$HOME/bin/add-agent"

  run "$ROOT_DIR/bin/a47" uninstall
  assert_success
  [ ! -f "$HOME/bin/add-agent" ]
  [ ! -L "$HOME/bin/a47" ]
}

@test "bin/a47 exits non-zero for unknown commands" {
  run "$ROOT_DIR/bin/a47" does-not-exist
  [ "$status" -ne 0 ]
  assert_contains "$output" "Unknown command: does-not-exist"
}

@test "bin/a47 no expone install como comando publico" {
  run "$ROOT_DIR/bin/a47" install
  [ "$status" -ne 0 ]
  assert_contains "$output" "Unknown command: install"
}

@test "bin/a47 no expone upgrade como comando publico" {
  run "$ROOT_DIR/bin/a47" upgrade
  [ "$status" -ne 0 ]
  assert_contains "$output" "Unknown command: upgrade"
}

@test "bin/a47 no expone add-spec como comando publico" {
  run "$ROOT_DIR/bin/a47" add-spec
  [ "$status" -ne 0 ]
  assert_contains "$output" "Unknown command: add-spec"
}

@test "bin/a47 no expone add-cli-prompt como comando publico" {
  run "$ROOT_DIR/bin/a47" add-cli-prompt
  [ "$status" -ne 0 ]
  assert_contains "$output" "Unknown command: add-cli-prompt"
}

@test "bin/a47 no expone templates como comando publico" {
  run "$ROOT_DIR/bin/a47" templates
  [ "$status" -ne 0 ]
  assert_contains "$output" "Unknown command: templates"
}

@test "bin/a47 no expone check-update como comando publico" {
  run "$ROOT_DIR/bin/a47" check-update
  [ "$status" -ne 0 ]
  assert_contains "$output" "Unknown command: check-update"
}

@test "bin/a47 no expone add-default-skills como comando publico" {
  run "$ROOT_DIR/bin/a47" add-default-skills
  [ "$status" -ne 0 ]
  assert_contains "$output" "Unknown command: add-default-skills"
}
