#!/usr/bin/env bats

load ../helpers/common

setup() {
  setup_workdir
}

teardown() {
  teardown_workdir
}

@test "afs analyze reports base bundle in an empty repo" {
  run "$ROOT_DIR/bin/afs" analyze
  assert_success
  assert_contains "$output" "type: unknown"
  assert_contains "$output" "bundles: base"
}

@test "afs analyze --json emits install_plan" {
  run "$ROOT_DIR/bin/afs" analyze --json
  assert_success
  assert_contains "$output" "\"install_plan\""
  assert_contains "$output" "\"base_bundle\": true"
}

@test "afs analyze detects cli and scripts signals" {
  mkdir -p cmd scripts
  echo "module example.com/test" > go.mod
  echo '#!/usr/bin/env bash' > install.sh

  run "$ROOT_DIR/bin/afs" analyze
  assert_success
  assert_contains "$output" "type: cli, scripts"
  assert_contains "$output" "bundles: base, project-cli, project-scripts, shared-cli-behavior, shared-testing"
}

@test "afs analyze detects cli and monorepo-tooling signals" {
  mkdir -p cmd apps
  echo '{"devDependencies":{"turbo":"1.0.0"}}' > package.json
  echo 'packages:' > pnpm-workspace.yaml

  run "$ROOT_DIR/bin/afs" analyze
  assert_success
  assert_contains "$output" "type: cli, monorepo-tooling"
  assert_contains "$output" "bundles: base, project-cli, project-monorepo-tooling, shared-cli-behavior, shared-testing"
}

@test "afs analyze detects plugin and desktop signals" {
  mkdir -p plugins
  echo '{"dependencies":{"electron":"1.0.0"}}' > package.json
  echo '{"name":"sample"}' > plugin.json

  run "$ROOT_DIR/bin/afs" analyze
  assert_success
  assert_contains "$output" "type: desktop, plugin"
  assert_contains "$output" "bundles: base, project-desktop, project-plugin, shared-testing"
}

@test "afs analyze --verbose reports unresolved conflict details" {
  mkdir -p src api
  echo '{"dependencies":{"react":"1.0.0","express":"1.0.0"}}' > package.json

  run "$ROOT_DIR/bin/afs" analyze --verbose
  assert_success
  assert_contains "$output" "Conflict"
  assert_contains "$output" "unsupported automatic composition: backend, frontend"
  assert_contains "$output" "fallback: base bundle only"
}

@test "afs analyze --verbose reports testing stacks and mapped skills" {
  mkdir -p tests
  cat > package.json <<'JSON'
{"devDependencies":{"vitest":"1.0.0","playwright":"1.0.0"}}
JSON
  echo 'module example.com/test' > go.mod
  echo 'package main' > service_test.go
  echo 'export default {}' > vitest.config.ts
  echo 'export default {}' > playwright.config.ts
  echo '#!/usr/bin/env bats' > tests/smoke.bats

  run "$ROOT_DIR/bin/afs" analyze --verbose
  assert_success
  assert_contains "$output" "Testing stacks"
  assert_contains "$output" "vitest"
  assert_contains "$output" "playwright"
  assert_contains "$output" "go-test"
  assert_contains "$output" "bats"
  assert_contains "$output" "refactor"
  assert_contains "$output" "optimize"
}
