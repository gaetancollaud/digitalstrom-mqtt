#!/usr/bin/env bash
set -euo pipefail

readonly RELEASE_VERSION="${1:?release version is required}"
readonly APP_VERSION="${RELEASE_VERSION}-haos.1"
readonly CONFIG_PATH="${2:-home-assistant-app/config.yaml}"
readonly CHANGELOG_PATH="${3:-home-assistant-app/CHANGELOG.md}"

if [[ ! "${RELEASE_VERSION}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Release version must look like 2.4.0, without a v prefix." >&2
    exit 2
fi

if [[ ! -f "${CONFIG_PATH}" ]]; then
    echo "Home Assistant App config not found: ${CONFIG_PATH}" >&2
    exit 1
fi

temporary_config="$(mktemp)"
trap 'rm -f "${temporary_config}"' EXIT

awk -v version="${APP_VERSION}" '
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

printf 'Set Home Assistant App version to %s for project release %s in %s.\n' \
    "${APP_VERSION}" "${RELEASE_VERSION}" "${CONFIG_PATH}"
if ! grep --fixed-strings --line-regexp --quiet "## ${APP_VERSION}" "${CHANGELOG_PATH}"; then
    printf 'Next: add a ## %s entry to %s before creating the tag.\n' \
        "${APP_VERSION}" "${CHANGELOG_PATH}"
fi
