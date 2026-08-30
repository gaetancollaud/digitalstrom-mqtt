#!/usr/bin/env bash
set -euo pipefail

readonly RELEASE_VERSION="${1:?release version is required}"
readonly RELEASE_BRANCH="master"
readonly RELEASE_REMOTE="origin"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly REPO_ROOT

fail() {
    echo "$1" >&2
    exit 1
}

if [[ ! "${RELEASE_VERSION}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    fail "Release version must look like 2.4.0, without a v prefix."
fi

cd "${REPO_ROOT}"

CURRENT_BRANCH="$(git symbolic-ref --quiet --short HEAD)" ||
    fail "Release must run from the ${RELEASE_BRANCH} branch, not detached HEAD."
if [[ "${CURRENT_BRANCH}" != "${RELEASE_BRANCH}" ]]; then
    fail "Release must run from ${RELEASE_BRANCH}; current branch is ${CURRENT_BRANCH}."
fi

if [[ -n "$(git status --porcelain=v1 --untracked-files=all)" ]]; then
    fail "Release requires a clean working tree."
fi

git remote get-url "${RELEASE_REMOTE}" >/dev/null 2>&1 ||
    fail "Release remote ${RELEASE_REMOTE} is not configured."
git fetch --quiet "${RELEASE_REMOTE}" "${RELEASE_BRANCH}" --tags

if [[ "$(git rev-parse HEAD)" != "$(git rev-parse "refs/remotes/${RELEASE_REMOTE}/${RELEASE_BRANCH}")" ]]; then
    fail "Local ${RELEASE_BRANCH} must exactly match ${RELEASE_REMOTE}/${RELEASE_BRANCH}."
fi

if git show-ref --verify --quiet "refs/tags/${RELEASE_VERSION}"; then
    fail "Release tag ${RELEASE_VERSION} already exists."
fi

bash scripts/prepare-release.sh "${RELEASE_VERSION}"

while IFS= read -r -d '' status_entry; do
    changed_path="${status_entry:3}"
    case "${changed_path}" in
        home-assistant-app/config.yaml|home-assistant-app/CHANGELOG.md)
            ;;
        *)
            fail "Release preparation changed an unexpected path: ${changed_path}"
            ;;
    esac
done < <(git status --porcelain=v1 -z --untracked-files=all)

git diff --check
bash -n \
    scripts/prepare-release.sh \
    scripts/release.sh \
    scripts/test-prepare-release.sh \
    scripts/test-prepare-haos-local-app.sh \
    home-assistant-app/run.sh \
    home-assistant-app/test-run.sh
bash scripts/test-prepare-release.sh
bash scripts/test-prepare-haos-local-app.sh
bash home-assistant-app/test-run.sh
go test -count=1 -timeout 2m ./...
go vet ./...

git add -- home-assistant-app/config.yaml home-assistant-app/CHANGELOG.md
if ! git diff --cached --quiet; then
    git commit -m "chore: prepare release ${RELEASE_VERSION}"
fi

git tag "${RELEASE_VERSION}"

if ! git push --atomic "${RELEASE_REMOTE}" \
    "HEAD:refs/heads/${RELEASE_BRANCH}" \
    "refs/tags/${RELEASE_VERSION}"; then
    cat >&2 <<EOF
The atomic push failed. No partial remote release was created.
The local release commit and tag ${RELEASE_VERSION} were kept for inspection.
EOF
    exit 1
fi

printf 'Released %s; Home Assistant App version is %s-haos.1.\n' \
    "${RELEASE_VERSION}" "${RELEASE_VERSION}"
