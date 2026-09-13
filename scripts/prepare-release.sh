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

if [[ ! -f "${CHANGELOG_PATH}" ]]; then
    echo "Home Assistant App changelog not found: ${CHANGELOG_PATH}" >&2
    exit 1
fi

if [[ "$(sed -n '1p' "${CHANGELOG_PATH}")" != "# Changelog" ]]; then
    echo "Expected ${CHANGELOG_PATH} to start with # Changelog." >&2
    exit 1
fi

temporary_config="$(mktemp)"
temporary_changelog="$(mktemp)"
trap 'rm -f "${temporary_config}" "${temporary_changelog}"' EXIT

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

if grep --fixed-strings --line-regexp --quiet "## ${APP_VERSION}" "${CHANGELOG_PATH}"; then
    cp "${CHANGELOG_PATH}" "${temporary_changelog}"
else
    awk -v app_version="${APP_VERSION}" -v release_version="${RELEASE_VERSION}" '
        NR == 1 {
            print
            print ""
            print "## " app_version
            print ""
            print "- Update digitalSTROM MQTT to " release_version "."
            next
        }
        { print }
    ' "${CHANGELOG_PATH}" > "${temporary_changelog}"
fi

mv "${temporary_config}" "${CONFIG_PATH}"
mv "${temporary_changelog}" "${CHANGELOG_PATH}"
trap - EXIT

printf 'Prepared Home Assistant App version %s for project release %s.\n' \
    "${APP_VERSION}" "${RELEASE_VERSION}"
