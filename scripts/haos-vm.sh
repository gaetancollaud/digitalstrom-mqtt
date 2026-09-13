#!/bin/sh
# Boot Home Assistant OS in a QEMU VM, which is the only place with a real app
# store. Home Assistant lands on http://localhost:8123.
#
#   scripts/haos-vm.sh [APP_DIR]   download the image if needed, then boot
#                                  (Ctrl-A X quits)
#
# APP_DIR is shared with the guest over 9p so it can be copied into the local
# addons directory, which is how home-assistant-app/DEVELOPMENT.md tests a
# local App build. The script prints the guest commands to run after boot.
#
# Home Assistant serves on port 80 in the guest; its port 8123 only redirects
# there, dropping the port and breaking the websocket. So host 8123 maps to 80.
#
# To install this app in the VM, add this repository under
# Settings > Apps > App Store > Repositories. That installs the published image.
# To test local code instead, generate an App directory with
# scripts/prepare-haos-local-app.sh and pass it to this script.
set -eu

APP_DIR=${1:-}
if [ -n "$APP_DIR" ]; then
    [ -d "$APP_DIR" ] || { echo "not a directory: $APP_DIR" >&2; exit 2; }
    APP_DIR=$(CDPATH='' cd -- "$APP_DIR" && pwd)
    APP_NAME=$(basename -- "$APP_DIR")
    # ponytail: 9p, not virtiofs; no extra daemon and HAOS ships the module.
    set -- -virtfs "local,path=$APP_DIR,mount_tag=addons,security_model=none"
else
    set --
fi

VERSION=${HAOS_VERSION:-18.2}
DIR=${HAOS_DIR:-$HOME/.cache/haos-vm}
IMAGE="$DIR/haos_ova-$VERSION.qcow2"
VARS="$DIR/OVMF_VARS-$VERSION.fd"
OVMF=${OVMF_CODE:-/usr/share/edk2/ovmf/OVMF_CODE.fd}

mkdir -p "$DIR"
if [ ! -f "$IMAGE" ]; then
    echo "Downloading Home Assistant OS $VERSION..."
    curl -fL --progress-bar -o "$IMAGE.xz" \
        "https://github.com/home-assistant/operating-system/releases/download/$VERSION/haos_ova-$VERSION.qcow2.xz"
    xz -d "$IMAGE.xz"
fi
# HAOS boots over UEFI, so the firmware needs its own writable variable store.
# Start from a pristine copy every time: a store HAOS has already written to
# makes OVMF drop the disk and fall through to PXE on the next boot. Nothing
# here depends on the boot entries surviving, since HAOS boots through the
# default \EFI\BOOT\BOOTX64.EFI path.
cp "$(dirname "$OVMF")/OVMF_VARS.fd" "$VARS"

echo "Home Assistant will be on http://localhost:8123 once it has started."
if [ -n "$APP_DIR" ]; then
    cat <<EOF

$APP_DIR is shared with the guest. Type \`login\` at the console prompt below
once HAOS has booted, then run:

  mkdir -p /tmp/host
  mount -t 9p -o trans=virtio,version=9p2000.L addons /tmp/host
  cp -a /tmp/host /mnt/data/supervisor/apps/local/$APP_NAME

Then use Settings > Apps > App store > Check for updates to see it under
Local apps. Copy rather than mounting straight onto the addons directory: the
Supervisor container bind-mounted that path when it started, so a mount made
later is not visible inside it and the App never shows up. Repeat the copy
after changing the App on the host; the 9p mount is lost on reboot.
EOF
fi
exec qemu-system-x86_64 \
    -machine q35,accel=kvm -cpu host -smp 2 -m 4G \
    -drive if=pflash,format=raw,readonly=on,file="$OVMF" \
    -drive if=pflash,format=raw,file="$VARS" \
    -drive file="$IMAGE",if=virtio,format=qcow2 \
    -netdev user,id=net0,hostfwd=tcp::8123-:80,hostfwd=tcp::22222-:22222 \
    -device virtio-net-pci,netdev=net0 \
    -display none -serial mon:stdio "$@"
