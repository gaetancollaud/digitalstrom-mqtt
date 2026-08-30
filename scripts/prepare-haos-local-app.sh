#!/bin/sh
set -eu

usage() {
    cat >&2 <<'EOF'
usage: scripts/prepare-haos-local-app.sh OUTPUT_DIR [GIT_REF]

Create a local Home Assistant App source tree from GIT_REF (default: HEAD).
OUTPUT_DIR must not exist and its final path component must be
digitalstrom_mqtt_pr_test.
EOF
    exit 2
}

if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
    usage
fi

output_dir=$1
git_ref=${2:-HEAD}
local_slug=digitalstrom_mqtt_pr_test

case "$git_ref" in
    -*)
        echo "Git ref must not start with a dash: $git_ref" >&2
        exit 2
        ;;
esac

script_dir=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/.." && pwd)

if [ "$(basename -- "$output_dir")" != "$local_slug" ]; then
    echo "output directory must end in $local_slug" >&2
    exit 2
fi

output_parent=$(dirname -- "$output_dir")
if [ ! -d "$output_parent" ]; then
    echo "output parent does not exist: $output_parent" >&2
    exit 2
fi
output_parent=$(CDPATH= cd -- "$output_parent" && pwd)
output_dir="$output_parent/$local_slug"

if [ -e "$output_dir" ]; then
    echo "refusing to replace existing output: $output_dir" >&2
    exit 1
fi

commit=$(git -C "$repo_root" rev-parse --verify "$git_ref^{commit}") || {
    echo "could not resolve Git ref: $git_ref" >&2
    exit 1
}

staging=$(mktemp -d "$output_parent/.digitalstrom-mqtt-haos-local.XXXXXX")
cleanup() {
    status=$?
    trap - EXIT HUP INT TERM
    case "$staging" in
        "$output_parent"/.digitalstrom-mqtt-haos-local.*)
            rm -rf -- "$staging"
            ;;
        *)
            echo "refusing to clean unexpected staging path: $staging" >&2
            status=1
            ;;
    esac
    exit "$status"
}
trap cleanup EXIT
trap 'exit 1' HUP INT TERM

git -C "$repo_root" archive --format=tar "$commit" | tar -xf - -C "$staging"

app_source="$staging/home-assistant-app"
for required in \
    config.yaml Dockerfile run.sh apparmor.txt README.md DOCS.md DOCS.de.md \
    CHANGELOG.md icon.png logo.png translations; do
    if [ ! -e "$app_source/$required" ]; then
        echo "missing App source at $git_ref: home-assistant-app/$required" >&2
        exit 1
    fi
done

cp "$app_source/Dockerfile" "$staging/Dockerfile"
cp "$app_source/run.sh" "$staging/run.sh"
cp "$app_source/apparmor.txt" "$staging/apparmor.txt"
cp "$app_source/README.md" "$staging/README.md"
cp "$app_source/DOCS.md" "$staging/DOCS.md"
cp "$app_source/DOCS.de.md" "$staging/DOCS.de.md"
cp "$app_source/CHANGELOG.md" "$staging/CHANGELOG.md"
cp "$app_source/icon.png" "$staging/icon.png"
cp "$app_source/logo.png" "$staging/logo.png"
cp -R "$app_source/translations" "$staging/translations"

awk '
    /^name:/  { print "name: digitalSTROM MQTT PR Test"; next }
    /^slug:/  { print "slug: digitalstrom_mqtt_pr_test"; next }
    /^boot:/  { print "boot: manual"; next }
    /^stage:/ { print "stage: experimental"; next }
    /^image:/ { next }
    { print }
' "$app_source/config.yaml" > "$staging/config.yaml"

if ! grep -q '^COPY home-assistant-app/run\.sh /run\.sh$' "$staging/Dockerfile"; then
    echo "App Dockerfile no longer contains the expected run.sh copy instruction" >&2
    exit 1
fi
sed 's#^COPY home-assistant-app/run\.sh /run\.sh$#COPY run.sh /run.sh#' \
    "$staging/Dockerfile" > "$staging/Dockerfile.local"
mv "$staging/Dockerfile.local" "$staging/Dockerfile"

if ! grep -q '^profile digitalstrom_mqtt ' "$staging/apparmor.txt"; then
    echo "AppArmor profile no longer contains the release App slug" >&2
    exit 1
fi
sed 's/^profile digitalstrom_mqtt /profile digitalstrom_mqtt_pr_test /' \
    "$staging/apparmor.txt" > "$staging/apparmor.local"
mv "$staging/apparmor.local" "$staging/apparmor.txt"

# The release App directory has been flattened into the local App root. Keeping
# the nested config.yaml would make Supervisor discover a second App.
rm -rf -- "$app_source"

{
    printf 'Git commit: %s\n' "$commit"
    printf 'Requested ref: %s\n' "$git_ref"
    printf 'Generated slug: %s\n' "$local_slug"
} > "$staging/LOCAL_BUILD_SOURCE.txt"

config_count=$(find "$staging" -type f -name config.yaml -print | wc -l | tr -d ' ')
if [ "$config_count" -ne 1 ]; then
    echo "local App must contain exactly one config.yaml, found $config_count" >&2
    exit 1
fi

mv "$staging" "$output_dir"
trap - EXIT HUP INT TERM

printf 'Prepared local Home Assistant App at %s\n' "$output_dir"
printf 'Source commit: %s\n' "$commit"
