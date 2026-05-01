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
  assert_file_exists "rules/security-global.yaml"
  assert_file_exists "rules/security-shell.yaml"
  assert_file_exists "skills/analyze/SKILL.md"
  assert_file_exists "skills/review/SKILL.md"
  assert_file_exists "skills/AVAILABLE_SKILLS.xml"
  assert_file_exists "skills/AVAILABLE_SKILLS.json"
  assert_file_exists "skills/SUMMARY.md"
  assert_file_exists "README.md"
  assert_file_exists "specs/spec.yml"
  [ ! -d "prompts" ]
  [ ! -f "rules/rules-backend.yaml" ]
}

@test "add-agent --only-skills instala solo skills" {
  run "$ROOT_DIR/bin/afs" add-agent --only-skills
  assert_success
  [ ! -f "AGENTS.md" ]
  [ ! -d "rules" ]
  assert_file_exists "skills/analyze/SKILL.md"
  assert_file_exists "skills/AVAILABLE_SKILLS.xml"
  assert_file_exists "skills/AVAILABLE_SKILLS.json"
  assert_file_exists "skills/SUMMARY.md"
}

@test "add-agent --only-skills --preview no escribe archivos" {
  run "$ROOT_DIR/bin/afs" add-agent --only-skills --preview
  assert_success
  assert_contains "$output" "mode: only-skills"
  [ ! -d "skills" ]
  [ ! -f "AGENTS.md" ]
  [ ! -d "rules" ]
}

@test "add-agent --only-skills respeta bundles explicitos" {
  run "$ROOT_DIR/bin/afs" add-agent --only-skills --bundle cli --yes
  assert_success
  assert_file_exists "skills/cli-design/SKILL.md"
  [ ! -d "skills/optimize" ]
  [ ! -d "skills/refactor" ]
  [ ! -f "AGENTS.md" ]
  [ ! -d "rules" ]
}

