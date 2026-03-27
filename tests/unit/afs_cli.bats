#!/usr/bin/env bats
# shellcheck disable=SC2030,SC2031

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

@test "bin/afs delegates to configured go cli bridge when present" {
  cat > "$TEST_WORKDIR/fake-go-cli" <<'EOF'
#!/bin/bash
echo "go-cli-bridge:$*"
exit 0
EOF
  chmod +x "$TEST_WORKDIR/fake-go-cli"

  run env AGENT47_GO_CLI="$TEST_WORKDIR/fake-go-cli" "$ROOT_DIR/bin/afs" help
  assert_success
  assert_contains "$output" "go-cli-bridge:help"
}

@test "bin/afs ignores self-referential AGENT47_GO_CLI" {
  run env AGENT47_GO_CLI="$ROOT_DIR/bin/afs" "$ROOT_DIR/bin/afs" help
  assert_success
  assert_contains "$output" "agent47 Agent CLI"
}

@test "bin/afs fails fast when explicit AGENT47_GO_CLI path is missing" {
  run env AGENT47_GO_CLI="$TEST_WORKDIR/missing-go-cli" "$ROOT_DIR/bin/afs" help
  [ "$status" -ne 0 ]
  assert_contains "$output" "AGENT47_GO_CLI points to a missing path"
}

@test "bin/afs uses configured repo cli when Go is unavailable" {
  cat > "$TEST_WORKDIR/fake-repo-cli" <<'EOF'
#!/bin/bash
echo "repo-cli:$*"
exit 0
EOF
  chmod +x "$TEST_WORKDIR/fake-repo-cli"

  run env PATH="/usr/bin:/bin" AGENT47_REPO_CLI="$TEST_WORKDIR/fake-repo-cli" "$ROOT_DIR/bin/afs" help
  assert_success
  assert_contains "$output" "repo-cli:help"
}

@test "bin/afs fails fast when explicit AGENT47_REPO_CLI is not executable" {
  printf '%s\n' '#!/bin/bash' > "$TEST_WORKDIR/not-executable-repo-cli"
  run env PATH="/usr/bin:/bin" AGENT47_REPO_CLI="$TEST_WORKDIR/not-executable-repo-cli" "$ROOT_DIR/bin/afs" help
  [ "$status" -ne 0 ]
  assert_contains "$output" "AGENT47_REPO_CLI is not executable"
}

@test "bin/afs does not use implicit repo binary fallback when Go is unavailable" {
  temp_repo="$(mktemp -d "$TEST_TMP_ROOT/launcher-XXXXXX")"
  mkdir -p "$temp_repo/bin"
  cp "$ROOT_DIR/bin/afs" "$temp_repo/bin/afs"
  chmod +x "$temp_repo/bin/afs"
  cat > "$temp_repo/afs" <<'EOF'
#!/bin/bash
echo implicit-fallback
exit 0
EOF
  chmod +x "$temp_repo/afs"

  run "$temp_repo/bin/afs" help
  [ "$status" -ne 0 ]
  assert_not_contains "$output" "implicit-fallback"
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
