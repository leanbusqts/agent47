#!/bin/bash

required_agents_sections() {
  printf "%s\n" "${REQUIRED_AGENTS_SECTIONS[@]}"
}

check_agents_sections() {
  local agents_file="$1"
  local missing=0

  if [ ! -f "$agents_file" ]; then
    echo "[WARN] AGENTS.md missing"
    return 1
  fi

  while IFS= read -r section; do
    if [ -n "$section" ] && ! grep -Fq "$section" "$agents_file"; then
      echo "[WARN] AGENTS missing section: $section"
      missing=1
    fi
  done < <(required_agents_sections)

  if [ "$missing" -eq 0 ]; then
    echo "[OK] AGENTS required sections present"
    return 0
  fi

  return 1
}

check_security_templates() {
  local templates_dir="$1"
  local missing=0

  for file in "${SECURITY_TEMPLATE_FILES[@]}"; do
    if [ ! -f "$templates_dir/rules/$file" ]; then
      echo "[WARN] Missing security template: rules/$file"
      missing=1
    fi
  done

  if [ "$missing" -eq 0 ]; then
    echo "[OK] Security templates present"
    return 0
  fi

  return 1
}

check_prompt_template() {
  local templates_dir="$1"

  if [ -f "$templates_dir/prompts/agent-prompt.txt" ]; then
    echo "[OK] Prompt template present"
  else
    echo "[WARN] Prompt template missing"
    return 1
  fi
}

check_security_rule_ids() {
  local templates_dir="$1"
  local security_dir="$templates_dir/rules"
  local dupes

  if [ ! -d "$security_dir" ]; then
    echo "[WARN] Security rules directory missing"
    return 1
  fi

  dupes="$(grep -ho 'id:[[:space:]]*"SEC-[^"]*"' "$security_dir"/security-*.yaml 2>/dev/null | sed -E 's/.*"(SEC-[^"]*)"/\1/' | sort | uniq -d)"
  if [ -n "$dupes" ]; then
    echo "[WARN] Duplicate security rule IDs detected"
    echo "$dupes"
    return 1
  fi

  echo "[OK] Security rule IDs unique"
  return 0
}
