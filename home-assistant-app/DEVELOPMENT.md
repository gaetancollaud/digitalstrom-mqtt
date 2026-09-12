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
shared multi-architecture manifest to the existing
`gaetancollaud/digitalstrom-mqtt` Docker Hub repository. Users pull that image
instead of compiling the bridge on their Home Assistant host.

The App uses the project's existing Docker Hub registry and `DOCKERHUB_TOKEN`
secret. The standalone and App images contain different runtime packaging, so
their Docker tags distinguish them while both remain in the same image
repository. A project release tag `2.4.0` publishes the standalone image as
`2.4.0` and the App image as `2.4.0-haos.1`. Home Assistant uses the App
`version` as the image tag and pulls that exact variant.

The `.1` suffix is the first App packaging revision for a project release. The
initial workflow does not add a separate App-only Git tag path; packaging fixes
use the next normal project patch release until an independent App revision is
actually needed. The App workflow rejects an official project tag unless the
configured App version is `<tag>-haos.1`. This does not block the existing
standalone Go and Docker Hub release workflow. Normal pull requests do not
require an App version change and never publish images, so fork pull requests
can run all checks without access to repository publishing secrets.

## Test a pull request on Home Assistant OS

The normal App configuration points at the official Docker Hub image. Pull
requests validate the App image build without publishing a tag, so adding a
pull-request branch as an App repository is not enough to run the branch on
Home Assistant OS.

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
does not validate anonymous Docker Hub access, the release manifest, or runtime
on a different architecture. Those remain release and physical-hardware gates.

The existing `gaetancollaud/digitalstrom-mqtt` Docker Hub repository is already
public. The release workflow verifies that the exact versioned App manifest can
be fetched anonymously.

## Presentation decisions

- **MQTT selection**: `mqtt:want` permits manual brokers without installing a
  Supervisor MQTT provider. Service mode reads only `/services/mqtt`; manual
  mode reads only the App's broker options. Neither reads HA Core's MQTT config
  entry. The selector values are readable English because the App form does
  not translate list choices; field labels and help text are translated.
  Manual fields stay visible and are ignored in service mode. Both modes use
  the existing TCP client; custom TLS trust configuration remains out of scope.
- **Visible password**: an empty default keeps the dSS password field visible.
  After API-key setup, clear its value instead of deleting the option. Do not
  clear the separate MQTT password, which is needed for subsequent connections.
- **Automatic updates**: after complete controller startup, set `auto_update`
  once for image-based Apps. Supervisor's `build` flag excludes locally built
  test Apps. The `updates_initialized` marker shares the private state file
  below; older watchdog-only files remain compatible. Watchdog and update
  settings are written independently, without changing boot or App options.
  The marker survives container replacement, so later manual choices win.
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
- **Container health check**: the health endpoint remains private to the App
  network. Docker checks `/health/live`, which verifies in-flight bridge work
  without treating a recoverable MQTT outage as a reason to restart.
  `/health/ready` continues to include the MQTT connection state for diagnosis.
  Supervisor only restarts an unhealthy App when its optional watchdog is
  enabled. The App enables it once after complete controller startup, pauses
  it before setup/retries, and preserves a later manual disable. The private
  `/data/watchdog-state.json` records activation and interrupted pauses.
  An unexpected startup crash restores only a recorded pause before handing
  the exit to Supervisor. If restoring fails, the launcher waits for the API
  without starting the bridge again. A crash never marks setup complete.
  Runtime exit code 75 requests a delayed retry; 78 requests user correction.
  An unavailable Supervisor MQTT service also returns 75 through all launcher
  layers; invalid manual settings and unsupported TLS still require correction.
  A failed Supervisor pause blocks further login attempts. Startup is bounded
  Option cleanup failures also return 75: keep the stored key, wait, and retry
  cleanup before starting the bridge. Never keep a temporary dSS password
  configured while normal bridge operation proceeds.
  Startup is bounded
  to two minutes and dSS HTTP requests to 30 seconds. Callback stalls are
  unhealthy after two minutes; idle connections and reconnect delays are not.
- **Store artwork is active**: `icon.png` is a 128 x 128 square icon and
  `logo.png` is a 250 x 100 wide logo. Both images are project-specific artwork
  and do not reuse an official digitalSTROM or Home Assistant logo.

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
  the watchdog helper also stores its state there and calls Supervisor using
  the existing network permission. CI parses this exact profile. A repeat HAOS
  install should confirm the
updated profile before the first public release.

Physical `aarch64` hardware remains untested. CI builds that architecture, but
an actual ARM start and connection test is still required before claiming live
hardware coverage.

## Validation before publication

The automatic watchdog flow has unit and local protocol-fixture coverage.
Password-field defaults, manual/service MQTT selection and once-only update
activation also have local test coverage. Repeat the UI and update path on
HAOS: check the initially visible password, empty field after setup, manual
broker without Mosquitto installed, local-test update exclusion, and both
manually disabled switches after replacing the regular App container.
Before release, repeat the HAOS acceptance run: verify first activation,
successful restart, manual watchdog disable, rejected credentials, Supervisor
API failure, MQTT loss/recovery, and a deliberately blocked callback. Confirm
the AppArmor profile still permits the helper. This new flow has not yet been
validated on a live HAOS instance.

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
8. Verify that the exact versioned Docker Hub App manifest is anonymously
   pullable before announcing a release.

## Release sequence

1. Merge the validated pull request into `master` and update the local branch.
2. Run `bash scripts/release.sh VERSION`, for example
   `bash scripts/release.sh 2.4.0`. This single command verifies a clean and
   current `master`, sets the App version to `2.4.0-haos.1`, adds a matching
   App changelog entry when needed, runs the local checks, creates the release
   commit and tag, and pushes both atomically.
3. Confirm that the GoReleaser and Home Assistant App workflows both complete.
4. Confirm the GitHub release and standalone Docker Hub image use `2.4.0`, and
   the Docker Hub App manifest uses `2.4.0-haos.1`.

The script is the supported release entry point; the App workflow still rejects
a manually pushed project tag when `config.yaml` does not contain the matching
`<tag>-haos.1` App version. Until the tag jobs have published the versioned
Docker Hub App manifest, the App version on `master` is not ready to install or
announce.
