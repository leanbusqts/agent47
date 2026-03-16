#!/usr/bin/env bats

load ../helpers/common

setup() {
  setup_workdir
}

teardown() {
  teardown_workdir
}

@test "add-agent copies core files and creates README" {
  run "$ROOT_DIR/scripts/add-agent"
  assert_success
  assert_file_exists "AGENTS.md"
  assert_file_exists "rules/rules-mobile.yaml"
  assert_file_exists "rules/rules-frontend.yaml"
  assert_file_exists "rules/rules-backend.yaml"
  assert_file_exists "rules/security-global.yaml"
  assert_file_exists "rules/security-js-ts.yaml"
  assert_file_exists "rules/security-py.yaml"
  assert_file_exists "rules/security-java-kotlin.yaml"
  assert_file_exists "rules/security-swift.yaml"
  assert_file_exists "rules/security-csharp.yaml"
  assert_file_exists "skills/analyze/SKILL.md"
  assert_file_exists "skills/AVAILABLE_SKILLS.xml"
  assert_file_exists "README.md"
}

@test "add-agent --only-skills instala solo skills" {
  run "$ROOT_DIR/scripts/add-agent" --only-skills
  assert_success
  [ ! -f "AGENTS.md" ]
  [ ! -d "rules" ]
  assert_file_exists "skills/analyze/SKILL.md"
  assert_file_exists "skills/AVAILABLE_SKILLS.xml"
}

@test "add-agent discovers skill templates dynamically" {
  mkdir -p "$AGENT47_HOME/templates/skills/custom-skill"
  cat > "$AGENT47_HOME/templates/skills/custom-skill/SKILL.md" <<'EOF'
---
name: custom-skill
description: Dynamic test skill.
---

# Custom Skill
EOF

  run "$ROOT_DIR/scripts/add-agent" --only-skills
  assert_success
  assert_file_exists "skills/custom-skill/SKILL.md"
  run grep -F "<name>custom-skill</name>" skills/AVAILABLE_SKILLS.xml
  assert_success
}

@test "add-agent --force updates managed files and preserves user project files" {
  mkdir -p rules prompts specs skills/analyze
  echo "old agents" > AGENTS.md
  echo "old rule" > rules/rules-backend.yaml
  echo "old prompt" > prompts/agent-prompt.txt
  echo "custom spec" > specs/spec.yml
  echo "custom readme" > README.md
  echo "custom snapshot" > SNAPSHOT.md
  echo "custom skill" > skills/analyze/SKILL.md

  run "$ROOT_DIR/scripts/add-agent" --force
  assert_success

  run grep -F "single source of operating policy" AGENTS.md
  assert_success
  run grep -F "Controllers and transport adapters handle transport concerns only" rules/rules-backend.yaml
  assert_success
  run grep -F "old prompt" prompts/agent-prompt.txt
  assert_success
  run grep -F "custom spec" specs/spec.yml
  assert_success
  run grep -F "custom readme" README.md
  assert_success
  run grep -F "custom snapshot" SNAPSHOT.md
  assert_success
  run grep -F "name: analyze" skills/analyze/SKILL.md
  assert_success
}

@test "add-agent --only-skills --force refresca solo skills" {
  mkdir -p skills/analyze rules
  echo "custom" > skills/analyze/SKILL.md
  echo "keep rule" > rules/rules-backend.yaml

  run "$ROOT_DIR/scripts/add-agent" --only-skills --force
  assert_success
  run grep -q "custom" skills/analyze/SKILL.md
  [ "$status" -ne 0 ]
  run grep -q "keep rule" rules/rules-backend.yaml
  assert_success
}

@test "add-agent --force removes stale managed rules" {
  mkdir -p rules
  echo "stale rule" > rules/obsolete-managed.yaml
  echo "keep me" > rules/custom.txt

  run "$ROOT_DIR/scripts/add-agent" --force
  assert_success
  [ ! -f "rules/obsolete-managed.yaml" ]
  assert_file_exists "rules/rules-backend.yaml"
  run cat "rules/custom.txt"
  assert_success
  [ "$output" = "keep me" ]
}

