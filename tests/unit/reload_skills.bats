#!/usr/bin/env bats

load ../helpers/common

setup() {
  setup_workdir
}

teardown() {
  teardown_workdir
}

@test "reload-skills no-ops with warning when skills dir missing" {
  run "$ROOT_DIR/scripts/reload-skills"
  [ "$status" -eq 0 ]
  assert_contains "$output" "[WARN]"
}

@test "reload-skills regenerates AVAILABLE_SKILLS.xml" {
  run "$ROOT_DIR/scripts/add-skills"
  assert_success
  assert_file_exists "skills/AVAILABLE_SKILLS.xml"
  rm "skills/AVAILABLE_SKILLS.xml"

  run "$ROOT_DIR/scripts/reload-skills"
  assert_success
  assert_file_exists "skills/AVAILABLE_SKILLS.xml"
  run grep -q "<name>implement</name>" "skills/AVAILABLE_SKILLS.xml"
  assert_success
}
