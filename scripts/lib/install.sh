#!/bin/bash

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

  if [ -d "$AGENT47_HOME/templates" ]; then
    if [ "$force" != "true" ]; then
      echo "[WARN] Templates already exist at $AGENT47_HOME/templates (use --force to overwrite)"
      return
    fi
    echo "[WARN] Overwriting existing templates at $AGENT47_HOME/templates"
    ts="$(date +%Y%m%d%H%M%S)"
    bak_dir="$AGENT47_HOME/templates.bak.$ts"
    rm -rf "$AGENT47_HOME"/templates.bak.*
    cp -R "$AGENT47_HOME/templates" "$bak_dir"
    echo "[INFO] Backup created: $bak_dir"
  fi

  if [ -d "$ROOT_DIR/templates" ]; then
    rm -rf "$AGENT47_HOME/templates"
    cp -R "$ROOT_DIR/templates" "$AGENT47_HOME/"
    echo "[OK] Templates installed"
  else
    echo "[WARN] Templates directory not found in repo"
  fi

  mkdir -p "$AGENT47_HOME/scripts" "$AGENT47_HOME/scripts/lib"
  for helper in skill-utils.sh "${INSTALLABLE_SCRIPTS[@]}"; do
    if [ -f "$SCRIPTS_DIR/$helper" ]; then
      cp "$SCRIPTS_DIR/$helper" "$AGENT47_HOME/scripts/"
      chmod +x "$AGENT47_HOME/scripts/$helper"
      clear_quarantine_attrs "$AGENT47_HOME/scripts/$helper"
      echo "[OK] Helper installed: $helper"
    else
      echo "[WARN] Helper not found: $helper"
    fi
  done

  if [ -d "$SCRIPTS_DIR/lib" ]; then
    rm -rf "$AGENT47_HOME/scripts/lib"
    cp -R "$SCRIPTS_DIR/lib" "$AGENT47_HOME/scripts/"
    find "$AGENT47_HOME/scripts/lib" -type f -exec chmod +x {} +
    clear_quarantine_attrs "$AGENT47_HOME/scripts/lib"
    echo "[OK] Helper library installed"
  else
    echo "[WARN] Helper library directory not found"
  fi

  if [ -f "$ROOT_DIR/VERSION" ]; then
    cp "$ROOT_DIR/VERSION" "$AGENT47_HOME/VERSION"
    echo "[OK] VERSION installed"
  else
    echo "[WARN] VERSION file not found in repo"
  fi
}

install_scripts() {
  echo "[*] Installing agent47 scripts..."

  local force=false
  if [ "${1:-}" = "--force" ]; then
    force=true
    shift
  fi

  mkdir -p "$USER_DIR"
  mkdir -p "$AGENT47_HOME/bin" "$AGENT47_HOME/scripts"

  if [ -f "$ROOT_DIR/bin/a47" ]; then
    cp "$ROOT_DIR/bin/a47" "$AGENT47_HOME/bin/a47"
    chmod +x "$AGENT47_HOME/bin/a47"
    clear_quarantine_attrs "$AGENT47_HOME/bin/a47"
    echo "[OK] Installed a47 launcher"
  else
    echo "[WARN] a47 launcher not found in repo"
  fi

  for script in "${INSTALLABLE_SCRIPTS[@]}"; do
    if [ -f "$SCRIPTS_DIR/$script" ]; then
      if [ -f "$USER_DIR/$script" ] && [ "$force" != "true" ]; then
        echo "[WARN] $script already exists in $USER_DIR (use --force to overwrite)"
      else
        cp "$SCRIPTS_DIR/$script" "$USER_DIR/$script"
        chmod +x "$USER_DIR/$script"
        clear_quarantine_attrs "$USER_DIR/$script"
        echo "[OK] Installed $script"
      fi

      if [ -f "$AGENT47_HOME/scripts/$script" ] && [ "$force" != "true" ]; then
        echo "[WARN] $script already exists in $AGENT47_HOME/scripts (use --force to overwrite)"
      else
        cp "$SCRIPTS_DIR/$script" "$AGENT47_HOME/scripts/$script"
        chmod +x "$AGENT47_HOME/scripts/$script"
        clear_quarantine_attrs "$AGENT47_HOME/scripts/$script"
      fi
    else
      echo "[WARN] Script not found: $script"
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

  install_templates "$force"

  echo "[OK] a47 installation complete"
}

upgrade() {
  echo "[*] Upgrading a47 scripts..."
  install_scripts "${@:1}"
  echo "[OK] Upgrade completed"
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
