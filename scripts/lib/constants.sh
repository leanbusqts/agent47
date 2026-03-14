#!/bin/bash

INSTALLABLE_SCRIPTS=(
  add-agent
  add-agent-prompt
  add-snapshot-prompt
)

LEGACY_SCRIPTS=(
  add-agent-prompt-base
  add-agent-prompt-skills
  add-agent-prompt-sdd
  add-agent-prompt-ss
  copy-snapshot-prompt
)

CORE_RULE_TEMPLATE_FILES=(
  rules-mobile.yaml
  rules-frontend.yaml
  rules-backend.yaml
)

SECURITY_TEMPLATE_FILES=(
  security-global.yaml
  security-js-ts.yaml
  security-py.yaml
  security-java-kotlin.yaml
  security-swift.yaml
  security-csharp.yaml
)

ALL_RULE_TEMPLATE_FILES=(
  "${CORE_RULE_TEMPLATE_FILES[@]}"
  "${SECURITY_TEMPLATE_FILES[@]}"
)

REQUIRED_AGENTS_SECTIONS=(
  "## Purpose"
  "## Authority Order"
  "## Executable Commands"
  "## Filesystem And Approval Boundaries"
  "### Always"
  "### Ask"
  "### Never"
  "## Security Expectations"
)
