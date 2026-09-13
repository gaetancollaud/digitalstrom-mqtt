#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly REPO_ROOT

test_directory="$(mktemp -d)"
trap 'rm -rf "${test_directory}"' EXIT

readonly remote_path="${test_directory}/origin.git"
readonly checkout_path="${test_directory}/checkout"
readonly other_checkout_path="${test_directory}/other"
readonly fake_bin_path="${test_directory}/bin"

git init --quiet --bare "${remote_path}"
git --git-dir="${remote_path}" symbolic-ref HEAD refs/heads/master
git init --quiet --initial-branch=master "${checkout_path}"
mkdir -p \
    "${checkout_path}/scripts" \
    "${checkout_path}/home-assistant-app" \
    "${fake_bin_path}"

cp "${REPO_ROOT}/scripts/prepare-release.sh" "${checkout_path}/scripts/prepare-release.sh"
cp "${REPO_ROOT}/scripts/release.sh" "${checkout_path}/scripts/release.sh"

for script_path in \
    scripts/test-prepare-release.sh \
    scripts/test-prepare-haos-local-app.sh \
    home-assistant-app/run.sh \
    home-assistant-app/test-run.sh; do
    printf '#!/usr/bin/env bash\nset -euo pipefail\n' > "${checkout_path}/${script_path}"
done

printf '#!/usr/bin/env bash\nset -euo pipefail\n' > "${fake_bin_path}/go"
chmod +x "${fake_bin_path}/go"

printf 'name: test\nversion: "2.4.0-haos.1"\nslug: test\n' \
    > "${checkout_path}/home-assistant-app/config.yaml"
printf '# Changelog\n\n## 2.4.0-haos.1\n\n- Initial test release.\n' \
    > "${checkout_path}/home-assistant-app/CHANGELOG.md"

git -C "${checkout_path}" config user.name "Release Test"
git -C "${checkout_path}" config user.email "release-test@example.invalid"
git -C "${checkout_path}" add .
git -C "${checkout_path}" commit --quiet -m "Initial fixture"
git -C "${checkout_path}" remote add origin "${remote_path}"
git -C "${checkout_path}" push --quiet --set-upstream origin master

(
    cd "${checkout_path}"
    PATH="${fake_bin_path}:${PATH}" bash scripts/release.sh 2.4.1
)

grep --fixed-strings --line-regexp --quiet \
    'version: "2.4.1-haos.1"' "${checkout_path}/home-assistant-app/config.yaml"
grep --fixed-strings --line-regexp --quiet \
    '## 2.4.1-haos.1' "${checkout_path}/home-assistant-app/CHANGELOG.md"
grep --fixed-strings --line-regexp --quiet -- \
    '- Update digitalSTROM MQTT to 2.4.1.' "${checkout_path}/home-assistant-app/CHANGELOG.md"
test "$(git -C "${checkout_path}" log -1 --format=%s)" = "chore: prepare release 2.4.1"
test "$(git -C "${checkout_path}" rev-parse HEAD)" = \
    "$(git --git-dir="${remote_path}" rev-parse refs/heads/master)"
test "$(git -C "${checkout_path}" rev-parse refs/tags/2.4.1)" = \
    "$(git --git-dir="${remote_path}" rev-parse refs/tags/2.4.1)"
test -z "$(git -C "${checkout_path}" status --porcelain=v1 --untracked-files=all)"

(
    cd "${checkout_path}"
    bash scripts/prepare-release.sh 2.4.2
    git add home-assistant-app/config.yaml home-assistant-app/CHANGELOG.md
    git commit --quiet -m "Pre-prepare release 2.4.2"
    git push --quiet origin master
)
preprepared_head="$(git -C "${checkout_path}" rev-parse HEAD)"
(
    cd "${checkout_path}"
    PATH="${fake_bin_path}:${PATH}" bash scripts/release.sh 2.4.2
)
test "$(git -C "${checkout_path}" rev-parse HEAD)" = "${preprepared_head}"
test "$(git --git-dir="${remote_path}" rev-parse refs/tags/2.4.2)" = \
    "${preprepared_head}"

touch "${checkout_path}/dirty-file"
if (
    cd "${checkout_path}"
    PATH="${fake_bin_path}:${PATH}" bash scripts/release.sh 2.4.3
) >/dev/null 2>&1; then
    echo "Release accepted a dirty working tree." >&2
    exit 1
fi
rm "${checkout_path}/dirty-file"

git -C "${checkout_path}" switch --quiet -c feature
if (
    cd "${checkout_path}"
    PATH="${fake_bin_path}:${PATH}" bash scripts/release.sh 2.4.3
) >/dev/null 2>&1; then
    echo "Release accepted a non-master branch." >&2
    exit 1
fi
git -C "${checkout_path}" switch --quiet master

if (
    cd "${checkout_path}"
    PATH="${fake_bin_path}:${PATH}" bash scripts/release.sh 2.4.2
) >/dev/null 2>&1; then
    echo "Release accepted an existing tag." >&2
    exit 1
fi

git clone --quiet "${remote_path}" "${other_checkout_path}"
git -C "${other_checkout_path}" config user.name "Release Test"
git -C "${other_checkout_path}" config user.email "release-test@example.invalid"
printf 'remote advance\n' > "${other_checkout_path}/remote-advance"
git -C "${other_checkout_path}" add remote-advance
git -C "${other_checkout_path}" commit --quiet -m "Advance remote"
git -C "${other_checkout_path}" push --quiet origin master

if (
    cd "${checkout_path}"
    PATH="${fake_bin_path}:${PATH}" bash scripts/release.sh 2.4.3
) >/dev/null 2>&1; then
    echo "Release accepted a local master behind origin/master." >&2
    exit 1
fi

echo "Release workflow checks passed."
