#!/usr/bin/env bats

load ../helpers/common

setup() {
  setup_workdir
}

teardown() {
  teardown_workdir
}

@test "install copies templates and managed launcher into AGENT47_HOME" {
  PATH="$HOME/bin:$PATH" run "$ROOT_DIR/install.sh" --force
  assert_success
  assert_file_exists "$AGENT47_HOME/bin/afs"
  assert_file_exists "$AGENT47_HOME/templates/AGENTS.md"
  assert_file_exists "$AGENT47_HOME/templates/specs/spec.yml"
  assert_file_exists "$AGENT47_HOME/templates/rules/security-shell.yaml"
  [ ! -d "$AGENT47_HOME/scripts/lib" ]
  [ ! -e "$AGENT47_HOME/scripts/add-agent" ]
  [ ! -e "$AGENT47_HOME/scripts/add-agent-prompt" ]
  [ ! -e "$AGENT47_HOME/scripts/add-ss-prompt" ]
}

@test "install.sh uses the native Go installer by default" {
  if ! command -v go >/dev/null 2>&1; then
    skip "go not installed"
  fi

  PATH="$HOME/bin:$PATH" run "$ROOT_DIR/install.sh" --force --non-interactive
  assert_success
  assert_file_exists "$AGENT47_HOME/bin/afs"
  assert_file_exists "$HOME/bin/add-agent"
  assert_file_exists "$HOME/bin/add-agent-prompt"
  [ -L "$HOME/bin/afs" ]

  run "$AGENT47_HOME/bin/afs" help
  assert_success
  assert_contains "$output" "agent47 Agent CLI"
}

@test "install.sh resolves the repo root through symlinks" {
  if ! command -v go >/dev/null 2>&1; then
    skip "go not installed"
  fi

  ln -s "$ROOT_DIR/install.sh" "$TEST_WORKDIR/install-link.sh"
  PATH="$HOME/bin:$PATH" run "$TEST_WORKDIR/install-link.sh" --force --non-interactive
  assert_success
  assert_file_exists "$AGENT47_HOME/bin/afs"
  run "$AGENT47_HOME/bin/afs" help
  assert_success
  assert_contains "$output" "agent47 Agent CLI"
}

@test "default go install preserves existing runtime files without force" {
  if ! command -v go >/dev/null 2>&1; then
    skip "go not installed"
  fi

  mkdir -p "$AGENT47_HOME/bin" "$HOME/bin"
  printf '%s\n' old-launcher > "$AGENT47_HOME/bin/afs"
  printf '%s\n' old-user-helper > "$HOME/bin/add-agent"
  touch "$TEST_WORKDIR/old-afs"
  rm -f "$HOME/bin/afs"
  ln -s "$TEST_WORKDIR/old-afs" "$HOME/bin/afs"

  PATH="$HOME/bin:$PATH" run "$ROOT_DIR/install.sh" --non-interactive
  assert_success
  assert_contains "$output" "afs launcher already exists"
  assert_contains "$output" "add-agent already exists in "
  assert_contains "$output" "afs entry already exists in ~/bin"

  assert_file_exists "$AGENT47_HOME/bin/afs"
  assert_file_exists "$HOME/bin/add-agent"
  run readlink "$HOME/bin/afs"
  assert_success
  [ "$output" = "$TEST_WORKDIR/old-afs" ]
}

@test "default go install rolls back public scripts if publish fails mid-flight" {
  if ! command -v go >/dev/null 2>&1; then
    skip "go not installed"
  fi

  canonical_home="$(cd "$HOME" && pwd)"
  printf '%s\n' old-add-agent > "$HOME/bin/add-agent"
  printf '%s\n' old-add-agent-prompt > "$HOME/bin/add-agent-prompt"
  rm -f "$HOME/bin/afs"

  run env \
    AGENT47_ENABLE_TEST_HOOKS=true \
    AGENT47_FAIL_SYMLINK_TARGET="$canonical_home/bin/add-agent-prompt" \
    PATH="$HOME/bin:$PATH" \
    "$ROOT_DIR/install.sh" --force --non-interactive
  [ "$status" -ne 0 ]

  assert_file_exists "$HOME/bin/add-agent"
  assert_file_exists "$HOME/bin/add-agent-prompt"

  [ ! -L "$HOME/bin/afs" ]
}

@test "default go install preserves existing afs symlink if swap fails" {
  if ! command -v go >/dev/null 2>&1; then
    skip "go not installed"
  fi

  canonical_home="$(cd "$HOME" && pwd)"
  mkdir -p "$TEST_WORKDIR/old"
  touch "$TEST_WORKDIR/old/afs"
  rm -f "$HOME/bin/afs"
  ln -s "$TEST_WORKDIR/old/afs" "$HOME/bin/afs"

  run env \
    AGENT47_ENABLE_TEST_HOOKS=true \
    AGENT47_FAIL_SYMLINK_TARGET="$canonical_home/bin/afs" \
    PATH="$HOME/bin:$PATH" \
    "$ROOT_DIR/install.sh" --force --non-interactive
  [ "$status" -ne 0 ]
  [ -L "$HOME/bin/afs" ]

  run readlink "$HOME/bin/afs"
  assert_success
  [ "$output" = "$TEST_WORKDIR/old/afs" ]
}

