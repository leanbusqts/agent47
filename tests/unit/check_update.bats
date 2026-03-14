#!/usr/bin/env bats

load ../helpers/common

setup() {
  setup_workdir
}

teardown() {
  teardown_workdir
}

@test "check-update cache round-trips special characters safely" {
  export CACHE_DIR="$AGENT47_HOME/cache"
  export UPDATE_CACHE_FILE="$CACHE_DIR/update.cache"
  export UPDATE_CACHE_TTL_SECONDS=86400
  export AGENT47_VERSION="1.2.3"

  # shellcheck disable=SC1090
  source "$ROOT_DIR/scripts/lib/update.sh"

  export UPDATE_STATUS="error"
  export UPDATE_METHOD='remote "quoted"'
  export UPDATE_LOCAL_VERSION="$AGENT47_VERSION"
  export UPDATE_LATEST_VERSION="2.0.0"
  export UPDATE_MESSAGE='problem with "quotes" and \ slashes'

  save_update_cache
  UPDATE_STATUS=""
  UPDATE_METHOD=""
  UPDATE_LOCAL_VERSION=""
  UPDATE_LATEST_VERSION=""
  UPDATE_MESSAGE=""
  UPDATE_FROM_CACHE=false
  UPDATE_CACHE_AGE=0

  load_update_cache "$(date +%s)"
  [ "$?" -eq 0 ]
  [ "$UPDATE_STATUS" = "error" ]
  [ "$UPDATE_METHOD" = 'remote "quoted"' ]
  [ "$UPDATE_LOCAL_VERSION" = "1.2.3" ]
  [ "$UPDATE_LATEST_VERSION" = "2.0.0" ]
  [ "$UPDATE_MESSAGE" = 'problem with "quotes" and \ slashes' ]
}

@test "check-update warns when curl unavailable" {
  PATH="/usr/sbin:/sbin:/bin"
  AGENT47_VERSION_URL=""
  rm -f "$AGENT47_HOME/cache/update.cache"
  run "$ROOT_DIR/bin/a47" check-update --force
  assert_success
  assert_contains "$output" "Cannot check for updates"
}

@test "check-update succeeds when remote VERSION is readable" {
  export AGENT47_VERSION_URL="file://$ROOT_DIR/VERSION"
  rm -f "$AGENT47_HOME/cache/update.cache"
  run "$ROOT_DIR/bin/a47" check-update
  assert_success
  assert_contains "$output" "Up to date"
}

@test "check-update warns when git and remote both fail" {
  PATH="/usr/sbin:/sbin:/bin"
  unset AGENT47_VERSION_URL
  rm -f "$AGENT47_HOME/cache/update.cache"
  run "$ROOT_DIR/bin/a47" check-update --force
  assert_success
  assert_contains "$output" "Cannot check for updates"
}
