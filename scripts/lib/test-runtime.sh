#!/bin/bash
set -euo pipefail

init_test_runtime() {
  ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
  TEST_TMP_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/a47-test-XXXXXX")"
  BATS_BIN="${BATS_BIN:-}"
}

cleanup_test_runtime() {
  rm -rf "$TEST_TMP_ROOT"
}

seed_test_runtime_environment() {
  mkdir -p "$HOME/bin" "$AGENT47_HOME" "$AGENT47_HOME/scripts" "$AGENT47_HOME/scripts/lib"
  rm -rf "$AGENT47_HOME/templates"
  mkdir -p "$AGENT47_HOME/templates"

  cp -R "$ROOT_DIR/templates/." "$AGENT47_HOME/templates/"
  cp "$ROOT_DIR/VERSION" "$AGENT47_HOME/VERSION"
  cp "$ROOT_DIR/scripts/add-agent" "$AGENT47_HOME/scripts/"
  cp "$ROOT_DIR/scripts/add-agent-prompt" "$AGENT47_HOME/scripts/"
  cp "$ROOT_DIR/scripts/add-snapshot-prompt" "$AGENT47_HOME/scripts/"
  rm -rf "$AGENT47_HOME/scripts/lib"
  mkdir -p "$AGENT47_HOME/scripts/lib"
  cp -R "$ROOT_DIR/scripts/lib/." "$AGENT47_HOME/scripts/lib/"
  find "$AGENT47_HOME/scripts" -type f -exec chmod +x {} +
}

resolve_bats_bin() {
  local vendor_bin="$ROOT_DIR/tests/vendor/bats/bin/bats"
  local vendor_install="$ROOT_DIR/tests/vendor/bats/install.sh"
  local temp_bats_root="$TEST_TMP_ROOT/tools/bats"

  if [ -n "$BATS_BIN" ] && [ -x "$BATS_BIN" ]; then
    return
  fi

  if [ -x "$vendor_bin" ]; then
    BATS_BIN="$vendor_bin"
    return
  fi

  if command -v bats >/dev/null 2>&1; then
    BATS_BIN="$(command -v bats)"
    return
  fi

  if [ -x "$vendor_install" ]; then
    echo "[INFO] bats not found; installing a temporary copy from tests/vendor/bats"
    mkdir -p "$temp_bats_root"
    bash "$vendor_install" "$temp_bats_root" >/dev/null
    BATS_BIN="$temp_bats_root/bin/bats"
    return
  fi

  echo "[ERR] bats not found."
  echo "      Set BATS_BIN, install bats on PATH, or restore tests/vendor/bats."
  exit 1
}

stage_test_environment() {
  export HOME="$TEST_TMP_ROOT/home"
  export AGENT47_HOME="$HOME/.agent47"
  export PATH="$ROOT_DIR/bin:$ROOT_DIR/scripts:$HOME/bin:$PATH"
  export BATS_LIB_PATH="$ROOT_DIR/tests/helpers:$ROOT_DIR/tests"
  export TEST_TMP_ROOT
  export ROOT_DIR

  seed_test_runtime_environment
}

collect_test_paths() {
  if [ "$#" -gt 0 ]; then
    printf "%s\n" "$@"
    return
  fi

  find "$ROOT_DIR/tests" -path "$ROOT_DIR/tests/vendor" -prune -o -type f -name '*.bats' -print | sort
}

run_test_suite() {
  local test_paths=()
  local file

  while IFS= read -r file; do
    [ -n "$file" ] || continue
    test_paths+=("$file")
  done < <(collect_test_paths "$@")

  if [ "${#test_paths[@]}" -eq 0 ]; then
    echo "[WARN] No tests found under $ROOT_DIR/tests"
    exit 0
  fi

  echo "[INFO] Running tests with HOME=$HOME and AGENT47_HOME=$AGENT47_HOME"
  echo "[INFO] Using bats binary: $BATS_BIN"
  "$BATS_BIN" "${test_paths[@]}"
}
