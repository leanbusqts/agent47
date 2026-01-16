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
  run "$ROOT_DIR/bin/a47" doctor
  assert_success
  assert_contains "$output" "[OK] a47 in PATH"
  assert_contains "$output" "[OK] Templates installed"
}
