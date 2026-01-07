#!/bin/bash

validate_skill() {
  local skill_path="$1"
  local fm name desc

  if [ ! -f "$skill_path" ]; then
    echo "⚠️  SKILL.md missing at $skill_path"
    return 1
  fi

  fm="$(awk 'BEGIN{in=0} /^---[ \t]*$/{if(in){exit}; in=1; next} in{print}' "$skill_path")"
  name="$(printf "%s\n" "$fm" | sed -n 's/^name:[[:space:]]*//p' | head -n 1)"
  desc="$(printf "%s\n" "$fm" | sed -n 's/^description:[[:space:]]*//p' | head -n 1)"

  if [ -z "$name" ] || [ -z "$desc" ]; then
    echo "⚠️  Invalid frontmatter in $skill_path (name/description required)"
    return 1
  fi

  if ! printf "%s" "$name" | grep -Eq '^[a-z0-9]+(-[a-z0-9]+)*$'; then
    echo "⚠️  Skill name not kebab-case in $skill_path (got '$name')"
    return 1
  fi

  if [ "${#name}" -gt 64 ]; then
    echo "⚠️  Skill name too long in $skill_path (${#name} chars)"
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
      fm="$(awk 'BEGIN{in=0} /^---[ \t]*$/{if(in){exit}; in=1; next} in{print}' "$skill_path")"
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

  echo "✅ Generated $output_file"
}
