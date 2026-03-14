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

prepare_user_script_stage() {
  local force="$1"
  local stage_root="$2"
  local stage_dir="$stage_root/stage"
  local script

  mkdir -p "$stage_dir"

  for script in "${INSTALLABLE_SCRIPTS[@]}"; do
    if [ -f "$USER_DIR/$script" ] && [ "$force" != "true" ]; then
      echo "[WARN] $script already exists in $USER_DIR (use --force to overwrite)"
      continue
    fi

    cp "$SCRIPTS_DIR/$script" "$stage_dir/$script"
  done
}

publish_user_scripts() {
  local stage_root="$1"
  local stage_dir="$stage_root/stage"
  local backup_dir="$stage_root/backup"
  local promoted_dir="$stage_root/promoted"
  local script target_path

  mkdir -p "$backup_dir" "$promoted_dir"

  for script in "${INSTALLABLE_SCRIPTS[@]}"; do
    target_path="$USER_DIR/$script"
    if [ ! -f "$stage_dir/$script" ]; then
      continue
    fi

    if [ -f "$target_path" ]; then
      cp "$target_path" "$backup_dir/$script"
    fi

    if mv "$stage_dir/$script" "$target_path"; then
      chmod +x "$target_path"
      clear_quarantine_attrs "$target_path"
      touch "$promoted_dir/$script"
      echo "[OK] Installed $script"
      continue
    fi

    rm -f "$target_path"
    if [ -f "$backup_dir/$script" ]; then
      mv "$backup_dir/$script" "$target_path"
    fi

    local promoted backup
    for promoted in "$promoted_dir"/*; do
      [ -e "$promoted" ] || continue
      script="$(basename "$promoted")"
      target_path="$USER_DIR/$script"
      if [ -f "$backup_dir/$script" ]; then
        cp "$backup_dir/$script" "$target_path"
      else
        rm -f "$target_path"
      fi
    done

    for backup in "$backup_dir"/*; do
      [ -e "$backup" ] || continue
      script="$(basename "$backup")"
      cp "$backup" "$USER_DIR/$script"
    done
    return 1
  done
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

preflight_install_assets() {
  require_install_asset file "$ROOT_DIR/bin/a47" || return 1
  require_install_asset dir "$ROOT_DIR/templates" || return 1
  require_install_asset file "$ROOT_DIR/templates/AGENTS.md" || return 1
  require_install_asset dir "$ROOT_DIR/scripts/lib" || return 1
  require_install_asset file "$ROOT_DIR/VERSION" || return 1
  require_install_asset file "$SCRIPTS_DIR/skill-utils.sh" || return 1

  local helper
  for helper in "${INSTALLABLE_SCRIPTS[@]}"; do
    require_install_asset file "$SCRIPTS_DIR/$helper" || return 1
  done
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

install_templates() {
  echo "[*] Installing agent47 templates..."

  mkdir -p "$AGENT47_HOME"

  local force="${1:-false}"

  if [ -d "$ROOT_DIR/templates" ]; then
    if [ -d "$AGENT47_HOME/templates" ] && [ "$force" != "true" ]; then
      echo "[WARN] Templates already exist at $AGENT47_HOME/templates (use --force to overwrite)"
    else
      install_dir_atomically "$ROOT_DIR/templates" "$AGENT47_HOME/templates" "$force" || return 1
      echo "[OK] Templates installed"
    fi
  else
    echo "[WARN] Templates directory not found in repo"
  fi

  mkdir -p "$AGENT47_HOME/scripts" "$AGENT47_HOME/scripts/lib"
  for helper in skill-utils.sh "${INSTALLABLE_SCRIPTS[@]}"; do
    if [ -f "$SCRIPTS_DIR/$helper" ]; then
      install_file_atomically "$SCRIPTS_DIR/$helper" "$AGENT47_HOME/scripts/$helper" || return 1
      chmod +x "$AGENT47_HOME/scripts/$helper"
      clear_quarantine_attrs "$AGENT47_HOME/scripts/$helper"
      echo "[OK] Helper installed: $helper"
    else
      echo "[WARN] Helper not found: $helper"
    fi
  done

  if [ -d "$SCRIPTS_DIR/lib" ]; then
    install_dir_atomically "$SCRIPTS_DIR/lib" "$AGENT47_HOME/scripts/lib" true || return 1
    find "$AGENT47_HOME/scripts/lib" -type f -exec chmod +x {} +
    clear_quarantine_attrs "$AGENT47_HOME/scripts/lib"
    echo "[OK] Helper library installed"
  else
    echo "[WARN] Helper library directory not found"
  fi

  if [ -f "$ROOT_DIR/VERSION" ]; then
    install_file_atomically "$ROOT_DIR/VERSION" "$AGENT47_HOME/VERSION" || return 1
    echo "[OK] VERSION installed"
  else
    echo "[WARN] VERSION file not found in repo"
  fi
}

install_scripts() {
  echo "[*] Installing agent47 scripts..."

  local force=false
  local user_stage_root
  if [ "${1:-}" = "--force" ]; then
    force=true
    shift
  fi

  preflight_install_assets || return 1

  mkdir -p "$USER_DIR"
  mkdir -p "$AGENT47_HOME/bin" "$AGENT47_HOME/scripts"
  user_stage_root="$(mktemp -d "${TMPDIR:-/tmp}/a47-user-install-XXXXXX")"

  prepare_user_script_stage "$force" "$user_stage_root"

  install_file_atomically "$ROOT_DIR/bin/a47" "$AGENT47_HOME/bin/a47"
  chmod +x "$AGENT47_HOME/bin/a47"
  clear_quarantine_attrs "$AGENT47_HOME/bin/a47"
  echo "[OK] Installed a47 launcher"

  for script in "${INSTALLABLE_SCRIPTS[@]}"; do
    if [ -f "$AGENT47_HOME/scripts/$script" ] && [ "$force" != "true" ]; then
      echo "[WARN] $script already exists in $AGENT47_HOME/scripts (use --force to overwrite)"
    else
      install_file_atomically "$SCRIPTS_DIR/$script" "$AGENT47_HOME/scripts/$script"
      chmod +x "$AGENT47_HOME/scripts/$script"
      clear_quarantine_attrs "$AGENT47_HOME/scripts/$script"
    fi
  done

  for legacy in "${LEGACY_SCRIPTS[@]}"; do
    if [ -f "$USER_DIR/$legacy" ]; then
      rm -f "$USER_DIR/$legacy"
      echo "[INFO] Removed legacy script: $legacy"
    fi
    if [ -f "$AGENT47_HOME/scripts/$legacy" ]; then
      rm -f "$AGENT47_HOME/scripts/$legacy"
    fi
  done

  if ! install_templates "$force"; then
    rm -rf "$user_stage_root"
    return 1
  fi

  if ! publish_user_scripts "$user_stage_root"; then
    rm -rf "$user_stage_root"
    return 1
  fi

  rm -rf "$user_stage_root"

  echo "[OK] a47 installation complete"
}

uninstall() {
  echo "[*] Uninstalling a47 scripts..."

  for script in "${INSTALLABLE_SCRIPTS[@]}" "${LEGACY_SCRIPTS[@]}"; do
    if [ -f "$USER_DIR/$script" ]; then
      rm "$USER_DIR/$script"
      echo "[OK] Removed $script"
    else
      echo "[WARN] $script not found in $USER_DIR"
    fi
  done

  if [ -L "$USER_DIR/a47" ]; then
    rm "$USER_DIR/a47"
    echo "[OK] Removed a47 symlink"
  fi

  if [ -f "$AGENT47_HOME/bin/a47" ]; then
    rm "$AGENT47_HOME/bin/a47"
    echo "[OK] Removed installed a47 launcher"
  fi

  echo "[OK] a47 tools removed from system"
}
