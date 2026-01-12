#!/bin/bash

validate_skill() {
  local skill_path="$1"
  local fm name desc

  if [ ! -f "$skill_path" ]; then
    echo "[WARN] SKILL.md missing at $skill_path" >&2
    return 1
  fi

  fm="$(sed -n '1,/^---[[:space:]]*$/p' "$skill_path" | sed '1d;$d')"
  name="$(printf "%s\n" "$fm" | sed -n 's/^name:[[:space:]]*//p' | head -n 1)"
  desc="$(printf "%s\n" "$fm" | sed -n 's/^description:[[:space:]]*//p' | head -n 1)"

  if [ -z "$name" ] || [ -z "$desc" ]; then
    echo "[WARN] Invalid frontmatter in $skill_path (name/description required)" >&2
    return 1
  fi

  if ! printf "%s" "$name" | grep -Eq '^[a-z0-9]+(-[a-z0-9]+)*$'; then
    echo "[WARN] Skill name not kebab-case in $skill_path (got '$name')" >&2
    return 1
  fi

  if [ "${#name}" -gt 64 ]; then
    echo "[WARN] Skill name too long in $skill_path (${#name} chars)" >&2
    return 1
  fi

  return 0
}

write_available_skills() {
  local skills_dir="$1"
  local output_file="$2"

  mkdir -p "$skills_dir"

  {
    echo "<available_skills>"
    while IFS= read -r skill_path; do
      if ! validate_skill "$skill_path"; then
        continue
      fi
      fm="$(sed -n '1,/^---[[:space:]]*$/p' "$skill_path" | sed '1d;$d')"
      name="$(printf "%s\n" "$fm" | sed -n 's/^name:[[:space:]]*//p' | head -n 1)"
      desc="$(printf "%s\n" "$fm" | sed -n 's/^description:[[:space:]]*//p' | head -n 1)"
      echo "  <skill>"
      echo "    <name>$name</name>"
      echo "    <description>$desc</description>"
      echo "    <location>$(dirname "$skill_path")/SKILL.md</location>"
      echo "  </skill>"
    done < <(find "$skills_dir" -maxdepth 2 -name "SKILL.md" | sort)
    echo "</available_skills>"
  } >"$output_file"

  echo "[OK] Generated $output_file"
}
