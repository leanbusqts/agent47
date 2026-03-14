#!/usr/bin/env bats

load ../helpers/common

setup() {
  setup_workdir
}

teardown() {
  teardown_workdir
}

@test "install copies templates and skill-utils into AGENT47_HOME" {
  PATH="$HOME/bin:$PATH" run "$ROOT_DIR/install.sh" --force
  assert_success
  assert_file_exists "$AGENT47_HOME/bin/a47"
  assert_file_exists "$AGENT47_HOME/templates/AGENTS.md"
  assert_file_exists "$AGENT47_HOME/templates/specs/spec.yml"
  assert_file_exists "$AGENT47_HOME/scripts/skill-utils.sh"
  assert_file_exists "$AGENT47_HOME/scripts/add-agent-prompt"
  assert_file_exists "$AGENT47_HOME/scripts/add-snapshot-prompt"
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

  run "$ROOT_DIR/bin/a47" uninstall
  assert_success
  [ ! -f "$HOME/bin/add-agent" ]
  [ ! -f "$HOME/bin/add-agent-prompt" ]
  [ ! -f "$HOME/bin/add-snapshot-prompt" ]
  [ ! -L "$HOME/bin/a47" ]
}

@test "install.sh rejects unexpected arguments" {
  run "$ROOT_DIR/install.sh" unexpected-arg
  [ "$status" -ne 0 ]
  assert_contains "$output" "Usage: ./install.sh [--force] [--no-prompt]"
}

@test "install succeeds without tty when ~/bin is not in PATH" {
  run bash -c "PATH=/usr/bin:/bin \"$ROOT_DIR/install.sh\" --no-prompt </dev/null"
  assert_success
  assert_contains "$output" "Non-interactive install; skipping shell rc update"
  assert_file_exists "$AGENT47_HOME/bin/a47"
}

@test "install fails fast when launcher cannot be written" {
  mkdir -p "$AGENT47_HOME/bin"
  chmod 500 "$AGENT47_HOME/bin"

  PATH="$HOME/bin:$PATH" run "$ROOT_DIR/install.sh"
  [ "$status" -ne 0 ]
  assert_not_contains "$output" "[OK] Installed a47 launcher"

  chmod 700 "$AGENT47_HOME/bin"
}

@test "install fails fast when a core install asset is missing" {
  mv "$ROOT_DIR/templates/AGENTS.md" "$ROOT_DIR/templates/AGENTS.md.bak"

  PATH="$HOME/bin:$PATH" run "$ROOT_DIR/install.sh"
  [ "$status" -ne 0 ]
  assert_contains "$output" "Required install asset missing"

  mv "$ROOT_DIR/templates/AGENTS.md.bak" "$ROOT_DIR/templates/AGENTS.md"
}

@test "install rolls back public scripts if publish fails mid-flight" {
  mkdir -p "$TEST_WORKDIR/fake-bin"
  cat > "$TEST_WORKDIR/fake-bin/mv" <<EOF
#!/bin/bash
if [ "\${!#}" = "$HOME/bin/add-agent-prompt" ]; then
  exit 1
fi
exec /bin/mv "\$@"
EOF
  chmod +x "$TEST_WORKDIR/fake-bin/mv"

  printf '%s\n' old-add-agent > "$HOME/bin/add-agent"
  printf '%s\n' old-add-agent-prompt > "$HOME/bin/add-agent-prompt"
  rm -f "$HOME/bin/a47"

  PATH="$TEST_WORKDIR/fake-bin:$HOME/bin:/usr/bin:/bin" run "$ROOT_DIR/install.sh" --force --no-prompt
  [ "$status" -ne 0 ]

  run cat "$HOME/bin/add-agent"
  assert_success
  [ "$output" = "old-add-agent" ]

  run cat "$HOME/bin/add-agent-prompt"
  assert_success
  [ "$output" = "old-add-agent-prompt" ]

  [ ! -L "$HOME/bin/a47" ]
}

@test "install preserves existing a47 symlink if link swap fails" {
  mkdir -p "$TEST_WORKDIR/fake-bin" "$TEST_WORKDIR/old"
  cat > "$TEST_WORKDIR/fake-bin/mv" <<EOF
#!/bin/bash
if [ "\${!#}" = "$HOME/bin/a47" ]; then
  exit 1
fi
exec /bin/mv "\$@"
EOF
  chmod +x "$TEST_WORKDIR/fake-bin/mv"

  touch "$TEST_WORKDIR/old/a47"
  rm -f "$HOME/bin/a47"
  ln -s "$TEST_WORKDIR/old/a47" "$HOME/bin/a47"

  PATH="$TEST_WORKDIR/fake-bin:$HOME/bin:/usr/bin:/bin" run "$ROOT_DIR/install.sh" --force --no-prompt
  [ "$status" -ne 0 ]
  [ -L "$HOME/bin/a47" ]
  run readlink "$HOME/bin/a47"
  assert_success
  [ "$output" = "$TEST_WORKDIR/old/a47" ]
}

@test "install restores previous templates directory if forced swap fails" {
  temp_home="$TEST_WORKDIR/install-home"
  temp_agent47_home="$temp_home/.agent47"
  fail_marker="$TEST_WORKDIR/fail-dir-swap-once"

  mkdir -p "$temp_agent47_home/templates"
  echo "old template" > "$temp_agent47_home/templates/AGENTS.md"

  run bash -c "HOME=\"$temp_home\" AGENT47_HOME=\"$temp_agent47_home\" AGENT47_ENABLE_TEST_HOOKS=true AGENT47_FAIL_DIR_SWAP_TARGET=\"$temp_agent47_home/templates\" AGENT47_FAIL_DIR_SWAP_MARKER=\"$fail_marker\" PATH=\"\$HOME/bin:/usr/bin:/bin\" \"$ROOT_DIR/install.sh\" --force --no-prompt"
  [ "$status" -ne 0 ]
  run cat "$temp_agent47_home/templates/AGENTS.md"
  assert_success
  [ "$output" = "old template" ]
}
