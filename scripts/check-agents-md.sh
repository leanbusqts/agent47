#!/usr/bin/env bash
# Validate AGENTS.md against the agent policy contract.
set -Eeuo pipefail

cd "$(git rev-parse --show-toplevel 2>/dev/null || pwd)"

err=0

fail() {
  printf 'ERROR: %s\n' "$*" >&2
  err=1
}

warn() {
  printf 'WARN: %s\n' "$*" >&2
}

exists_ref() {
  local ref="$1"
  case "$ref" in
    *'*'* | *'<'*'>'* | SKILL.md | claude.md | .cursorrules | .codex/config.toml | .windsurf*) return 0 ;;
  esac
  [ -e "$ref" ] && return 0
  [ -e "rules/$ref" ] && return 0
  [ -e "templates/base/rules/$ref" ] && return 0
  [ -e "skills/$ref" ] && return 0
  [ -e "skills/$ref.md" ] && return 0
  [ -e "skills/$ref.json" ] && return 0
  [ -e "templates/manifest.txt" ] && [ "$ref" = "manifest.txt" ] && return 0
  return 1
}

if ! command -v python3 >/dev/null 2>&1; then
  fail "python3 is required for metadata validation."
fi

if ! diff -q AGENTS.md templates/base/AGENTS.md >/dev/null 2>&1; then
  fail "AGENTS.md and templates/base/AGENTS.md differ."
fi

lines=$(wc -l < AGENTS.md | tr -d ' ')
if [ "$lines" -gt 200 ]; then
  fail "AGENTS.md is $lines lines, hard cap is 200."
elif [ "$lines" -gt 170 ]; then
  warn "AGENTS.md is $lines lines (target ~162, soft cap 170)."
fi

if command -v python3 >/dev/null 2>&1; then
  if ! python3 - <<'PY'
import re
from pathlib import Path

text = Path("AGENTS.md").read_text(encoding="utf-8")
match = re.search(r"^<!--\n(agents-md:\s*\{[^}]+\})\n-->", text, re.M)
if not match:
    raise SystemExit(1)
body = match.group(1)
required = [
    "version",
    "last_updated",
    "schema_version",
    "owner",
    "review_cadence",
    "applies_to",
    "mirror",
    "max_lines",
    "target_lines",
]
for key in required:
    if not re.search(rf"\b{re.escape(key)}\s*:", body):
        raise SystemExit(1)
date_match = re.search(r'last_updated:\s*"(\d{4}-\d{2}-\d{2})"', body)
if not date_match:
    raise SystemExit(1)
PY
  then
    fail "metadata header missing required agents-md keys or ISO last_updated."
  fi
fi

required_sections=(
  "Purpose"
  "Scope"
  "Glossary"
  "Authority Order"
  "Required Inputs"
  "Executable Commands"
  "Context Efficiency"
  "Execution"
  "Filesystem And Approval Boundaries"
  "Approval And Severity"
  "Security Expectations"
  "Dependency Policy"
  "Stack Notes"
  "Skills"
  "Output Expectations"
  "Verification And Rollback"
  "Git And Commits"
  "Communication Conventions"
  "Maintenance Of This File"
)

for section in "${required_sections[@]}"; do
  if ! grep -qE "^## ${section}$" AGENTS.md; then
    fail "required section '## ${section}' missing."
  fi
done

dupes=$(grep -oE '\[AG-[0-9]{3}\]' AGENTS.md | sort | uniq -d || true)
if [ -n "$dupes" ]; then
  fail "duplicate AG declarations: $dupes"
fi

section_count=$(grep -cE '^## ' AGENTS.md || true)
if [ "$section_count" -ne 19 ]; then
  fail "expected 19 section headings, found $section_count."
fi

# shellcheck disable=SC2016
while IFS= read -r raw; do
  ref=${raw#\`}
  ref=${ref%\`}
  if ! exists_ref "$ref"; then
    warn "referenced path '$ref' not found."
  fi
done < <(grep -oE '`[A-Za-z0-9_./<>*-]+\.(md|yaml|txt)`' AGENTS.md | sort -u || true)

if [ "$err" -eq 0 ]; then
  printf 'OK: AGENTS.md (%s lines) passes the contract.\n' "$lines"
fi
exit "$err"