@test "add-agent --only-skills ignores unselected discovered skills without breaking indexes" {
  temp_repo="$(make_test_repo_copy)"
  mkdir -p "$temp_repo/templates/base/skills/custom-skill"
  cat > "$temp_repo/templates/base/skills/custom-skill/SKILL.md" <<'EOF'
---
name: custom-skill
description: Dynamic test skill.
---

# Custom Skill
EOF

  run env AGENT47_REPO_ROOT="$temp_repo" "$ROOT_DIR/bin/afs" add-agent --only-skills
  assert_success
  [ ! -d "skills/custom-skill" ]
  assert_file_exists "skills/AVAILABLE_SKILLS.xml"
  assert_file_exists "skills/AVAILABLE_SKILLS.json"
  assert_file_exists "skills/SUMMARY.md"
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
  [ ! -f "rules/rules-backend.yaml" ]
  run grep -F "Never hardcode secrets" rules/security-global.yaml
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
  assert_file_exists "skills/AVAILABLE_SKILLS.json"
  assert_file_exists "skills/SUMMARY.md"
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

@test "add-agent --only-skills --force preview refleja reemplazo real de skills" {
  mkdir -p skills/analyze skills/custom-skill
  echo "custom" > skills/analyze/SKILL.md
  echo "remove me" > skills/notes.txt
  cat > skills/custom-skill/SKILL.md <<'EOF'
---
name: custom-skill
description: Local custom skill.
---
EOF

  run "$ROOT_DIR/bin/afs" add-agent --only-skills --force --preview
  assert_success
  assert_contains "$output" "update:"
  assert_contains "$output" "skills/"
  assert_contains "$output" "skills/analyze/"
  assert_contains "$output" "remove on --force:"
  assert_contains "$output" "skills/notes.txt"
  assert_contains "$output" "skills/custom-skill"
}

@test "add-agent --force removes stale managed yaml rules" {
  mkdir -p rules
  echo "stale managed rule" > rules/custom-rule.yaml
  echo "keep me" > rules/custom.txt

  run "$ROOT_DIR/bin/afs" add-agent --force
  assert_success
  [ ! -f "rules/custom-rule.yaml" ]
  assert_file_exists "rules/security-global.yaml"
  run cat "rules/custom.txt"
  assert_success
  [ "$output" = "keep me" ]
}

@test "add-agent --force migrates a legacy scaffold while preserving project files" {
  mkdir -p rules prompts specs skills/custom
  echo "old agents" > AGENTS.md
  echo "legacy managed rule" > rules/legacy.yaml
  echo "keep me" > rules/custom.txt
  cat > skills/custom/SKILL.md <<'EOF'
---
name: custom
description: Local custom skill.
---
EOF
  echo "existing prompt" > prompts/agent-prompt.txt
  echo "existing readme" > README.md
  echo "existing spec" > specs/spec.yml

  run "$ROOT_DIR/bin/afs" add-agent --force --yes
  assert_success
  [ ! -f "rules/legacy.yaml" ]
  assert_file_exists "rules/security-global.yaml"
  run grep -F "existing prompt" prompts/agent-prompt.txt
  assert_success
  [ ! -d "skills/custom" ]
  run grep -F "existing readme" README.md
  assert_success
  run grep -F "existing spec" specs/spec.yml
  assert_success
  run grep -F "keep me" rules/custom.txt
  assert_success
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

@test "add-agent --preview shows the planned base bundle without writing files" {
  run "$ROOT_DIR/bin/afs" add-agent --preview
  assert_success
  assert_contains "$output" "Preview"
  assert_contains "$output" "bundles: base"
  [ ! -f "AGENTS.md" ]
  [ ! -d "rules" ]
}

@test "add-agent --dry-run does not write files" {
  run "$ROOT_DIR/bin/afs" add-agent --dry-run
  assert_success
  assert_contains "$output" "Preview"
  [ ! -f "AGENTS.md" ]
}

@test "add-agent preview warns on unresolved conflict and falls back to base bundle" {
  mkdir -p src api
  echo '{"dependencies":{"react":"1.0.0","express":"1.0.0"}}' > package.json

  run "$ROOT_DIR/bin/afs" add-agent --preview
  assert_success
  assert_contains "$output" "Multiple project types detected with no supported automatic composition"
  assert_contains "$output" "bundles: base"
}

@test "add-agent rejects incompatible explicit bundles" {
  run "$ROOT_DIR/bin/afs" add-agent --preview --bundle frontend --bundle backend
  [ "$status" -ne 0 ]
  assert_contains "$output" "explicit bundle selection is incompatible"
}

@test "add-agent accepts compatible explicit bundles" {
  run "$ROOT_DIR/bin/afs" add-agent --preview --bundle cli --bundle scripts
  assert_success
  assert_contains "$output" "bundles: base, project-cli, project-scripts, shared-cli-behavior, shared-testing"
}

@test "add-agent fails if a required template is missing" {
  temp_repo="$(make_test_repo_copy)"
  mv "$temp_repo/templates/base/AGENTS.md" "$temp_repo/templates/base/AGENTS.md.bak"

  run env AGENT47_REPO_ROOT="$temp_repo" "$ROOT_DIR/bin/afs" add-agent
  [ "$status" -ne 0 ]
  assert_contains "$output" "Template not found"
  assert_contains "$output" "Restore the missing template asset"
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
    for file in "$1"/templates/base/skills/*/SKILL.md; do
      mv "$file" "$file.bak"
    done
    for file in "$1"/templates/bundles/*/skills/*/SKILL.md; do
      [ -e "$file" ] || continue
      mv "$file" "$file.bak"
    done
    trap '"'"'
      for file in "$1"/templates/base/skills/*/SKILL.md.bak; do
        [ -e "$file" ] || continue
        mv "$file" "${file%.bak}"
      done
      for file in "$1"/templates/bundles/*/skills/*/SKILL.md.bak; do
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
    cp "$1/templates/base/skills/analyze/SKILL.md" "$1/templates/base/skills/analyze/SKILL.md.bak"
    trap '"'"'mv "$1/templates/base/skills/analyze/SKILL.md.bak" "$1/templates/base/skills/analyze/SKILL.md"'"'"' EXIT
    printf "%s\n" not-valid-frontmatter > "$1/templates/base/skills/analyze/SKILL.md"
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
    cp "$1/templates/base/skills/analyze/SKILL.md" "$1/templates/base/skills/analyze/SKILL.md.bak"
    trap '"'"'mv "$1/templates/base/skills/analyze/SKILL.md.bak" "$1/templates/base/skills/analyze/SKILL.md"'"'"' EXIT
    printf "%s\n" not-valid-frontmatter > "$1/templates/base/skills/analyze/SKILL.md"
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
