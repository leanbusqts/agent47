#!/bin/bash
set -euo pipefail

install_file_atomically() {
  local source_path="$1"
  local target_path="$2"
  local target_dir tmp_file

  target_dir="$(dirname "$target_path")"
  mkdir -p "$target_dir"
  tmp_file="$(mktemp "$target_dir/.a47-tmp.XXXXXX")"
  cp "$source_path" "$tmp_file"
  mv -f "$tmp_file" "$target_path"
}

install_dir_atomically() {
  local source_dir="$1"
  local target_dir="$2"
  local force="${3:-false}"
  local parent_dir target_name stage_dir bak_dir ts

  parent_dir="$(dirname "$target_dir")"
  target_name="$(basename "$target_dir")"
  mkdir -p "$parent_dir"
  stage_dir="$(mktemp -d "$parent_dir/.${target_name}.tmp.XXXXXX")"
  cp -R "$source_dir/." "$stage_dir/"

  if [ -d "$target_dir" ]; then
    if [ "$force" != "true" ]; then
      rm -rf "$stage_dir"
      echo "[WARN] Templates already exist at $target_dir (use --force to overwrite)"
      return 1
    fi

    echo "[WARN] Overwriting existing templates at $target_dir"
    ts="$(date +%Y%m%d%H%M%S)"
    bak_dir="$parent_dir/${target_name}.bak.$ts"
    rm -rf "$parent_dir"/"${target_name}.bak."*
    mv "$target_dir" "$bak_dir"
    echo "[INFO] Backup created: $bak_dir"
  fi

  if [ "${AGENT47_ENABLE_TEST_HOOKS:-false}" = "true" ] && [ -n "${AGENT47_FAIL_DIR_SWAP_TARGET:-}" ] && [ "$target_dir" = "$AGENT47_FAIL_DIR_SWAP_TARGET" ]; then
    if [ -z "${AGENT47_FAIL_DIR_SWAP_MARKER:-}" ] || [ ! -f "$AGENT47_FAIL_DIR_SWAP_MARKER" ]; then
      if [ -n "${AGENT47_FAIL_DIR_SWAP_MARKER:-}" ]; then
        : > "$AGENT47_FAIL_DIR_SWAP_MARKER"
      fi
      if [ -n "${bak_dir:-}" ] && [ -d "$bak_dir" ] && [ ! -e "$target_dir" ]; then
        mv "$bak_dir" "$target_dir" || true
      fi
      rm -rf "$stage_dir"
      return 1
    fi
  fi

  if mv "$stage_dir" "$target_dir"; then
    return 0
  fi

  if [ -n "${bak_dir:-}" ] && [ -d "$bak_dir" ] && [ ! -e "$target_dir" ]; then
    mv "$bak_dir" "$target_dir" || true
  fi

  rm -rf "$stage_dir"
  return 1
}

install_symlink_atomically() {
  local target_path="$1"
  local link_path="$2"
  local link_dir link_name tmp_link

  link_dir="$(dirname "$link_path")"
  link_name="$(basename "$link_path")"
  mkdir -p "$link_dir"

  tmp_link="$link_dir/.${link_name}.tmp.$$"
  rm -f "$tmp_link"

  ln -s "$target_path" "$tmp_link"

  if mv "$tmp_link" "$link_path"; then
    return 0
  fi

  rm -f "$tmp_link"
  return 1
}

clear_quarantine_attrs() {
  if ! command -v xattr >/dev/null 2>&1; then
    return
  fi

  local target
  for target in "$@"; do
    [ -e "$target" ] || continue
    xattr -dr com.apple.quarantine "$target" >/dev/null 2>&1 || true
  done
}

require_install_asset() {
  local kind="$1"
  local path="$2"

  case "$kind" in
    file)
      [ -f "$path" ] || {
        echo "[ERR] Required install asset missing: $path"
        return 1
      }
      ;;
    dir)
      [ -d "$path" ] || {
        echo "[ERR] Required install asset missing: $path"
        return 1
      }
      ;;
    *)
      echo "[ERR] Unknown asset kind: $kind"
      return 1
      ;;
  esac
}
