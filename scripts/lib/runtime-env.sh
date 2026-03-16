#!/bin/bash
set -euo pipefail

resolve_real_source_path() {
  local source_path="$1"
  local dir

  while [ -h "$source_path" ]; do
    dir="${source_path%/*}"
    if [ "$dir" = "$source_path" ]; then
      dir="."
    fi
    dir="$(cd -P "$dir" && pwd)"
    source_path="$(readlink "$source_path")"
    [[ $source_path != /* ]] && source_path="$dir/$source_path"
  done

  printf "%s\n" "$source_path"
}

init_agent47_runtime_from_entrypoint() {
  local entrypoint_path="$1"
  local real_source
  local script_dir

  # Runtime globals are initialized here for other sourced modules.
  # shellcheck disable=SC2034
  USER_DIR="${USER_DIR:-$HOME/bin}"
  AGENT47_HOME="${AGENT47_HOME:-$HOME/.agent47}"

  real_source="$(resolve_real_source_path "$entrypoint_path")"
  script_dir="${real_source%/*}"
  if [ "$script_dir" = "$real_source" ]; then
    script_dir="."
  fi
  script_dir="$(cd -P "$script_dir" && pwd)"

  # shellcheck disable=SC2034
  SCRIPT_DIR="$script_dir"
  ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
  # shellcheck disable=SC2034
  SCRIPTS_DIR="$ROOT_DIR/scripts"
  LIB_DIR="$ROOT_DIR/scripts/lib"
  VERSION_FILE="$AGENT47_HOME/VERSION"
  CACHE_DIR="$AGENT47_HOME/cache"
  # shellcheck disable=SC2034
  UPDATE_CACHE_FILE="$CACHE_DIR/update.cache"
  # shellcheck disable=SC2034
  UPDATE_CACHE_TTL_SECONDS=86400
  DEFAULT_VERSION_URL="https://raw.githubusercontent.com/leanbusqts/agent47/main/VERSION"
  # shellcheck disable=SC2034
  REMOTE_VERSION_URL="${AGENT47_VERSION_URL:-$DEFAULT_VERSION_URL}"

  if [ -f "$ROOT_DIR/VERSION" ]; then
    AGENT47_VERSION="$(cat "$ROOT_DIR/VERSION")"
  elif [ -f "$VERSION_FILE" ]; then
    AGENT47_VERSION="$(cat "$VERSION_FILE")"
  else
    # shellcheck disable=SC2034
    AGENT47_VERSION="unknown"
  fi
}

require_agent47_lib() {
  local lib_name="$1"
  local lib_path="$LIB_DIR/$lib_name"

  if [ ! -f "$lib_path" ]; then
    echo "[ERR] Missing library: $lib_path"
    exit 1
  fi
}

load_agent47_libs() {
  local lib_name

  for lib_name in "$@"; do
    require_agent47_lib "$lib_name"
    # shellcheck disable=SC1090
    source "$LIB_DIR/$lib_name"
  done
}

require_runtime_vars() {
  local var_name
  local var_value

  for var_name in "$@"; do
    var_value="${!var_name-}"
    if [ -z "$var_value" ]; then
      echo "[ERR] Missing required runtime variable: $var_name" >&2
      return 1
    fi
  done
}
