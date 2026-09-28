#!/usr/bin/env bats
# shellcheck disable=SC2030,SC2031

load ../helpers/common

setup() {
  setup_workdir
}

teardown() {
  teardown_workdir
}

@test "doctor reports missing afs in PATH" {
  PATH="/usr/bin:/bin"
  run "$ROOT_DIR/bin/afs" doctor
  assert_success
  assert_contains "$output" "afs not in PATH"
  assert_contains "$output" "Skipping update check by default"
}

@test "doctor json keeps a versioned machine-readable envelope" {
  PATH="/usr/bin:/bin"
  run "$ROOT_DIR/bin/afs" doctor --json
  assert_success
  assert_contains "$output" '"schema_version": 1'
  assert_contains "$output" '"status": "warning"'
  assert_contains "$output" '"checks": ['
  assert_contains "$output" '"warnings": ['
}

@test "doctor reports ok when the managed CLI is on PATH" {
  export PATH="$HOME/bin:$PATH"
  export AGENT47_VERSION_URL="file://$ROOT_DIR/VERSION"
  mkdir -p "$HOME/bin" "$AGENT47_HOME/bin"
  rm -f "$HOME/bin/afs"
  cp "$ROOT_DIR/bin/afs" "$AGENT47_HOME/bin/afs"
  chmod +x "$AGENT47_HOME/bin/afs"
  ln -s "$AGENT47_HOME/bin/afs" "$HOME/bin/afs"
  run "$ROOT_DIR/bin/afs" doctor
  assert_success
  assert_contains "$output" "[OK] afs in PATH"
  assert_contains "$output" "[OK] Templates installed"
  assert_contains "$output" "[OK] Required template files present"
  assert_contains "$output" "[OK] Required template dirs present"
  assert_contains "$output" "[OK] Rule templates present"
  assert_contains "$output" "[OK] Security templates present"
  assert_contains "$output" "[OK] Security rule IDs unique"
  assert_contains "$output" "[OK] AGENTS required sections present"
  assert_contains "$output" "[OK] bats available"
  assert_contains "$output" "[OK] afs symlink present in ~/bin"
  assert_contains "$output" "Skipping update check by default"
}

@test "doctor runs update check only when requested" {
  export PATH="$ROOT_DIR/bin:$PATH"
  export AGENT47_ENABLE_TEST_HOOKS="true"
  export AGENT47_VERSION_URL="file://$ROOT_DIR/VERSION"

  run "$ROOT_DIR/bin/afs" doctor --check-update
  assert_success
  assert_contains "$output" "Up to date"
}

@test "doctor warns when PATH contains a non-managed afs" {
  mkdir -p "$TEST_WORKDIR/fake-bin"
  cat > "$TEST_WORKDIR/fake-bin/afs" <<'EOF'
#!/bin/bash
exit 0
EOF
  chmod +x "$TEST_WORKDIR/fake-bin/afs"
  export PATH="$TEST_WORKDIR/fake-bin:/usr/bin:/bin"

  run "$ROOT_DIR/bin/afs" doctor
  assert_success
  assert_contains "$output" "afs in PATH, but not the managed launcher"
}

@test "doctor warns when ~/bin afs symlink is broken" {
  mkdir -p "$HOME/bin"
  rm -f "$HOME/bin/afs"
  ln -s "$TEST_WORKDIR/missing-afs" "$HOME/bin/afs"
  export PATH="$HOME/bin:/usr/bin:/bin"

  run "$ROOT_DIR/bin/afs" doctor
  assert_success
  assert_contains "$output" "afs symlink in ~/bin is broken or points to a non-executable target"
}

@test "doctor warns when ~/bin afs points to the wrong executable" {
  mkdir -p "$HOME/bin" "$TEST_WORKDIR/wrong"
  cat > "$TEST_WORKDIR/wrong/afs" <<'EOF'
#!/bin/bash
exit 0
EOF
  chmod +x "$TEST_WORKDIR/wrong/afs"
  rm -f "$HOME/bin/afs"
  ln -s "$TEST_WORKDIR/wrong/afs" "$HOME/bin/afs"
  export PATH="$HOME/bin:/usr/bin:/bin"

  run "$ROOT_DIR/bin/afs" doctor
  assert_success
  assert_contains "$output" "afs in PATH, but not the managed launcher"
  assert_contains "$output" "afs symlink in ~/bin points to the wrong executable"
}

@test "doctor --fail-on-warn exits non-zero when warnings are present" {
  PATH="/usr/bin:/bin"
  run "$ROOT_DIR/bin/afs" doctor --fail-on-warn
  [ "$status" -ne 0 ]
  assert_contains "$output" "afs not in PATH"
  assert_contains "$output" "doctor reported warnings"
}
