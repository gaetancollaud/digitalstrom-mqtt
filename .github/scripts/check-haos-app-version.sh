#!/usr/bin/env bash
set -euo pipefail

readonly BASE_REF="${1:?base Git reference is required}"
readonly HEAD_REF="${2:-HEAD}"

if [[ "${BASE_REF}" =~ ^0+$ ]] || ! git cat-file -e "${BASE_REF}^{commit}" 2>/dev/null; then
    echo "No previous commit is available; skipping the Home Assistant App version check."
    exit 0
fi

runtime_changes="$(
    git diff --name-only "${BASE_REF}" "${HEAD_REF}" -- \
        '*.go' \
        go.mod \
        go.sum \
        home-assistant-app/config.yaml \
        home-assistant-app/apparmor.txt \
        home-assistant-app/Dockerfile \
        home-assistant-app/run.sh \
        'home-assistant-app/translations/*.yaml'
)"
if [[ -z "${runtime_changes}" ]]; then
    echo "No Home Assistant App runtime inputs changed."
    exit 0
fi

read_app_version() {
    sed -n 's/^version:[[:space:]]*"\{0,1\}\([^"[:space:]]*\)"\{0,1\}[[:space:]]*$/\1/p'
}

current_version="$(git show "${HEAD_REF}:home-assistant-app/config.yaml" | read_app_version)"
previous_version="$(git show "${BASE_REF}:home-assistant-app/config.yaml" 2>/dev/null | read_app_version || true)"

if [[ -z "${current_version}" ]]; then
    echo "The Home Assistant App version is missing from config.yaml." >&2
    exit 1
fi

if [[ -n "${previous_version}" && "${current_version}" == "${previous_version}" ]]; then
    printf 'Home Assistant App runtime inputs changed without a version bump (%s):\n%s\n' \
        "${current_version}" "${runtime_changes}" >&2
    exit 1
fi

printf 'Home Assistant App runtime version changed from %s to %s.\n' \
    "${previous_version:-not present}" "${current_version}"
