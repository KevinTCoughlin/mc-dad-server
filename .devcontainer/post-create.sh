#!/usr/bin/env bash
set -euo pipefail

go mod download
just tools

if command -v kotlinc >/dev/null 2>&1; then
  exit 0
fi

if [ -n "${SDKMAN_DIR:-}" ] && [ -s "${SDKMAN_DIR}/bin/sdkman-init.sh" ]; then
  # shellcheck source=/dev/null
  source "${SDKMAN_DIR}/bin/sdkman-init.sh"
elif [ -s "/usr/local/sdkman/bin/sdkman-init.sh" ]; then
  # shellcheck source=/dev/null
  source "/usr/local/sdkman/bin/sdkman-init.sh"
fi

if command -v sdk >/dev/null 2>&1; then
  sdk install kotlin
elif command -v apt-get >/dev/null 2>&1; then
  sudo apt-get update
  sudo apt-get install -y kotlin
else
  echo "Unable to install Kotlin: no sdkman or apt-get available." >&2
  exit 1
fi
