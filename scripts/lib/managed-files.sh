#!/bin/bash

# Shared path constants are consumed by sourced modules.
# shellcheck disable=SC2034

PROJECT_RULES_DIR="rules"
PROJECT_SKILLS_DIR="skills"
PROJECT_AGENTS_FILE="AGENTS.md"
PROJECT_README_FILE="README.md"
resolve_template_manifest() {
  if [ -n "${TEMPLATE_MANIFEST_FILE:-}" ] && [ -f "$TEMPLATE_MANIFEST_FILE" ]; then
    printf "%s\n" "$TEMPLATE_MANIFEST_FILE"
    return 0
  fi

  if [ -n "${TEMPLATES_DIR:-}" ] && [ -f "$TEMPLATES_DIR/manifest.txt" ]; then
    printf "%s\n" "$TEMPLATES_DIR/manifest.txt"
    return 0
  fi

  if [ -n "${ROOT_DIR:-}" ] && [ -f "$ROOT_DIR/templates/manifest.txt" ]; then
    printf "%s\n" "$ROOT_DIR/templates/manifest.txt"
    return 0
  fi

  return 1
}

manifest_read_section() {
  local section="$1"
  local manifest_file

  manifest_file="$(resolve_template_manifest)" || {
    echo "[ERR] Missing manifest file" >&2
    return 1
  }

  awk -v target="[$section]" '
    $0 == target { in_section=1; next }
    /^\[/ && in_section { exit }
    in_section && NF { print }
  ' "$manifest_file"
}

project_rule_template_files() {
  manifest_read_section "rule_templates"
}

project_managed_targets() {
  manifest_read_section "managed_targets"
}

project_preserved_files() {
  manifest_read_section "preserved_targets"
}

project_required_template_files() {
  manifest_read_section "required_template_files"
}

project_required_template_dirs() {
  manifest_read_section "required_template_dirs"
}

manifest_contains_entry() {
  local section="$1"
  local expected="$2"
  local entry

  while IFS= read -r entry; do
    [ -n "$entry" ] || continue
    if [ "$entry" = "$expected" ]; then
      return 0
    fi
  done < <(manifest_read_section "$section")

  return 1
}

assert_manifest_contract() {
  local section
  local required_sections=(
    managed_targets
    preserved_targets
    rule_templates
    required_template_files
    required_template_dirs
  )
  local entry_count

  for section in "${required_sections[@]}"; do
    entry_count="$(manifest_read_section "$section" | awk 'NF { count += 1 } END { print count + 0 }')" || return 1
    if [ "$entry_count" -eq 0 ]; then
      echo "[ERR] Manifest section has no entries: $section" >&2
      return 1
    fi
  done
}
