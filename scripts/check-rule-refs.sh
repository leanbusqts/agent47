#!/usr/bin/env bash
set -Eeuo pipefail

cd "$(git rev-parse --show-toplevel 2>/dev/null || pwd)"

all_ids="$(grep -hE '^  - id: ' rules/*.yaml | sed -E 's/.*"([^"]+)".*/\1/' | sort -u)"
[ -n "$all_ids" ] || { echo "ERROR: no rule IDs found"; exit 2; }

err=0
while IFS=: read -r src ref; do
  case "$ref" in
    *.md|*#*) continue ;;
  esac
  if ! grep -qx "$ref" <<<"$all_ids"; then
    echo "ERROR: $src refs missing ID $ref"
    err=1
  fi
done < <(python3 - <<'PY'
from pathlib import Path
for path in sorted(Path("rules").glob("*.yaml")):
    current = None
    in_refs = False
    for line in path.read_text(encoding="utf-8").splitlines():
        if line.startswith("  - id: "):
            current = line.split('"', 2)[1]
            in_refs = False
        elif current and line.startswith("    refs:"):
            in_refs = True
            rest = line.split(":", 1)[1]
            for ref in rest.replace("[", "").replace("]", "").split(","):
                ref = ref.strip().strip('"')
                if ref:
                    print(f"{current}:{ref}")
        elif in_refs and line.startswith("      - "):
            ref = line.split('"', 2)[1] if '"' in line else line.split("-", 1)[1].strip()
            print(f"{current}:{ref}")
        elif line.startswith("    ") and not line.startswith("      "):
            in_refs = False
PY
)

[ "$err" -eq 0 ] && echo "OK: all refs resolve."
exit "$err"
