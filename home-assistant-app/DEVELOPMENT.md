# Home Assistant App development

This document records packaging and release decisions for maintainers. User
setup and troubleshooting belong in `DOCS.md`.

## Packaging model

The App references a pre-built multi-architecture image. Home Assistant's local
App builder only receives the App directory, while the image must be built from
the repository root to include the Go bridge.

Pull requests build `amd64` and `aarch64` images without registry access. When
an App version change reaches the official `master` branch, the release job
publishes versioned images and a shared multi-architecture manifest to GHCR.
Users pull that image instead of compiling the bridge on their Home Assistant
host.

Changes to Go source or module files, App configuration or translations, the
AppArmor profile, Dockerfile, or runtime script require a new App version in
`config.yaml`. CI rejects runtime changes when an existing App version was not
incremented.

After the first publication, the `digitalstrom-mqtt-haos` package must be made
public in the GitHub package settings. The release workflow verifies that the
versioned manifest can be fetched without GitHub credentials.

## Presentation decisions

- **No Ingress**: `digitalstrom-mqtt` is a background bridge and has no web
  interface. Adding a web server only to expose Ingress would add code and
  attack surface without a user workflow.
- **No canary branch initially**: the App has one release line tied to normal
  project releases. A separate beta repository and image line should be added
  only when parallel stable and pre-release maintenance is actually needed.
- **Store artwork is active**: `icon.png` is a 128 x 128 square icon and
  `logo.png` is a 250 x 100 wide logo. CI verifies the PNG format and exact
  dimensions. The earlier proposals remain below `assets/provisional` as
  design history.

## Validated AppArmor baseline

The enforce-mode profile was validated on 2026-08-09 in an isolated HAOS 18.2
`qemux86-64` VM cloned from a working installation. The production VM remained
stopped throughout the test.

The validation covered a fresh App install, an App restart, and a complete
HAOS reboot. After every path, the App ran with its custom profile and protected
mode enabled, connected to both the Home Assistant MQTT service and the dSS
notification WebSocket, and produced no AppArmor denials. The same final profile
was then installed and started under the retained local test App slug before the
test VM was shut down.

Physical `aarch64` hardware remains untested. CI builds that architecture, but
an actual ARM start and connection test is still required before claiming live
hardware coverage.

## Validation before publication

1. Run App schema, translation, artwork, AppArmor, workflow, shell, bootstrap,
   Go test, Go vet, and Race Detector checks.
2. Build both supported image architectures.
3. Install the App in a fresh Home Assistant OS environment with a real dSS.
4. Confirm API-key creation, password cleanup, MQTT Discovery, commands, dSS
   state updates, App restart, and full HAOS restart.
5. Exercise failed and successful API-key regeneration.
6. Stop and restore Mosquitto and confirm useful recovery logs without exposed
   credentials.
7. On physical ARM hardware, confirm that the published `aarch64` image starts
   and reaches MQTT and the dSS. The architecture is built in CI, but this
   hardware path remains unverified until such a device is available.
8. Verify that the exact versioned GHCR manifest is anonymously pullable before
   announcing a release.
