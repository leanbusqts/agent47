#!/usr/bin/env bash

ensure_test_runtime_seeded() {
  local launcher_path="${TEST_AFS_LAUNCHER:-$ROOT_DIR/bin/afs}"
  mkdir -p "$HOME/bin" "$AGENT47_HOME" "$AGENT47_HOME/bin"
  rm -rf "$AGENT47_HOME/templates"
  mkdir -p "$AGENT47_HOME/templates"

  cp -R "$ROOT_DIR/templates/." "$AGENT47_HOME/templates/"
  cp "$ROOT_DIR/VERSION" "$AGENT47_HOME/VERSION"
  cp "$launcher_path" "$AGENT47_HOME/bin/afs"
  chmod +x "$AGENT47_HOME/bin/afs"
  rm -f "$HOME/bin/afs"
  cp "$AGENT47_HOME/bin/afs" "$HOME/bin/afs"
  chmod +x "$HOME/bin/afs"
}

setup_workdir() {
  ensure_test_runtime_seeded
  TEST_WORKDIR="$(mktemp -d "$TEST_TMP_ROOT/work-XXXXXX")"
  cd "$TEST_WORKDIR" || return 1
}

teardown_workdir() {
  if [ -n "${TEST_WORKDIR:-}" ] && [ -d "$TEST_WORKDIR" ]; then
    rm -rf "$TEST_WORKDIR"
  fi
  cd "$ROOT_DIR" || return 1
}

make_test_repo_copy() {
  local repo_copy
  repo_copy="$(mktemp -d "$TEST_TMP_ROOT/repo-XXXXXX")"
  mkdir -p "$repo_copy/templates"
  cp "$ROOT_DIR/AGENTS.md" "$repo_copy/AGENTS.md"
  cp "$ROOT_DIR/VERSION" "$repo_copy/VERSION"
  cp -R "$ROOT_DIR/templates/." "$repo_copy/templates/"
  printf "%s\n" "$repo_copy"
}

assert_success() {
  # shellcheck disable=SC2154  # bats populates status/output dynamically
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

assert_not_contains() {
  local haystack="$1"
  local needle="$2"
  if [[ "$haystack" == *"$needle"* ]]; then
    echo "Expected output to not contain: $needle"
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
