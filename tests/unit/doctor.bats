#!/usr/bin/env bats

load ../helpers/common

setup() {
  setup_workdir
}

teardown() {
  teardown_workdir
}

@test "doctor reports missing a47 in PATH" {
  PATH="/usr/bin:/bin"
  run "$ROOT_DIR/bin/a47" doctor
  assert_success
  assert_contains "$output" "a47 not in PATH"
}

@test "doctor reports ok when tools are on PATH" {
  export PATH="$ROOT_DIR/bin:$ROOT_DIR/scripts:$PATH"
  export AGENT47_VERSION_URL="file://$ROOT_DIR/VERSION"
  mkdir -p "$HOME/bin"
  ln -s "$ROOT_DIR/bin/a47" "$HOME/bin/a47"
  run "$ROOT_DIR/bin/a47" doctor
  assert_success
  assert_contains "$output" "[OK] a47 in PATH"
  assert_contains "$output" "[OK] Templates installed"
  assert_contains "$output" "[OK] Prompt template present"
  assert_contains "$output" "[OK] Security templates present"
  assert_contains "$output" "[OK] Security rule IDs unique"
  assert_contains "$output" "[OK] AGENTS required sections present"
  assert_contains "$output" "[OK] bats available"
  assert_contains "$output" "[OK] a47 symlink present in ~/bin"
}

@test "doctor detects legacy add-agent-prompt-base script" {
  mkdir -p "$HOME/bin"
  cat > "$HOME/bin/add-agent-prompt-base" <<'EOF'
#!/bin/bash
exit 0
EOF
  chmod +x "$HOME/bin/add-agent-prompt-base"
  export PATH="$HOME/bin:$ROOT_DIR/bin:$ROOT_DIR/scripts:$PATH"

  run "$ROOT_DIR/bin/a47" doctor
  assert_success
  assert_contains "$output" "Legacy script detected: add-agent-prompt-base"
}
