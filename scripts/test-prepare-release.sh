#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly REPO_ROOT
readonly PREPARE_SCRIPT="${REPO_ROOT}/scripts/prepare-release.sh"

test_directory="$(mktemp -d)"
trap 'rm -rf "${test_directory}"' EXIT

config_path="${test_directory}/config.yaml"
changelog_path="${test_directory}/CHANGELOG.md"
printf 'name: test\nversion: "2.4.0"\nslug: test\n' > "${config_path}"
printf '# Changelog\n' > "${changelog_path}"

bash "${PREPARE_SCRIPT}" "2.4.1" "${config_path}" "${changelog_path}"
grep --fixed-strings --line-regexp --quiet 'version: "2.4.1"' "${config_path}"
grep --fixed-strings --line-regexp --quiet 'slug: test' "${config_path}"

if bash "${PREPARE_SCRIPT}" "v2.4.1" "${config_path}" "${changelog_path}" >/dev/null 2>&1; then
    echo "Release preparation accepted a v-prefixed version." >&2
    exit 1
fi

printf 'name: test\nslug: test\n' > "${config_path}"
if bash "${PREPARE_SCRIPT}" "2.4.2" "${config_path}" "${changelog_path}" >/dev/null 2>&1; then
    echo "Release preparation accepted a config without a version field." >&2
    exit 1
fi

echo "Release preparation checks passed."
