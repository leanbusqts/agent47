#!/bin/bash
set -euo pipefail

resolve_self_dir() {
  local source dir

  source="${BASH_SOURCE[0]}"
  while [ -L "$source" ]; do
    case "$source" in
      */*) dir="$(cd "${source%/*}" && pwd)" ;;
      *) dir="$(pwd)" ;;
    esac
    source="$(readlink "$source")"
    case "$source" in
      /*) ;;
      *) source="$dir/$source" ;;
    esac
  done

  case "$source" in
    */*) dir="$(cd "${source%/*}" && pwd)" ;;
    *) dir="$(pwd)" ;;
  esac
  printf "%s\n" "$dir"
}

REPO_DIR="$(resolve_self_dir)"
VERSION="$(cat "$REPO_DIR/VERSION" 2>/dev/null || echo "unknown")"

install_with_go_runtime() {
  local -a install_args

  install_args=()
  if [ "$FORCE" = true ]; then
    install_args+=(--force)
  fi
  if [ "$NON_INTERACTIVE" = true ]; then
    install_args+=(--non-interactive)
  fi

  if [ "${#install_args[@]}" -gt 0 ]; then
    "$REPO_DIR/bin/afs" __agent47_internal_install "${install_args[@]}"
    return
  fi

  "$REPO_DIR/bin/afs" __agent47_internal_install
}

usage() {
  echo "Usage: ./install.sh [--force] [--non-interactive]"
}

FORCE=false
NON_INTERACTIVE=false
while [ "$#" -gt 0 ]; do
  case "$1" in
    --force)
      FORCE=true
      shift
      ;;
    --non-interactive)
      NON_INTERACTIVE=true
      shift
      ;;
    *)
      usage
      exit 1
      ;;
  esac
done

echo "[ AGENT47 v$VERSION ]"
echo "[*] Installing agent47..."

# 1) Install templates + managed Go launcher through the repo launcher
install_with_go_runtime

# 2) The Go installer handles PATH hints and shell-rc interaction
