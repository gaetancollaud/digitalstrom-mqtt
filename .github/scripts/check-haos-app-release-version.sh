#!/usr/bin/env bash
set -euo pipefail

readonly RELEASE_TAG="${1:?release tag is required}"
readonly CONFIG_PATH="${2:-home-assistant-app/config.yaml}"

read_app_version() {
    sed -n 's/^version:[[:space:]]*"\{0,1\}\([^"[:space:]]*\)"\{0,1\}[[:space:]]*$/\1/p' "${CONFIG_PATH}"
}

app_version="$(read_app_version)"

if [[ -z "${app_version}" ]]; then
    echo "The Home Assistant App version is missing from ${CONFIG_PATH}." >&2
    exit 1
fi

if [[ "${RELEASE_TAG}" != "${app_version}" ]]; then
    printf 'Release tag %s does not match the Home Assistant App version %s.\n' \
        "${RELEASE_TAG}" "${app_version}" >&2
    exit 1
fi

printf 'Release tag %s matches the Home Assistant App version.\n' "${RELEASE_TAG}"
