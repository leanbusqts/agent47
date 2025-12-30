#!/bin/bash
set -e

REPO_DIR="$(cd "$(dirname "$0")" && pwd)"
BIN="$REPO_DIR/bin/agent47"
USER_BIN="$HOME/bin"
VERSION="$(cat "$REPO_DIR/VERSION" 2>/dev/null || echo "unknown")"

detect_rc_file() {
  shell_name="$(basename "${SHELL:-}")"
  case "$shell_name" in
    zsh) echo "$HOME/.zshrc" ;;
    bash) echo "$HOME/.bashrc" ;;
    *) echo "$HOME/.profile" ;;
  esac
}

ensure_path_persistent() {
  rc_file="$(detect_rc_file)"
  export_line='export PATH="$HOME/bin:$PATH"'

  if grep -F "$export_line" "$rc_file" >/dev/null 2>&1; then
    echo "[OK] ~/bin export already present in $rc_file"
    return
  fi

  echo "[HINT] Add \"$export_line\" to $rc_file to persist PATH"
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

echo "[ AGENT47 v$VERSION ]"
echo "[*] Installing agent47..."

# 1) Ensure executable
chmod +x "$BIN"

# 2) Install scripts + templates
"$BIN" install

# 3) Ensure ~/bin exists
mkdir -p "$USER_BIN"

# 4) Create symlink if missing
if [ ! -L "$USER_BIN/agent47" ]; then
  ln -s "$BIN" "$USER_BIN/agent47"
echo "[OK] Linked agent47 into ~/bin"
else
  echo "[INFO] agent47 already linked"
fi

# 5) PATH check
if [[ ":$PATH:" != *":$USER_BIN:"* ]]; then
  echo "[WARN] ~/bin not in PATH"
  echo "[HINT] Add this to your shell config:"
  echo 'export PATH="$HOME/bin:$PATH"'
  export PATH="$HOME/bin:$PATH"
  echo "[OK] Temporarily added ~/bin to PATH for this session"
  ensure_path_persistent
else
  echo "[OK] ~/bin in PATH"
fi

echo "[OK] agent47 installed"
echo "[HINT] Run: agent47 doctor"
