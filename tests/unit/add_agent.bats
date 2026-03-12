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
  assert_file_exists "prompts/agent-prompt.txt"
  assert_file_exists "README.md"
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
  run grep -F 'Use `AGENTS.md` as the single source of policy.' prompts/agent-prompt.txt
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
