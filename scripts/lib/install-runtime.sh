#!/bin/bash
set -euo pipefail

install_runtime_require_context() {
  require_runtime_vars ROOT_DIR SCRIPTS_DIR USER_DIR AGENT47_HOME PROJECT_AGENTS_FILE
  assert_manifest_contract
}

cleanup_managed_runtime_backups() {
  rm -rf "${AGENT47_HOME:?}"/templates.bak.*
  rm -rf "${AGENT47_HOME:?}"/scripts/*.bak.*
}

detect_shell_rc_file() {
  local shell_name="${1:-$(basename "${SHELL:-}")}"

  case "$shell_name" in
    zsh)
      printf "%s\n" "$HOME/.zshrc"
      ;;
    bash)
      if [ -f "$HOME/.bash_profile" ]; then
        printf "%s\n" "$HOME/.bash_profile"
      elif [ "$(uname -s 2>/dev/null || echo unknown)" = "Darwin" ]; then
        printf "%s\n" "$HOME/.bash_profile"
      elif [ -f "$HOME/.bashrc" ]; then
        printf "%s\n" "$HOME/.bashrc"
      else
        printf "%s\n" "$HOME/.profile"
      fi
      ;;
    *)
      printf "%s\n" "$HOME/.profile"
      ;;
  esac
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

preflight_install_assets() {
  local helper
  local template_path

  install_runtime_require_context || return 1

  require_install_asset file "$ROOT_DIR/bin/a47" || return 1
  require_install_asset dir "$ROOT_DIR/templates" || return 1
  require_install_asset dir "$ROOT_DIR/scripts/lib" || return 1
  require_install_asset file "$ROOT_DIR/VERSION" || return 1
  require_install_asset file "$SCRIPTS_DIR/lib/skill-utils.sh" || return 1

  while IFS= read -r template_path; do
    require_install_asset file "$ROOT_DIR/templates/$template_path" || return 1
  done < <(project_required_template_files)

  while IFS= read -r template_path; do
    require_install_asset dir "$ROOT_DIR/templates/$template_path" || return 1
  done < <(project_required_template_dirs)

  while IFS= read -r helper; do
    require_install_asset file "$ROOT_DIR/templates/rules/$helper" || return 1
  done < <(project_rule_template_files)

  for helper in "${INSTALLABLE_SCRIPTS[@]}"; do
    require_install_asset file "$SCRIPTS_DIR/$helper" || return 1
  done
}

install_managed_templates() {
  echo "[*] Installing agent47 templates..."

  install_runtime_require_context || return 1
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

  if [ -f "$ROOT_DIR/VERSION" ]; then
    install_file_atomically "$ROOT_DIR/VERSION" "$AGENT47_HOME/VERSION" || return 1
    echo "[OK] VERSION installed"
  else
    echo "[WARN] VERSION file not found in repo"
  fi
}

install_managed_runtime() {
  local force="${1:-false}"
  local script
  local legacy
  local user_stage_root

  echo "[*] Installing agent47 scripts..."

  install_runtime_require_context || return 1
  preflight_install_assets || return 1

  mkdir -p "$USER_DIR"
  mkdir -p "$AGENT47_HOME/bin" "$AGENT47_HOME/scripts"
  user_stage_root="$(mktemp -d "${TMPDIR:-/tmp}/a47-user-install-XXXXXX")"

  prepare_user_script_stage "$force" "$user_stage_root"

  if ! install_managed_templates "$force"; then
    rm -rf "$user_stage_root"
    return 1
  fi

  if [ -f "$AGENT47_HOME/bin/a47" ] && [ "$force" != "true" ]; then
    echo "[WARN] a47 launcher already exists in $AGENT47_HOME/bin (use --force to overwrite)"
  else
    install_file_atomically "$ROOT_DIR/bin/a47" "$AGENT47_HOME/bin/a47"
    chmod +x "$AGENT47_HOME/bin/a47"
    clear_quarantine_attrs "$AGENT47_HOME/bin/a47"
    echo "[OK] Installed a47 launcher"
  fi

  for script in "${INSTALLABLE_SCRIPTS[@]}"; do
    if [ -f "$AGENT47_HOME/scripts/$script" ] && [ "$force" != "true" ]; then
      echo "[WARN] $script already exists in $AGENT47_HOME/scripts (use --force to overwrite)"
    else
      install_file_atomically "$SCRIPTS_DIR/$script" "$AGENT47_HOME/scripts/$script"
      chmod +x "$AGENT47_HOME/scripts/$script"
      clear_quarantine_attrs "$AGENT47_HOME/scripts/$script"
      echo "[OK] Helper installed: $script"
    fi
  done

  if [ -d "$SCRIPTS_DIR/lib" ]; then
    if [ -d "$AGENT47_HOME/scripts/lib" ] && [ "$force" != "true" ]; then
      echo "[WARN] helper library already exists in $AGENT47_HOME/scripts/lib (use --force to overwrite)"
    else
      install_dir_atomically "$SCRIPTS_DIR/lib" "$AGENT47_HOME/scripts/lib" true || return 1
      find "$AGENT47_HOME/scripts/lib" -type f -exec chmod +x {} +
      clear_quarantine_attrs "$AGENT47_HOME/scripts/lib"
      echo "[OK] Helper library installed"
    fi
  else
    echo "[WARN] Helper library directory not found"
  fi

  for legacy in "${LEGACY_SCRIPTS[@]}"; do
    if [ -f "$USER_DIR/$legacy" ]; then
      rm -f "$USER_DIR/$legacy"
      echo "[INFO] Removed legacy script: $legacy"
    fi
    if [ -f "$AGENT47_HOME/scripts/$legacy" ]; then
      rm -f "$AGENT47_HOME/scripts/$legacy"
    fi
  done

  if ! publish_user_scripts "$user_stage_root"; then
    rm -rf "$user_stage_root"
    return 1
  fi

  rm -rf "$user_stage_root"
  echo "[OK] a47 installation complete"
}

uninstall_managed_runtime() {
  local script

  echo "[*] Uninstalling a47 scripts..."

  install_runtime_require_context || return 1
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

  rm -rf "${AGENT47_HOME:?}/bin" "${AGENT47_HOME:?}/scripts" "${AGENT47_HOME:?}/templates" "${AGENT47_HOME:?}/cache"
  cleanup_managed_runtime_backups
  rm -f "$AGENT47_HOME/VERSION"
  if [ -d "$AGENT47_HOME" ]; then
    rmdir "$AGENT47_HOME" >/dev/null 2>&1 || true
  fi

  echo "[OK] a47 tools removed from system"
}
