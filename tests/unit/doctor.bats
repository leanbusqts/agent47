#!/usr/bin/env bats

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

@test "doctor reports ok when tools are on PATH" {
  export PATH="$HOME/bin:$PATH"
  export AGENT47_VERSION_URL="file://$ROOT_DIR/VERSION"
  mkdir -p "$HOME/bin" "$AGENT47_HOME/bin"
  rm -f "$HOME/bin/afs" "$HOME/bin/add-agent" "$HOME/bin/add-agent-prompt" "$HOME/bin/add-snapshot-prompt"
  cp "$ROOT_DIR/bin/afs" "$AGENT47_HOME/bin/afs"
  chmod +x "$AGENT47_HOME/bin/afs"
  cp "$ROOT_DIR/scripts/add-agent" "$AGENT47_HOME/scripts/add-agent"
  chmod +x "$AGENT47_HOME/scripts/add-agent"
  cp "$ROOT_DIR/scripts/add-agent-prompt" "$AGENT47_HOME/scripts/add-agent-prompt"
  chmod +x "$AGENT47_HOME/scripts/add-agent-prompt"
  cp "$ROOT_DIR/scripts/add-snapshot-prompt" "$AGENT47_HOME/scripts/add-snapshot-prompt"
  chmod +x "$AGENT47_HOME/scripts/add-snapshot-prompt"
  ln -s "$AGENT47_HOME/bin/afs" "$HOME/bin/afs"
  cp "$AGENT47_HOME/scripts/add-agent" "$HOME/bin/add-agent"
  chmod +x "$HOME/bin/add-agent"
  cp "$AGENT47_HOME/scripts/add-agent-prompt" "$HOME/bin/add-agent-prompt"
  chmod +x "$HOME/bin/add-agent-prompt"
  cp "$AGENT47_HOME/scripts/add-snapshot-prompt" "$HOME/bin/add-snapshot-prompt"
  chmod +x "$HOME/bin/add-snapshot-prompt"
  run "$ROOT_DIR/bin/afs" doctor
  assert_success
  assert_contains "$output" "[OK] afs in PATH"
  assert_contains "$output" "[OK] add-agent available"
  assert_contains "$output" "[OK] add-agent-prompt available"
  assert_contains "$output" "[OK] add-snapshot-prompt available"
  assert_contains "$output" "[OK] Templates installed"
  assert_contains "$output" "[OK] Prompt template present"
  assert_contains "$output" "[OK] Security templates present"
  assert_contains "$output" "[OK] Security rule IDs unique"
  assert_contains "$output" "[OK] AGENTS required sections present"
  assert_contains "$output" "[OK] bats available"
  assert_contains "$output" "[OK] afs symlink present in ~/bin"
  assert_contains "$output" "Skipping update check by default"
}

@test "doctor detects legacy add-agent-prompt-base script" {
  mkdir -p "$HOME/bin"
  cat > "$HOME/bin/add-agent-prompt-base" <<'EOF'
#!/bin/bash
exit 0
EOF
  chmod +x "$HOME/bin/add-agent-prompt-base"
  export PATH="$HOME/bin:$ROOT_DIR/bin:$ROOT_DIR/scripts:$PATH"

  run "$ROOT_DIR/bin/afs" doctor
  assert_success
  assert_contains "$output" "Legacy script detected: add-agent-prompt-base"
}

@test "doctor runs update check only when requested" {
  export PATH="$ROOT_DIR/bin:$ROOT_DIR/scripts:$PATH"
  export AGENT47_VERSION_URL="file://$ROOT_DIR/VERSION"

  run "$ROOT_DIR/bin/afs" doctor --check-update
  assert_success
  assert_contains "$output" "Up to date"
}

@test "doctor warns when PATH contains non-managed afs or helper scripts" {
  mkdir -p "$TEST_WORKDIR/fake-bin"
  cat > "$TEST_WORKDIR/fake-bin/afs" <<'EOF'
#!/bin/bash
exit 0
EOF
  cat > "$TEST_WORKDIR/fake-bin/add-agent" <<'EOF'
#!/bin/bash
exit 0
EOF
  chmod +x "$TEST_WORKDIR/fake-bin/afs" "$TEST_WORKDIR/fake-bin/add-agent"
  export PATH="$TEST_WORKDIR/fake-bin:/usr/bin:/bin"

  run "$ROOT_DIR/bin/afs" doctor
  assert_success
  assert_contains "$output" "afs in PATH, but not the managed launcher"
  assert_contains "$output" "add-agent in PATH, but not the managed installed copy"
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
  assert_contains "$output" "afs symlink in ~/bin is broken or points to a non-executable target"
}
