# Home Assistant App development

This document records packaging and release decisions for maintainers. User
setup and troubleshooting belong in `DOCS.md`.

## Packaging model

The App references a pre-built multi-architecture image. Home Assistant's local
App builder only receives the App directory, while the image must be built from
the repository root to include the Go bridge.

Pull requests build `amd64` and `aarch64` images without registry access. The
workflow uses the versioned Home Assistant builder composite actions and native
GitHub runners for both architectures. When a Git tag is pushed to the official
repository, the existing GoReleaser workflow publishes the standalone release
while the App workflow builds, signs, and publishes versioned images and a
shared multi-architecture manifest to GHCR. Users pull that image instead of
compiling the bridge on their Home Assistant host.

The Git tag, standalone release, Docker image, and App version use the same
version number. The App workflow rejects a release tag that does not exactly
match `version` in `config.yaml`. Normal pull requests do not require an App
version change and never publish images, so fork pull requests can run all
checks without access to repository publishing secrets.

## Test a pull request on Home Assistant OS

The normal App entry points at the official GHCR image. Pull requests validate
that image without publishing it, so adding a pull-request branch as an App
repository is not enough to run the branch on Home Assistant OS.

After checking out the branch or commit to test, use the local App generator
to prepare an exact Git commit without publishing an image:

```sh
sh scripts/prepare-haos-local-app.sh \
  /tmp/digitalstrom_mqtt_pr_test HEAD
```

Fetch or check out the pull request with the Git remote or GitHub tool used for
your checkout. Alternatively, pass any local commit, tag or branch as the
second argument instead of `HEAD`. The generator uses `git archive`, so it
exports the selected commit and deliberately ignores uncommitted working-tree
changes. It creates a standalone local App directory with these
development-only changes:

- name `digitalSTROM MQTT PR Test`;
- slug `digitalstrom_mqtt_pr_test`;
- manual boot and experimental stage;
- no `image` entry, which makes Supervisor build the App locally;
- an AppArmor profile matching the local test slug.

The release configuration and source checkout are not modified. The generated
`LOCAL_BUILD_SOURCE.txt` records the exact commit used for the package.

Copy the generated `digitalstrom_mqtt_pr_test` directory into the Home
Assistant local App directory. Home Assistant's
[local testing guide](https://developers.home-assistant.io/docs/apps/testing/)
documents two supported ways:

- copy it to the `addons` Samba share; or
- copy it to `/addons` through the SSH App.

In Home Assistant, open **Settings -> Apps -> App store**, select **Check for
updates** from the three-dot menu, and install **digitalSTROM MQTT PR Test**
from **Local apps**. The installed App slug is
`local_digitalstrom_mqtt_pr_test`.

Never run this test App and another `digitalstrom-mqtt` instance at the same
time when they use the same MQTT discovery prefix. For an end-to-end check,
stop the existing bridge before starting the test App, then verify first-start
API-key creation, MQTT and dSS connectivity, a reversible command with state
feedback, App restart, HAOS reboot, Mosquitto failure and recovery, and failed
and successful API-key regeneration.

To test a newer commit with the same App version, uninstall the local test App,
replace its directory in `/addons`, check for updates again, and reinstall it.
After testing, uninstall the local App, remove its directory from `/addons`,
and revoke the dedicated test API key in the dSS.

This path validates the selected source, local container build, Supervisor
options, runtime and AppArmor profile on the HAOS machine's architecture. It
does not validate anonymous GHCR access, the release manifest, or runtime on a
different architecture. Those remain release and physical-hardware gates.

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
- **Stable community package**: the package is marked stable because the full
  install, configuration, restart, and reboot path was validated on a fresh
  HAOS VM with a real dSS. User-facing text still states clearly that this is an
  unofficial community App. Physical ARM runtime coverage remains listed as an
  open validation item instead of changing the lifecycle status of the tested
  package.
- **Supervisor watchdog**: the health endpoint remains private to the App
  network. The Supervisor checks `/health/live`, which verifies the bridge
  process without treating a recoverable MQTT outage as a reason to restart.
  `/health/ready` continues to include the MQTT connection state for diagnosis.
- **Store artwork is active**: `icon.png` is a 128 x 128 square icon and
  `logo.png` is a 250 x 100 wide logo. CI verifies the PNG format and exact
  dimensions. Both images are project-specific artwork and do not reuse an
  official digitalSTROM or Home Assistant logo.

## Validated AppArmor baseline

The enforce-mode profile was validated on 2026-08-09 in an isolated HAOS 18.2
`qemux86-64` VM cloned from a working installation. The production VM remained
stopped throughout the test.

The validation covered a fresh App install, an App restart, and a complete
HAOS reboot. After every path, the App ran with its custom profile and protected
mode enabled, connected to both the Home Assistant MQTT service and the dSS
notification WebSocket, and produced no AppArmor denials. That baseline profile
was then installed and started under the retained local test App slug before the
test VM was shut down.

The current profile adds write access to the App's private `/data` volume for the
bridge child process. This is required by the first-start API-key transaction;
all other App setup and Supervisor option handling remains in the launcher
profile. CI parses this exact profile. A repeat HAOS install should confirm the
updated profile before the first public release.

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

## Release sequence

1. Set the release version in `home-assistant-app/config.yaml` and add the same
   version to `CHANGELOG.md`.
2. Merge the validated pull request into `master`.
3. Create and push a Git tag with exactly the same version, for example
   `2.4.0`.
4. Confirm that the GoReleaser and Home Assistant App workflows both complete.
5. Confirm that the GitHub release, DockerHub images, and GHCR App manifest all
   use the same version.

Treat the merge and matching tag as one release operation. Until the tag jobs
have published the versioned GHCR manifest, the App version on `master` is not
ready to install or announce.
