#!/usr/bin/env bash
set -euo pipefail

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly CHECK_SCRIPT="${SCRIPT_DIR}/check-haos-app-release-version.sh"

test_directory="$(mktemp -d)"
trap 'rm -rf "${test_directory}"' EXIT

config_path="${test_directory}/config.yaml"
printf 'name: test\nversion: "2.4.0"\n' > "${config_path}"

bash "${CHECK_SCRIPT}" "2.4.0" "${config_path}"

if bash "${CHECK_SCRIPT}" "2.4.1" "${config_path}" >/dev/null 2>&1; then
    echo "A mismatched release tag was accepted." >&2
    exit 1
fi

printf 'name: test\n' > "${config_path}"
if bash "${CHECK_SCRIPT}" "2.4.0" "${config_path}" >/dev/null 2>&1; then
    echo "A missing App version was accepted." >&2
    exit 1
fi

echo "Home Assistant App release version checks passed."
