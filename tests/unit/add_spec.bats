#!/usr/bin/env bats

load ../helpers/common

setup() {
  setup_workdir
}

teardown() {
  teardown_workdir
}

@test "add-spec creates spec from template" {
  run "$ROOT_DIR/scripts/add-spec"
  assert_success
  assert_file_exists "specs/spec.yml"
  run grep -q "id:" "specs/spec.yml"
  assert_success
}

@test "add-spec is idempotent and skips existing spec" {
  mkdir -p specs
  echo "existing" > specs/spec.yml

  run "$ROOT_DIR/scripts/add-spec"
  assert_success
  assert_contains "$output" "already exists"
  run cat specs/spec.yml
  assert_contains "$output" "existing"
}

@test "add-spec fails when template is missing" {
  rm "$AGENT47_HOME/templates/specs/spec.yml"

  run "$ROOT_DIR/scripts/add-spec"
  [ "$status" -ne 0 ]
  assert_contains "$output" "Template not found"
  [ ! -f "specs/spec.yml" ]

  # Restaurar templates
  rm -rf "$AGENT47_HOME/templates"
  cp -R "$ROOT_DIR/templates" "$AGENT47_HOME/"
}
