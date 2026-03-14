#!/bin/bash

doctor() {
  local check_updates=false
  local force_update=false

  case "${1:-}" in
    --check-update)
      check_updates=true
      ;;
    --check-update-force)
      check_updates=true
      force_update=true
      ;;
    "")
      ;;
    *)
      echo "Usage: a47 doctor [--check-update|--check-update-force]"
      return 1
      ;;
  esac

  echo "[*] a47 doctor"
  echo "[INFO] Version: $AGENT47_VERSION"

  if command -v a47 >/dev/null; then
    echo "[OK] a47 in PATH"
  else
    echo "[WARN] a47 not in PATH"
    echo "[HINT] Fix: run ./install.sh"
  fi

  for script in "${INSTALLABLE_SCRIPTS[@]}"; do
    if command -v "$script" >/dev/null; then
      echo "[OK] $script available"
    else
      echo "[WARN] $script missing"
      echo "[HINT] Fix: a47 install"
    fi
  done

  for legacy in "${LEGACY_SCRIPTS[@]}"; do
    if command -v "$legacy" >/dev/null; then
      echo "[INFO] Legacy script detected: $legacy (remove old install)"
      echo "[HINT] Run a47 install to clean legacy scripts"
    fi
  done

  if [ -d "$AGENT47_HOME/templates" ]; then
    echo "[OK] Templates installed"
    check_prompt_template "$AGENT47_HOME/templates" || true
    check_security_templates "$AGENT47_HOME/templates" || true
    check_security_rule_ids "$AGENT47_HOME/templates" || true
    check_agents_sections "$AGENT47_HOME/templates/AGENTS.md" || true
  else
    echo "[WARN] Templates missing"
    echo "[HINT] Fix: a47 install"
  fi

  if [ -d "$AGENT47_HOME/templates/skills" ]; then
    if ls "$AGENT47_HOME/templates/skills"/*.yml >/dev/null 2>&1; then
      echo "[WARN] Legacy .yml skills found in $AGENT47_HOME/templates/skills"
      echo "[HINT] Run a47 install to refresh .md skill templates"
    else
      echo "[OK] Skills templates (.md) present"
    fi
  fi

  if [ -x "$ROOT_DIR/tests/vendor/bats/bin/bats" ] || command -v bats >/dev/null 2>&1; then
    echo "[OK] bats available"
  else
    echo "[WARN] bats missing"
  fi

  if [ -L "$USER_DIR/a47" ]; then
    echo "[OK] a47 symlink present in ~/bin"
  else
    echo "[WARN] a47 symlink missing"
    echo "[HINT] Fix: run ./install.sh"
  fi

  if [[ ":$PATH:" == *":$USER_DIR:"* ]]; then
    echo "[OK] ~/bin in PATH"
  else
    echo "[WARN] ~/bin not in PATH"
    echo "[HINT] Add to your shell config:"
    echo '       export PATH="$HOME/bin:$PATH"'
  fi

  if [ "$check_updates" = true ]; then
    if [ "$force_update" = true ]; then
      check_update --force
    else
      check_update
    fi
  else
    echo "[INFO] Skipping update check by default"
    echo "[HINT] Run: a47 doctor --check-update"
  fi
}
