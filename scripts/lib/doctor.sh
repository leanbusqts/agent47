#!/bin/bash

doctor_resolve_path() {
  local target="$1"
  local current resolved_dir link_target depth=0

  if [ -z "$target" ]; then
    return 1
  fi

  current="$target"
  while [ -L "$current" ]; do
    depth=$((depth + 1))
    [ "$depth" -le 20 ] || return 1
    link_target="$(readlink "$current" 2>/dev/null)" || return 1
    if [[ "$link_target" = /* ]]; then
      current="$link_target"
    else
      resolved_dir="$(cd "$(dirname "$current")" >/dev/null 2>&1 && pwd -P)" || return 1
      current="$resolved_dir/$link_target"
    fi
  done

  if [ -d "$current" ]; then
    (
      cd "$current" >/dev/null 2>&1 && pwd -P
    )
    return $?
  fi

  if [ -e "$current" ]; then
    (
      cd "$(dirname "$current")" >/dev/null 2>&1 && printf "%s/%s\n" "$(pwd -P)" "$(basename "$current")"
    )
    return $?
  fi

  return 1
}

doctor_is_managed_command() {
  local cmd_name="$1"
  local managed_target="$2"
  local actual_path expected_resolved actual_resolved

  actual_path="$(command -v "$cmd_name" 2>/dev/null || true)"
  [ -n "$actual_path" ] || return 1

  actual_resolved="$(doctor_resolve_path "$actual_path" || true)"
  expected_resolved="$(doctor_resolve_path "$managed_target" || true)"

  [ -n "$actual_resolved" ] && [ -n "$expected_resolved" ] && [ "$actual_resolved" = "$expected_resolved" ]
}

doctor_symlink_matches() {
  local link_path="$1"
  local expected_target="$2"
  local resolved_target expected_resolved

  [ -L "$link_path" ] || return 1
  resolved_target="$(doctor_resolve_path "$link_path" || true)"
  expected_resolved="$(doctor_resolve_path "$expected_target" || true)"
  [ -n "$resolved_target" ] && [ -n "$expected_resolved" ] && [ "$resolved_target" = "$expected_resolved" ]
}

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

  if doctor_is_managed_command a47 "$AGENT47_HOME/bin/a47"; then
    echo "[OK] a47 in PATH"
  elif command -v a47 >/dev/null 2>&1; then
    echo "[WARN] a47 in PATH, but not the managed launcher from ~/bin"
    echo "[HINT] Fix: run ./install.sh"
  else
    echo "[WARN] a47 not in PATH"
    echo "[HINT] Fix: run ./install.sh"
  fi

  for script in "${INSTALLABLE_SCRIPTS[@]}"; do
    if doctor_is_managed_command "$script" "$AGENT47_HOME/scripts/$script"; then
      echo "[OK] $script available"
    elif command -v "$script" >/dev/null 2>&1; then
      echo "[WARN] $script in PATH, but not the managed copy from ~/bin"
      echo "[HINT] Fix: run ./install.sh"
    else
      echo "[WARN] $script missing"
      echo "[HINT] Fix: run ./install.sh"
    fi
  done

  for legacy in "${LEGACY_SCRIPTS[@]}"; do
    if command -v "$legacy" >/dev/null; then
      echo "[INFO] Legacy script detected: $legacy (remove old install)"
      echo "[HINT] Run ./install.sh to clean legacy scripts"
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
    echo "[HINT] Fix: run ./install.sh"
  fi

  if [ -d "$AGENT47_HOME/templates/skills" ]; then
    if ls "$AGENT47_HOME/templates/skills"/*.yml >/dev/null 2>&1; then
      echo "[WARN] Legacy .yml skills found in $AGENT47_HOME/templates/skills"
      echo "[HINT] Run ./install.sh to refresh .md skill templates"
    else
      echo "[OK] Skills templates (.md) present"
    fi
  fi

  if [ -x "$ROOT_DIR/tests/vendor/bats/bin/bats" ] || command -v bats >/dev/null 2>&1; then
    echo "[OK] bats available"
  else
    echo "[WARN] bats missing"
  fi

  if doctor_symlink_matches "$USER_DIR/a47" "$AGENT47_HOME/bin/a47"; then
    echo "[OK] a47 symlink present in ~/bin"
  elif [ -L "$USER_DIR/a47" ]; then
    echo "[WARN] a47 symlink in ~/bin is broken or points to a non-executable target"
    echo "[HINT] Fix: run ./install.sh"
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
