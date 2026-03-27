#!/usr/bin/env bats
# shellcheck disable=SC2030,SC2031

load ../helpers/common

setup() {
  setup_workdir
}

teardown() {
  teardown_workdir
}

@test "doctor update check warns when curl unavailable" {
  PATH="/usr/sbin:/sbin:/bin"
  export AGENT47_VERSION_URL=""
  rm -f "$AGENT47_HOME/cache/update.cache"
  run "$ROOT_DIR/bin/afs" doctor --check-update-force
  assert_success
  assert_contains "$output" "Cannot check for updates"
}

@test "doctor update check succeeds when remote VERSION is readable" {
  export AGENT47_VERSION_URL="file://$ROOT_DIR/VERSION"
  rm -f "$AGENT47_HOME/cache/update.cache"
  run "$ROOT_DIR/bin/afs" doctor --check-update
  assert_success
  assert_contains "$output" "Up to date"
}

@test "doctor update check warns when git and remote both fail" {
  PATH="/usr/sbin:/sbin:/bin"
  unset AGENT47_VERSION_URL
  rm -f "$AGENT47_HOME/cache/update.cache"
  run "$ROOT_DIR/bin/afs" doctor --check-update-force
  assert_success
  assert_contains "$output" "Cannot check for updates"
}

@test "doctor update check ignores corrupted cache and falls back cleanly" {
  mkdir -p "$AGENT47_HOME/cache"
  cat > "$AGENT47_HOME/cache/update.cache" <<'EOF'
checked_at=123
status_b64=%%%
method_b64=%%%
local_b64=%%%
latest_b64=%%%
message_b64=%%%
EOF
  export AGENT47_VERSION_URL="file://$ROOT_DIR/VERSION"

  run "$ROOT_DIR/bin/afs" doctor --check-update
  assert_success
  assert_contains "$output" "Up to date"
  assert_not_contains "$output" "Using cached update check"
}
