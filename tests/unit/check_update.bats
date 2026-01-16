#!/usr/bin/env bats

load ../helpers/common

setup() {
  setup_workdir
}

teardown() {
  teardown_workdir
}

@test "check-update warns when curl unavailable" {
  PATH="/usr/sbin:/sbin:/bin"
  AGENT47_VERSION_URL=""
  rm -f "$AGENT47_HOME/cache/update.json"
  run "$ROOT_DIR/bin/a47" check-update --force
  assert_success
  assert_contains "$output" "Cannot check for updates"
}

@test "check-update succeeds when remote VERSION is readable" {
  export AGENT47_VERSION_URL="file://$ROOT_DIR/VERSION"
  rm -f "$AGENT47_HOME/cache/update.json"
  run "$ROOT_DIR/bin/a47" check-update
  assert_success
  assert_contains "$output" "Up to date"
}

@test "check-update warns when git and remote both fail" {
  PATH="/usr/sbin:/sbin:/bin"
  unset AGENT47_VERSION_URL
  rm -f "$AGENT47_HOME/cache/update.json"
  run "$ROOT_DIR/bin/a47" check-update --force
  assert_success
  assert_contains "$output" "Cannot check for updates"
}
