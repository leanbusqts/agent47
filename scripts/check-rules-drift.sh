#!/usr/bin/env bash
set -Eeuo pipefail

cd "$(git rev-parse --show-toplevel 2>/dev/null || pwd)"

err=0

for f in security-global security-shell security-csharp security-java-kotlin security-js-ts security-py security-swift security-go rules-backend rules-frontend rules-mobile rules-cross rules-go; do
  if ! diff -q "rules/${f}.yaml" "templates/base/rules/${f}.yaml" >/dev/null; then
    echo "DRIFT (base): ${f}.yaml"
    err=1
  fi
done
for pair in \
  "project-backend:rules-backend" \
  "project-frontend:rules-frontend" \
  "project-mobile:rules-mobile" \
  "project-cli:rules-cli" \
  "project-desktop:rules-desktop" \
  "project-infra:rules-infra" \
  "project-monorepo-tooling:rules-monorepo-tooling" \
  "project-plugin:rules-plugin" \
  "project-scripts:rules-scripts" \
  "shared-cli-behavior:shared-cli-behavior" \
  "shared-testing:shared-testing"; do
  bundle="${pair%%:*}"
  file="${pair##*:}"
  if ! diff -q "rules/${file}.yaml" "templates/bundles/${bundle}/rules/${file}.yaml" >/dev/null; then
    echo "DRIFT (bundle): rules/${file}.yaml != templates/bundles/${bundle}/rules/${file}.yaml"
    err=1
  fi
done

exit "$err"