@test "add-agent fails if a required template is missing" {
  rm "$AGENT47_HOME/templates/AGENTS.md"

  run "$ROOT_DIR/scripts/add-agent"
  [ "$status" -ne 0 ]
  assert_contains "$output" "Template not found"
  assert_contains "$output" "required templates missing"
  [ ! -f "AGENTS.md" ]
  [ ! -f "rules/rules-backend.yaml" ]
  [ ! -f "rules/rules-frontend.yaml" ]
  [ ! -f "rules/rules-mobile.yaml" ]
  [ ! -f "rules/security-global.yaml" ]

  # Restaurar templates
  rm -rf "$AGENT47_HOME/templates"
  cp -R "$ROOT_DIR/templates" "$AGENT47_HOME/"
}

@test "add-agent aborts before writing when skills helper dependencies are missing" {
  run bash -c '
    set -euo pipefail
    rm -f "$1/scripts/lib/skill-utils.sh"
    mv "$2/scripts/lib/skill-utils.sh" "$2/scripts/lib/skill-utils.sh.bak"
    trap '"'"'
      mv "$2/scripts/lib/skill-utils.sh.bak" "$2/scripts/lib/skill-utils.sh"
      cp "$2/scripts/lib/skill-utils.sh" "$1/scripts/lib/skill-utils.sh"
    '"'"' EXIT
    cd "$3"
    "$2/scripts/add-agent"
  ' _ "$AGENT47_HOME" "$ROOT_DIR" "$PWD"
  [ "$status" -ne 0 ]
  assert_contains "$output" "missing helper dependency"
  [ ! -f "AGENTS.md" ]
  [ ! -f "README.md" ]
  [ ! -d "rules" ]
}

@test "add-agent falls back to local skill utils when installed copy is missing" {
  rm -f "$AGENT47_HOME/scripts/lib/skill-utils.sh"
  mkdir -p "$AGENT47_HOME/scripts"

  run "$ROOT_DIR/scripts/add-agent" --only-skills
  assert_success
  assert_file_exists "skills/analyze/SKILL.md"
  assert_file_exists "skills/AVAILABLE_SKILLS.xml"
}

@test "add-agent aborts when no valid skill templates are available" {
  rm -rf "$AGENT47_HOME/templates/skills"
  mkdir -p "$AGENT47_HOME/templates/skills"

  run "$ROOT_DIR/scripts/add-agent" --only-skills
  [ "$status" -ne 0 ]
  assert_contains "$output" "No valid skill templates found"
  [ ! -d "skills" ]

  rm -rf "$AGENT47_HOME/templates/skills"
  cp -R "$ROOT_DIR/templates/skills" "$AGENT47_HOME/templates/"
}

@test "add-agent --force rolls back if staged skills are invalid" {
  mkdir -p rules skills/analyze "$AGENT47_HOME/templates/skills/analyze"
  echo "existing agents" > AGENTS.md
  echo "existing rule" > rules/rules-backend.yaml
  echo "existing skill" > skills/analyze/SKILL.md
  printf '%s\n' 'not-valid-frontmatter' > "$AGENT47_HOME/templates/skills/analyze/SKILL.md"

  run "$ROOT_DIR/scripts/add-agent" --force
  [ "$status" -ne 0 ]
  run grep -F "existing agents" AGENTS.md
  assert_success
  run grep -F "existing rule" rules/rules-backend.yaml
  assert_success
  run grep -F "existing skill" skills/analyze/SKILL.md
  assert_success

  cp "$ROOT_DIR/templates/skills/analyze/SKILL.md" "$AGENT47_HOME/templates/skills/analyze/SKILL.md"
}

@test "add-agent --force cleans empty rules dir after rollback on fresh repo" {
  mkdir -p "$AGENT47_HOME/templates/skills/analyze"
  printf '%s\n' 'not-valid-frontmatter' > "$AGENT47_HOME/templates/skills/analyze/SKILL.md"

  run "$ROOT_DIR/scripts/add-agent" --force
  [ "$status" -ne 0 ]
  [ ! -d "rules" ]
  [ ! -f "AGENTS.md" ]
  [ ! -f "README.md" ]

  cp "$ROOT_DIR/templates/skills/analyze/SKILL.md" "$AGENT47_HOME/templates/skills/analyze/SKILL.md"
}
