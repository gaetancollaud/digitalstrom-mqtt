#!/usr/bin/env bash
set -euo pipefail

readonly VERSION="${1:?release version is required}"
readonly CONFIG_PATH="${2:-home-assistant-app/config.yaml}"
readonly CHANGELOG_PATH="${3:-home-assistant-app/CHANGELOG.md}"

if [[ ! "${VERSION}" =~ ^[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]]; then
    echo "Release version must look like 2.4.0, without a v prefix." >&2
    exit 2
fi

if [[ ! -f "${CONFIG_PATH}" ]]; then
    echo "Home Assistant App config not found: ${CONFIG_PATH}" >&2
    exit 1
fi

temporary_config="$(mktemp)"
trap 'rm -f "${temporary_config}"' EXIT

awk -v version="${VERSION}" '
    BEGIN { replacements = 0 }
    /^version:[[:space:]]*/ {
        print "version: \"" version "\""
        replacements++
        next
    }
    { print }
    END {
        if (replacements != 1) {
            exit 1
        }
    }
' "${CONFIG_PATH}" > "${temporary_config}" || {
    echo "Expected exactly one version field in ${CONFIG_PATH}." >&2
    exit 1
}

mv "${temporary_config}" "${CONFIG_PATH}"
trap - EXIT

printf 'Set Home Assistant App version to %s in %s.\n' "${VERSION}" "${CONFIG_PATH}"
if ! grep --fixed-strings --line-regexp --quiet "## ${VERSION}" "${CHANGELOG_PATH}"; then
    printf 'Next: add a ## %s entry to %s before creating the tag.\n' \
        "${VERSION}" "${CHANGELOG_PATH}"
fi
