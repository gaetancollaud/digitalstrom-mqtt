#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
readonly CHECK_SCRIPT="${SCRIPT_DIR}/check-haos-app-release-version.sh"

test_directory="$(mktemp -d)"
trap 'rm -rf "${test_directory}"' EXIT

config_path="${test_directory}/config.yaml"
changelog_path="${test_directory}/CHANGELOG.md"
printf 'name: test\nversion: "2.4.0"\n' > "${config_path}"
printf '# Changelog\n\n## 2.4.0\n\n- Test release.\n' > "${changelog_path}"

bash "${CHECK_SCRIPT}" "2.4.0" "${config_path}" "${changelog_path}"

if bash "${CHECK_SCRIPT}" "2.4.1" "${config_path}" "${changelog_path}" >/dev/null 2>&1; then
    echo "A mismatched release tag was accepted." >&2
    exit 1
fi

printf 'name: test\n' > "${config_path}"
if bash "${CHECK_SCRIPT}" "2.4.0" "${config_path}" "${changelog_path}" >/dev/null 2>&1; then
    echo "A missing App version was accepted." >&2
    exit 1
fi

printf 'name: test\nversion: "2.4.0"\n' > "${config_path}"
printf '# Changelog\n' > "${changelog_path}"
if bash "${CHECK_SCRIPT}" "2.4.0" "${config_path}" "${changelog_path}" >/dev/null 2>&1; then
    echo "A missing App changelog entry was accepted." >&2
    exit 1
fi

echo "Home Assistant App release version checks passed."
