#!/usr/bin/env bash

setup_workdir() {
  TEST_WORKDIR="$(mktemp -d "$TEST_TMP_ROOT/work-XXXXXX")"
  cd "$TEST_WORKDIR"
}

teardown_workdir() {
  if [ -n "${TEST_WORKDIR:-}" ] && [ -d "$TEST_WORKDIR" ]; then
    rm -rf "$TEST_WORKDIR"
  fi
  cd "$ROOT_DIR"
}

assert_success() {
  if [ "$status" -ne 0 ]; then
    echo "Command failed with status $status"
    echo "$output"
    return 1
  fi
}

assert_contains() {
  local haystack="$1"
  local needle="$2"
  if [[ "$haystack" != *"$needle"* ]]; then
    echo "Expected output to contain: $needle"
    echo "$haystack"
    return 1
  fi
}

assert_file_exists() {
  local file="$1"
  if [ ! -f "$file" ]; then
    echo "Expected file to exist: $file"
    return 1
  fi
}

assert_dir_exists() {
  local dir="$1"
  if [ ! -d "$dir" ]; then
    echo "Expected directory to exist: $dir"
    return 1
  fi
}