@test "install without force preserves existing installed runtime files" {
  mkdir -p "$AGENT47_HOME/bin" "$HOME/bin"
  printf '%s\n' old-launcher > "$AGENT47_HOME/bin/afs"
  printf '%s\n' old-user-helper > "$HOME/bin/add-agent"
  touch "$TEST_WORKDIR/old-afs"
  rm -f "$HOME/bin/afs"
  ln -s "$TEST_WORKDIR/old-afs" "$HOME/bin/afs"

  PATH="$HOME/bin:$PATH" run "$ROOT_DIR/install.sh" --non-interactive
  assert_success
  assert_contains "$output" "afs launcher already exists"
  assert_contains "$output" "add-agent already exists in "
  assert_contains "$output" "afs entry already exists in ~/bin"

  assert_file_exists "$AGENT47_HOME/bin/afs"
  assert_file_exists "$HOME/bin/add-agent"
  run readlink "$HOME/bin/afs"
  assert_success
  [ "$output" = "$TEST_WORKDIR/old-afs" ]
}

@test "install without force preserves an existing regular afs file in ~/bin" {
  run bash -c 'mkdir -p "${HOME}/bin"; rm -f "${HOME}/bin/afs"; printf "%s\n" user-owned-afs > "${HOME}/bin/afs"; PATH="${HOME}/bin:$PATH" "$1/install.sh" --non-interactive' _ "$ROOT_DIR"
  assert_success
  assert_contains "$output" "afs entry already exists in ~/bin"

  run cat "$HOME/bin/afs"
  assert_success
  [ "$output" = "user-owned-afs" ]
}

@test "smoke install completes without doctor warnings" {
  run bash -c 'cd "$1" && GOCACHE="${GOCACHE:-/tmp/agent47-go-build-cache}" go run ./cmd/afssmoke' _ "$ROOT_DIR"
  assert_success
  assert_not_contains "$output" "[WARN]"
}

@test "install fails if manifest contract drops managed or preserved targets" {
  temp_repo="$(make_test_repo_copy)"
  run bash -c 'cat > "$1/templates/manifest.txt" <<'"'"'EOF'"'"'
[rule_templates]
rules-mobile.yaml

[managed_targets]

[preserved_targets]
README.md

[required_template_files]
AGENTS.md
manifest.txt
prompts/agent-prompt.txt
prompts/ss-prompt.txt
specs/spec.yml

[required_template_dirs]
prompts
rules
skills
specs
EOF
PATH="$HOME/bin:$PATH" AGENT47_REPO_ROOT="$1" "$2/install.sh" --non-interactive
' _ "$temp_repo" "$ROOT_DIR"
  [ "$status" -ne 0 ]
  assert_contains "$output" "Manifest section has no entries: managed_targets"
}

@test "install creates a backup when overwriting templates" {
  PATH="$HOME/bin:$PATH" run "$ROOT_DIR/install.sh" --force
  assert_success

  # Segunda instalación genera backup rotativo
  PATH="$HOME/bin:$PATH" run "$ROOT_DIR/install.sh" --force
  assert_success
  run find "$AGENT47_HOME" -maxdepth 1 -type d -name "templates.bak.*"
  assert_success
  assert_contains "$output" "templates.bak."
}

@test "uninstall removes scripts from ~/bin" {
  PATH="$HOME/bin:$PATH" run "$ROOT_DIR/install.sh"
  assert_success
  mkdir -p "$AGENT47_HOME/cache"
  echo "cached" > "$AGENT47_HOME/cache/update.cache"

  run "$ROOT_DIR/bin/afs" uninstall
  assert_success
  [ ! -f "$HOME/bin/add-agent" ]
  [ ! -f "$HOME/bin/add-agent-prompt" ]
  [ ! -f "$HOME/bin/add-ss-prompt" ]
  [ ! -L "$HOME/bin/afs" ]
  [ ! -d "$AGENT47_HOME/templates" ]
  [ ! -d "$AGENT47_HOME/scripts" ]
  [ ! -f "$AGENT47_HOME/VERSION" ]
  [ ! -d "$AGENT47_HOME/cache" ]
}

@test "uninstall removes managed template backups created by force install" {
  PATH="$HOME/bin:$PATH" run "$ROOT_DIR/install.sh" --force
  assert_success
  PATH="$HOME/bin:$PATH" run "$ROOT_DIR/install.sh" --force
  assert_success
  run find "$AGENT47_HOME" -maxdepth 1 -type d -name "templates.bak.*"
  assert_success
  assert_contains "$output" "templates.bak."

  run "$ROOT_DIR/bin/afs" uninstall
  assert_success
  run find "$HOME" -maxdepth 3 -path "$AGENT47_HOME*" -print
  assert_success
  [ -z "$output" ]
}

