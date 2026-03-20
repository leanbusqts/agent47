#!/bin/bash
set -euo pipefail

REPO_DIR="$(cd "$(dirname "$0")" && pwd)"
BIN="$REPO_DIR/bin/afs"
USER_BIN="$HOME/bin"
INSTALLED_BIN_DIR="$HOME/.agent47/bin"
INSTALLED_BIN="$INSTALLED_BIN_DIR/afs"
VERSION="$(cat "$REPO_DIR/VERSION" 2>/dev/null || echo "unknown")"
RUNTIME_ENV_LIB="$REPO_DIR/scripts/lib/runtime-env.sh"

# shellcheck disable=SC1090
source "$RUNTIME_ENV_LIB"
init_agent47_runtime_from_entrypoint "$BIN"
load_agent47_libs constants.sh managed-files.sh install-assets.sh install-runtime.sh install.sh

ensure_path_persistent() {
  rc_file="$(detect_shell_rc_file)"
  # shellcheck disable=SC2016
  export_line='export PATH="$HOME/bin:$PATH"'

  if grep -F "$export_line" "$rc_file" >/dev/null 2>&1; then
    echo "[OK] ~/bin export already present in $rc_file"
    return
  fi

  echo "[HINT] Add \"$export_line\" to $rc_file to persist PATH"
  if [ "$NO_PROMPT" = true ] || [ ! -t 0 ] || [ ! -t 1 ]; then
    echo "[WARN] Non-interactive install; skipping shell rc update"
    return 0
  fi

  read -r -p "Add it now? [y/N]: " add_path_reply
  case "$add_path_reply" in
    y|Y|yes|YES)
      if [ -f "$rc_file" ]; then
        cp "$rc_file" "$rc_file.bak"
        echo "[INFO] Backup created: $rc_file.bak"
      fi
      echo "$export_line" >> "$rc_file"
      echo "[OK] Added to $rc_file"
      ;;
    *)
      echo "[WARN] Skipped adding to $rc_file; update manually if needed"
      ;;
  esac
}

usage() {
  echo "Usage: ./install.sh [--force] [--no-prompt]"
}

FORCE=false
NO_PROMPT=false
while [ "$#" -gt 0 ]; do
  case "$1" in
    --force)
      FORCE=true
      shift
      ;;
    --no-prompt)
      NO_PROMPT=true
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

# 1) Ensure executable
chmod +x "$BIN"

# 2) Install scripts + templates
if [ "$FORCE" = true ]; then
  install_scripts --force
else
  install_scripts
fi

# 3) Ensure ~/bin exists
mkdir -p "$USER_BIN"

# 4) Create/refresh symlink (primary entrypoint: afs)
if [ -e "$USER_BIN/afs" ] && [ "$FORCE" != true ]; then
  echo "[WARN] afs entry already exists in ~/bin (use --force to refresh)"
else
  install_symlink_atomically "$INSTALLED_BIN" "$USER_BIN/afs"
  echo "[OK] Linked afs into ~/bin -> $INSTALLED_BIN"
fi

# 5) PATH check
if [[ ":$PATH:" != *":$USER_BIN:"* ]]; then
  echo "[WARN] ~/bin not in PATH"
  echo "[HINT] Add this to your shell config:"
  # shellcheck disable=SC2016
  echo 'export PATH="$HOME/bin:$PATH"'
  export PATH="$HOME/bin:$PATH"
  echo "[OK] Temporarily added ~/bin to PATH for this session"
  ensure_path_persistent
else
  echo "[OK] ~/bin in PATH"
fi

echo "[OK] afs installed"
echo "[HINT] Run: afs doctor"
