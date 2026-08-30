#!/bin/sh
set -eu

repo_root=$(CDPATH='' cd -- "$(dirname "$0")/.." && pwd)
generator="$repo_root/scripts/prepare-haos-local-app.sh"
test_root=$(mktemp -d)
cleanup() {
    status=$?
    trap - EXIT HUP INT TERM
    rm -rf -- "$test_root"
    exit "$status"
}
trap cleanup EXIT
trap 'exit 1' HUP INT TERM

before_status=$(git -C "$repo_root" status --porcelain=v1 --untracked-files=all)
output="$test_root/digitalstrom_mqtt_pr_test"

# CI and this test deliberately exercise committed HEAD. The generator never
# mixes uncommitted working-tree files into a package attributed to a commit.
/bin/sh "$generator" "$output" HEAD

test -f "$output/config.yaml"
test -f "$output/Dockerfile"
test -f "$output/run.sh"
test -f "$output/apparmor.txt"
test -f "$output/go.mod"
test -f "$output/go.sum"
test -f "$output/main.go"
test -d "$output/pkg"
test ! -e "$output/home-assistant-app"

test "$(find "$output" -type f -name config.yaml -print | wc -l | tr -d ' ')" -eq 1
grep -qx 'name: digitalSTROM MQTT PR Test' "$output/config.yaml"
grep -qx 'slug: digitalstrom_mqtt_pr_test' "$output/config.yaml"
grep -qx 'boot: manual' "$output/config.yaml"
grep -qx 'stage: experimental' "$output/config.yaml"
if grep -q '^image:' "$output/config.yaml"; then
    echo "local App config still contains an image reference" >&2
    exit 1
fi

grep -qx 'COPY run.sh /run.sh' "$output/Dockerfile"
if grep -q 'home-assistant-app/run.sh' "$output/Dockerfile"; then
    echo "local App Dockerfile still expects the release directory layout" >&2
    exit 1
fi
grep -q '^profile digitalstrom_mqtt_pr_test ' "$output/apparmor.txt"
if grep -q '^profile digitalstrom_mqtt ' "$output/apparmor.txt"; then
    echo "local AppArmor profile still uses the release slug" >&2
    exit 1
fi

full_validation=${HAOS_LOCAL_APP_FULL_VALIDATION:-0}
if command -v ruby >/dev/null 2>&1; then
    ruby -ryaml -e '
        root = ARGV.fetch(0)
        app = YAML.load_file(File.join(root, "config.yaml"))
        abort "unexpected local App name" unless app.fetch("name") == "digitalSTROM MQTT PR Test"
        abort "unexpected local App slug" unless app.fetch("slug") == "digitalstrom_mqtt_pr_test"
        abort "local App must build locally" if app.key?("image")
        translations = %w[en de].map { |language| YAML.load_file(File.join(root, "translations", "#{language}.yaml")) }
        abort "translation schema mismatch" unless translations.all? { |translation| translation.fetch("configuration").keys.sort == app.fetch("schema").keys.sort }
    ' "$output"
elif command -v python >/dev/null 2>&1 && python -c 'import yaml' >/dev/null 2>&1; then
    python - "$output" <<'PY'
import pathlib
import sys
import yaml

root = pathlib.Path(sys.argv[1])
with (root / "config.yaml").open(encoding="utf-8") as handle:
    app = yaml.safe_load(handle)
assert app["name"] == "digitalSTROM MQTT PR Test"
assert app["slug"] == "digitalstrom_mqtt_pr_test"
assert "image" not in app
for language in ("en", "de"):
    with (root / "translations" / f"{language}.yaml").open(encoding="utf-8") as handle:
        translation = yaml.safe_load(handle)
    assert sorted(translation["configuration"]) == sorted(app["schema"])
PY
elif [ "$full_validation" = 1 ]; then
    echo "Ruby or Python with PyYAML is required for full local App validation" >&2
    exit 1
else
    echo "Skipping generated YAML parse: Ruby and Python with PyYAML are unavailable" >&2
fi

if command -v apparmor_parser >/dev/null 2>&1; then
    apparmor_parser --skip-kernel-load --skip-cache "$output/apparmor.txt"
elif [ "$full_validation" = 1 ]; then
    echo "apparmor_parser is required for full local App validation" >&2
    exit 1
else
    echo "Skipping generated AppArmor parse: apparmor_parser is unavailable" >&2
fi

expected_commit=$(git -C "$repo_root" rev-parse HEAD)
grep -qx "Git commit: $expected_commit" "$output/LOCAL_BUILD_SOURCE.txt"

mkdir "$test_root/existing"
printf 'preserve\n' > "$test_root/existing/sentinel"
if /bin/sh "$generator" "$test_root/existing" >/dev/null 2>&1; then
    echo "generator accepted an output path with the wrong basename" >&2
    exit 1
fi
test "$(cat "$test_root/existing/sentinel")" = preserve

mkdir "$test_root/occupied"
occupied="$test_root/occupied/digitalstrom_mqtt_pr_test"
mkdir "$occupied"
printf 'preserve\n' > "$occupied/sentinel"
if /bin/sh "$generator" "$occupied" >/dev/null 2>&1; then
    echo "generator replaced an existing output directory" >&2
    exit 1
fi
test "$(cat "$occupied/sentinel")" = preserve

invalid="$test_root/invalid/digitalstrom_mqtt_pr_test"
mkdir "$test_root/invalid"
if /bin/sh "$generator" "$invalid" refs/heads/does-not-exist >/dev/null 2>&1; then
    echo "generator accepted an invalid Git ref" >&2
    exit 1
fi
test ! -e "$invalid"

dashed="$test_root/dashed/digitalstrom_mqtt_pr_test"
mkdir "$test_root/dashed"
if /bin/sh "$generator" "$dashed" --not-a-ref >/dev/null 2>&1; then
    echo "generator accepted a Git ref starting with a dash" >&2
    exit 1
fi
test ! -e "$dashed"

after_status=$(git -C "$repo_root" status --porcelain=v1 --untracked-files=all)
test "$before_status" = "$after_status"

echo "HAOS local App preparation test passed"