@test "install runtime prefers bash_profile when present" {
  mkdir -p "$HOME"
  touch "$HOME/.bash_profile"
  canonical_home="$(cd "$HOME" && pwd)"

  run bash -c "SHELL=/bin/bash PATH=/usr/bin:/bin \"$ROOT_DIR/install.sh\" --non-interactive </dev/null"
  assert_success
  assert_contains "$output" "$canonical_home/.bash_profile"
}

@test "install.sh rejects unexpected arguments" {
  run "$ROOT_DIR/install.sh" unexpected-arg
  [ "$status" -ne 0 ]
  assert_contains "$output" "Usage: ./install.sh [--force] [--non-interactive]"
}

@test "install.sh rejects legacy no-prompt flag" {
  run "$ROOT_DIR/install.sh" --no-prompt
  [ "$status" -ne 0 ]
  assert_contains "$output" "Usage: ./install.sh [--force] [--non-interactive]"
}

@test "install succeeds without tty when ~/bin is not in PATH" {
  run bash -c "PATH=/usr/bin:/bin \"$ROOT_DIR/install.sh\" --non-interactive </dev/null"
  assert_success
  assert_contains "$output" "Non-interactive install; skipping shell rc update"
  assert_file_exists "$AGENT47_HOME/bin/afs"
}

@test "install --force fails fast when launcher cannot be written" {
  mkdir -p "$AGENT47_HOME/bin"
  rm -f "$AGENT47_HOME/bin/afs"
  chmod 500 "$AGENT47_HOME/bin"

  PATH="$HOME/bin:$PATH" run "$ROOT_DIR/install.sh" --force
  [ "$status" -ne 0 ]
  assert_not_contains "$output" "[OK] Installed afs launcher"

  chmod 700 "$AGENT47_HOME/bin"
}

@test "install fails fast when a core install asset is missing" {
  temp_repo="$(make_test_repo_copy)"
  mv "$temp_repo/templates/AGENTS.md" "$temp_repo/templates/AGENTS.md.bak"

  PATH="$HOME/bin:$PATH" run env AGENT47_REPO_ROOT="$temp_repo" "$ROOT_DIR/install.sh"
  [ "$status" -ne 0 ]
  assert_contains "$output" "Required install asset missing"
}

@test "install restores previous templates directory if forced swap fails" {
  temp_home="$TEST_WORKDIR/install-home"
  temp_agent47_home="$temp_home/.agent47"
  mkdir -p "$temp_home"
  canonical_temp_home="$(cd "$temp_home" && pwd)"
  canonical_temp_agent47_home="$canonical_temp_home/.agent47"
  fail_marker="$TEST_WORKDIR/fail-dir-swap-once"

  mkdir -p "$temp_agent47_home/templates"
  echo "old template" > "$temp_agent47_home/templates/AGENTS.md"

  run bash -c "HOME=\"$temp_home\" AGENT47_HOME=\"$temp_agent47_home\" AGENT47_ENABLE_TEST_HOOKS=true AGENT47_FAIL_DIR_SWAP_TARGET=\"$canonical_temp_agent47_home/templates\" AGENT47_FAIL_DIR_SWAP_MARKER=\"$fail_marker\" PATH=\"\$HOME/bin:/usr/bin:/bin\" \"$ROOT_DIR/install.sh\" --force --non-interactive"
  [ "$status" -ne 0 ]
  run cat "$temp_agent47_home/templates/AGENTS.md"
  assert_success
  [ "$output" = "old template" ]
}

@test "install preserves existing launcher when forced template refresh fails early" {
  temp_home="$TEST_WORKDIR/install-home"
  temp_agent47_home="$temp_home/.agent47"
  mkdir -p "$temp_home"
  canonical_temp_home="$(cd "$temp_home" && pwd)"
  canonical_temp_agent47_home="$canonical_temp_home/.agent47"
  fail_marker="$TEST_WORKDIR/fail-dir-swap-once"

  mkdir -p "$temp_agent47_home/bin" "$temp_agent47_home/templates"
  printf '%s\n' old-launcher > "$temp_agent47_home/bin/afs"
  echo "old template" > "$temp_agent47_home/templates/AGENTS.md"

  run bash -c "HOME=\"$temp_home\" AGENT47_HOME=\"$temp_agent47_home\" AGENT47_ENABLE_TEST_HOOKS=true AGENT47_FAIL_DIR_SWAP_TARGET=\"$canonical_temp_agent47_home/templates\" AGENT47_FAIL_DIR_SWAP_MARKER=\"$fail_marker\" PATH=\"\$HOME/bin:/usr/bin:/bin\" \"$ROOT_DIR/install.sh\" --force --non-interactive"
  [ "$status" -ne 0 ]
  run cat "$temp_agent47_home/bin/afs"
  assert_success
  [ "$output" = "old-launcher" ]
}
