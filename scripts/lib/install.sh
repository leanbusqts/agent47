#!/bin/bash
set -euo pipefail

install_scripts() {
  local force=false
  if [ "${1:-}" = "--force" ]; then
    force=true
  fi

  install_managed_runtime "$force"
}

uninstall() {
  uninstall_managed_runtime
}
