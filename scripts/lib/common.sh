#!/bin/bash

run_cmd() {
  local cmd="$1"
  shift || true

  if command -v "$cmd" >/dev/null 2>&1; then
    "$cmd" "$@"
    return $?
  fi

  if [ -x "$SCRIPTS_DIR/$cmd" ]; then
    "$SCRIPTS_DIR/$cmd" "$@"
    return $?
  fi

  echo "[ERR] Command not found: $cmd"
  return 1
}
