#!/usr/bin/env bats

load ../helpers/common

setup() {
  setup_workdir
}

teardown() {
  teardown_workdir
}

@test "add-agent copies core files and creates README" {
  run "$ROOT_DIR/bin/afs" add-agent
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
  assert_file_exists "specs/spec.yml"
}

@test "add-agent --only-skills instala solo skills" {
  run "$ROOT_DIR/bin/afs" add-agent --only-skills
  assert_success
  [ ! -f "AGENTS.md" ]
  [ ! -d "rules" ]
  assert_file_exists "skills/analyze/SKILL.md"
  assert_file_exists "skills/AVAILABLE_SKILLS.xml"
}

@test "add-agent discovers skill templates dynamically" {
  temp_repo="$(make_test_repo_copy)"
  mkdir -p "$temp_repo/templates/skills/custom-skill"
  cat > "$temp_repo/templates/skills/custom-skill/SKILL.md" <<'EOF'
---
name: custom-skill
description: Dynamic test skill.
---

# Custom Skill
EOF

  run env AGENT47_REPO_ROOT="$temp_repo" "$ROOT_DIR/bin/afs" add-agent --only-skills
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
  echo "custom product spec" > SPEC.md
  echo "custom skill" > skills/analyze/SKILL.md

  run "$ROOT_DIR/bin/afs" add-agent --force
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
  run grep -F "custom product spec" SPEC.md
  assert_success
  run grep -F "name: analyze" skills/analyze/SKILL.md
  assert_success
}

@test "add-agent --only-skills --force refresca solo skills" {
  mkdir -p skills/analyze rules
  echo "custom" > skills/analyze/SKILL.md
  echo "keep rule" > rules/rules-backend.yaml

  run "$ROOT_DIR/bin/afs" add-agent --only-skills --force
  assert_success
  run grep -q "custom" skills/analyze/SKILL.md
  [ "$status" -ne 0 ]
  run grep -q "keep rule" rules/rules-backend.yaml
  assert_success
}

@test "add-agent --force removes stale managed yaml rules" {
  mkdir -p rules
  echo "stale managed rule" > rules/custom-rule.yaml
  echo "keep me" > rules/custom.txt

  run "$ROOT_DIR/bin/afs" add-agent --force
  assert_success
  [ ! -f "rules/custom-rule.yaml" ]
  assert_file_exists "rules/rules-backend.yaml"
  run cat "rules/custom.txt"
  assert_success
  [ "$output" = "keep me" ]
}

@test "add-agent --force replaces skills dir with fresh managed install" {
  mkdir -p skills/custom-skill
  cat > skills/custom-skill/SKILL.md <<'EOF'
---
name: custom-skill
description: Local custom skill.
---

# Local Custom Skill
EOF

  run "$ROOT_DIR/bin/afs" add-agent --force
  assert_success
  [ ! -d "skills/custom-skill" ]
  assert_file_exists "skills/analyze/SKILL.md"
}

@test "add-agent fails if a required template is missing" {
  temp_repo="$(make_test_repo_copy)"
  mv "$temp_repo/templates/AGENTS.md" "$temp_repo/templates/AGENTS.md.bak"

  run env AGENT47_REPO_ROOT="$temp_repo" "$ROOT_DIR/bin/afs" add-agent
  [ "$status" -ne 0 ]
  assert_contains "$output" "Template not found"
  assert_contains "$output" "required templates missing"
  [ ! -f "AGENTS.md" ]
  [ ! -f "rules/rules-backend.yaml" ]
  [ ! -f "rules/rules-frontend.yaml" ]
  [ ! -f "rules/rules-mobile.yaml" ]
  [ ! -f "rules/security-global.yaml" ]
}

@test "add-agent aborts when no valid skill templates are available" {
  temp_repo="$(make_test_repo_copy)"
  run bash -c '
    set -euo pipefail
    for file in "$1"/templates/skills/*/SKILL.md; do
      mv "$file" "$file.bak"
    done
    trap '"'"'
      for file in "$1"/templates/skills/*/SKILL.md.bak; do
        [ -e "$file" ] || continue
        mv "$file" "${file%.bak}"
      done
    '"'"' EXIT
    cd "$2"
    AGENT47_REPO_ROOT="$1" "$3/bin/afs" add-agent --only-skills
  ' _ "$temp_repo" "$PWD" "$ROOT_DIR"
  [ "$status" -ne 0 ]
  assert_contains "$output" "no valid skill templates found in skills"
  [ ! -d "skills" ]
}

@test "add-agent --force rolls back if staged skills are invalid" {
  temp_repo="$(make_test_repo_copy)"
  mkdir -p rules skills/analyze
  echo "existing agents" > AGENTS.md
  echo "existing rule" > rules/rules-backend.yaml
  echo "existing skill" > skills/analyze/SKILL.md
  run bash -c '
    set -euo pipefail
    cp "$1/templates/skills/analyze/SKILL.md" "$1/templates/skills/analyze/SKILL.md.bak"
    trap '"'"'mv "$1/templates/skills/analyze/SKILL.md.bak" "$1/templates/skills/analyze/SKILL.md"'"'"' EXIT
    printf "%s\n" not-valid-frontmatter > "$1/templates/skills/analyze/SKILL.md"
    cd "$2"
    AGENT47_REPO_ROOT="$1" "$3/bin/afs" add-agent --force
  ' _ "$temp_repo" "$PWD" "$ROOT_DIR"
  [ "$status" -ne 0 ]
  run grep -F "existing agents" AGENTS.md
  assert_success
  run grep -F "existing rule" rules/rules-backend.yaml
  assert_success
  run grep -F "existing skill" skills/analyze/SKILL.md
  assert_success
}

@test "add-agent --force cleans empty rules dir after rollback on fresh repo" {
  temp_repo="$(make_test_repo_copy)"
  run bash -c '
    set -euo pipefail
    cp "$1/templates/skills/analyze/SKILL.md" "$1/templates/skills/analyze/SKILL.md.bak"
    trap '"'"'mv "$1/templates/skills/analyze/SKILL.md.bak" "$1/templates/skills/analyze/SKILL.md"'"'"' EXIT
    printf "%s\n" not-valid-frontmatter > "$1/templates/skills/analyze/SKILL.md"
    cd "$2"
    AGENT47_REPO_ROOT="$1" "$3/bin/afs" add-agent --force
  ' _ "$temp_repo" "$PWD" "$ROOT_DIR"
  [ "$status" -ne 0 ]
  [ ! -d "rules" ]
  [ ! -f "AGENTS.md" ]
  [ ! -f "README.md" ]
}

@test "add-agent preserves preexisting skills file on failure" {
  echo "do-not-delete" > skills

  run "$ROOT_DIR/bin/afs" add-agent
  [ "$status" -ne 0 ]
  run cat skills
  assert_success
  [ "$output" = "do-not-delete" ]
}
