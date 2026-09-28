#!/usr/bin/env bats

load ../helpers/common

setup() { setup_workdir; }
teardown() { teardown_workdir; }

@test "install publishes one CLI and the Lite template payload" {
  PATH="$HOME/bin:$PATH" run "$ROOT_DIR/install.sh" --force --non-interactive
  assert_success
  assert_file_exists "$AGENT47_HOME/bin/afs"
  assert_file_exists "$AGENT47_HOME/templates/base/AGENTS.md"
  assert_file_exists "$AGENT47_HOME/templates/base/rules/security-shell.yaml"
  [ -L "$HOME/bin/afs" ]
  [ ! -e "$HOME/bin/add-agent" ]
  [ ! -e "$HOME/bin/add-agent-prompt" ]
  [ ! -e "$HOME/bin/add-ss-prompt" ]
  [ ! -d "$AGENT47_HOME/templates/base/skills" ]
  [ ! -d "$AGENT47_HOME/templates/base/prompts" ]
}

@test "installed CLI exposes init and map but not removed commands" {
  PATH="$HOME/bin:$PATH" run "$ROOT_DIR/install.sh" --force --non-interactive
  assert_success
  run "$AGENT47_HOME/bin/afs" help
  assert_success
  assert_contains "$output" "afs init"
  assert_contains "$output" "afs map [--force]"
  assert_not_contains "$output" "add-agent"
}

@test "install.sh resolves the repo root through symlinks" {
  ln -s "$ROOT_DIR/install.sh" "$TEST_WORKDIR/install-link.sh"
  PATH="$HOME/bin:$PATH" run "$TEST_WORKDIR/install-link.sh" --force --non-interactive
  assert_success
  assert_file_exists "$AGENT47_HOME/bin/afs"
}

@test "install without force preserves runtime and unmanaged legacy names" {
  mkdir -p "$AGENT47_HOME/bin" "$HOME/bin"
  printf '%s\n' old-launcher > "$AGENT47_HOME/bin/afs"
  printf '%s\n' user-owned > "$HOME/bin/add-agent"
  printf '%s\n' user-afs > "$HOME/bin/afs"

  PATH="$HOME/bin:$PATH" run "$ROOT_DIR/install.sh" --non-interactive
  assert_success
  assert_contains "$output" "afs launcher already exists"
  assert_contains "$output" "Preserving unmanaged legacy helper add-agent"
  assert_contains "$output" "afs entry already exists in ~/bin"
  run cat "$HOME/bin/add-agent"
  [ "$output" = "user-owned" ]
  run cat "$HOME/bin/afs"
  [ "$output" = "user-afs" ]
}

@test "force install preserves an unmanaged legacy helper" {
  printf '%s\n' user-owned > "$HOME/bin/add-agent"
  PATH="$HOME/bin:$PATH" run "$ROOT_DIR/install.sh" --force --non-interactive
  assert_success
  run cat "$HOME/bin/add-agent"
  [ "$output" = "user-owned" ]
}

@test "force install creates a template backup" {
  PATH="$HOME/bin:$PATH" run "$ROOT_DIR/install.sh" --force --non-interactive
  assert_success
  PATH="$HOME/bin:$PATH" run "$ROOT_DIR/install.sh" --force --non-interactive
  assert_success
  run find "$AGENT47_HOME" -maxdepth 1 -type d -name 'templates.bak.*'
  assert_success
  assert_contains "$output" "templates.bak."
}

@test "installer rejects an invalid manifest contract" {
  temp_repo="$(make_test_repo_copy)"
  run bash -c 'for manifest in "$1/templates/manifest.txt" "$1/templates/base/manifest.txt"; do printf "%s\n" "[rule_templates]" "security-global.yaml" "[managed_targets]" "" > "$manifest"; done; PATH="$HOME/bin:$PATH" AGENT47_REPO_ROOT="$1" "$2/install.sh" --non-interactive' _ "$temp_repo" "$ROOT_DIR"
  [ "$status" -ne 0 ]
  assert_contains "$output" "Manifest section has no entries"
}

@test "uninstall removes managed runtime but preserves unmanaged helper" {
  PATH="$HOME/bin:$PATH" run "$ROOT_DIR/install.sh" --force --non-interactive
  assert_success
  printf '%s\n' user-owned > "$HOME/bin/add-agent"

  run "$AGENT47_HOME/bin/afs" uninstall
  assert_success
  [ ! -e "$HOME/bin/afs" ]
  [ ! -d "$AGENT47_HOME" ]
  assert_file_exists "$HOME/bin/add-agent"
}

@test "smoke install completes without doctor warnings" {
  run bash -c 'cd "$1" && GOCACHE="${GOCACHE:-/tmp/agent47-go-build-cache}" GOMODCACHE="${GOMODCACHE:-/tmp/agent47-go-mod-cache}" go run ./cmd/afssmoke' _ "$ROOT_DIR"
  assert_success
  assert_not_contains "$output" "[WARN]"
}

@test "install.sh rejects unsupported arguments" {
  run "$ROOT_DIR/install.sh" unexpected-arg
  [ "$status" -ne 0 ]
  assert_contains "$output" "Usage: ./install.sh [--force] [--non-interactive]"

  run "$ROOT_DIR/install.sh" --no-prompt
  [ "$status" -ne 0 ]
}
