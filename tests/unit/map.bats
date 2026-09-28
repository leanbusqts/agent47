#!/usr/bin/env bats

load ../helpers/common

setup() {
  setup_workdir
  mkdir -p cmd/tool
  printf '%s\n' 'module example.com/project' > go.mod
  printf '%s\n' 'package main' 'func main() {}' > cmd/tool/main.go
  printf '%s\n' 'test:' ' go test ./...' > Makefile
}

teardown() {
  teardown_workdir
}

@test "afs map creates and keeps a bounded repository context" {
  run "$ROOT_DIR/bin/afs" map
  assert_success
  assert_contains "$output" "Created .agent47/context.md"
  assert_file_exists ".agent47/context.md"
  run grep -F '<!-- afs-context' .agent47/context.md
  assert_success
  run grep -F '## Components' .agent47/context.md
  assert_success

  run "$ROOT_DIR/bin/afs" map
  assert_success
  assert_contains "$output" ".agent47/context.md is current"
}

@test "afs map updates structural changes and protects manual edits" {
  run "$ROOT_DIR/bin/afs" map
  assert_success
  mkdir -p internal/service
  printf '%s\n' 'package service' > internal/service/service.go

  run "$ROOT_DIR/bin/afs" map
  assert_success
  assert_contains "$output" "Updated .agent47/context.md"

  printf '%s\n' 'manual edit' >> .agent47/context.md
  run "$ROOT_DIR/bin/afs" map
  [ "$status" -eq 1 ]
  assert_contains "$output" "rerun with --force"

  run "$ROOT_DIR/bin/afs" map --force
  assert_success
  assert_contains "$output" "Updated .agent47/context.md"
}

@test "afs map rejects preview and unknown flags" {
  run "$ROOT_DIR/bin/afs" map --preview
  [ "$status" -ne 0 ]
  assert_contains "$output" "Usage: afs map [--force]"

  run "$ROOT_DIR/bin/afs" map --unknown
  [ "$status" -ne 0 ]
  assert_contains "$output" "Usage: afs map [--force]"
}
