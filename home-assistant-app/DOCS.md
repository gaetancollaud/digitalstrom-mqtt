# digitalSTROM MQTT

[English](https://github.com/gaetancollaud/digitalstrom-mqtt/blob/master/home-assistant-app/DOCS.md) |
[Deutsch](https://github.com/gaetancollaud/digitalstrom-mqtt/blob/master/home-assistant-app/DOCS.de.md)

This unofficial community App runs the existing `digitalstrom-mqtt` bridge on
Home Assistant OS. It is not provided or supported by digitalSTROM. The App
uses the Home Assistant MQTT service or a manually configured broker and MQTT
Discovery, so supported digitalSTROM devices appear automatically in Home Assistant.

## Compatibility

- Home Assistant OS with Apps support is required.
- Pre-built images are published for `amd64` and `aarch64`.
- The complete installation and restart path has been tested in a fresh
  `amd64` Home Assistant OS VM with a real dSS.
- The `aarch64` image is built in CI. A live start on physical ARM hardware has
  not yet been tested by the maintainers.

## Install the App repository

[![Open your Home Assistant instance and add this App repository.](https://my.home-assistant.io/badges/supervisor_add_addon_repository.svg)](https://my.home-assistant.io/redirect/supervisor_add_addon_repository/?repository_url=https%3A%2F%2Fgithub.com%2Fgaetancollaud%2Fdigitalstrom-mqtt)

Alternatively, open **Settings -> Apps -> App store -> three-dot menu ->
Repositories** and add:

```text
https://github.com/gaetancollaud/digitalstrom-mqtt
```

Install **digitalSTROM MQTT** after the repository appears.

## Before you start

- A reachable digitalSTROM Server (dSS) is required.
- Use the **Mosquitto Broker** App in Home Assistant or an existing MQTT broker.
- Confirm that the Home Assistant MQTT integration is connected to that same broker.
- Stop any existing `digitalstrom-mqtt` instance using the same dSS and MQTT
  topic prefix. Two active bridges can publish conflicting state and discovery
  messages.

## First start

1. Enter the dSS address, normally an IP address or local hostname. Keep the
   default HTTPS API port `8080` unless your dSS uses another port.
2. Keep the default username `dssadmin` unless your dSS uses another account.
3. Enter the dSS password in the visible password field.
4. Keep **Home Assistant MQTT service** when using the Mosquitto Broker App.
   Otherwise select **Manual configuration** and enter the broker details below.
5. Save the options and start the App.

The App creates a dedicated dSS API key and stores it in its private persistent
`/data` directory. The temporary password is cleared after successful setup;
the empty field remains visible. If Home Assistant cannot update the options
immediately, the App retries cleanup on its next start without creating another key. Normal
restarts use the stored API key and do not need the password again.

## Configuration

### digitalSTROM Server address

IP address or hostname of the dSS. This setting is required.

### digitalSTROM Server port

HTTPS API port of the dSS. The default is `8080`.

### digitalSTROM username

dSS account used once to create the dedicated API key. The default is
`dssadmin`.

### digitalSTROM password

Required only when creating or regenerating the stored API key. The App clears
the field after successful key creation. Leave it empty on later starts.

### MQTT connection

- **Home Assistant MQTT service** (default): obtains the broker address and
  credentials from Supervisor, normally from the Mosquitto Broker App. Manual
  MQTT fields below are ignored in this mode.
- **Manual configuration**: enter the broker hostname or IP address, port
  (default `1883`), username and password. Leave credentials empty only if the
  broker permits anonymous connections. Enter IPv6 addresses without brackets.
  This mode does not require the Mosquitto Broker App and never falls back to it.

The App does not copy the connection configured in Home Assistant's MQTT
integration. If that integration uses an external broker, enter that broker
here too. Both must use the same broker unless you manage MQTT bridging yourself.
The MQTT password remains stored for reconnects; only the temporary dSS password
is cleared after setup. Manual connections use plain TCP on a trusted local
network. TLS and custom certificates are not supported by these App options.

### Invert blind position

Enable this when Home Assistant should interpret `100%` as fully closed instead
of fully open. This changes the reported and commanded cover position.

### Enable metering sensors

Publishes available digitalSTROM consumption and power values to MQTT. Metering
uses periodic dSS requests because these values are not provided by the normal
event stream.

### Metering interval

Number of seconds between metering requests. The default is `10`. A shorter
interval produces fresher values but increases load on the dSS.

### Log level

Controls App log detail. Use `INFO` for normal operation and `DEBUG` only while
diagnosing a problem. Logs do not intentionally include passwords or API keys.

### Regenerate API key

Creates a replacement API key on the next App start. Enter the dSS password
first. The existing key remains in place if replacement fails.

## Automatic recovery

After the first complete startup, the App enables Home Assistant's watchdog
automatically. No extra switch is required. If you later disable the watchdog
manually, the App preserves that choice, including across restarts and updates.

During startup and API-key setup, the watchdog is paused. Rejected credentials
or invalid configuration stop the App with an explanation in its log. Correct
the settings and start it again; protection resumes after successful startup.
Temporary connection failures retry with increasing delays of 15 to 60 seconds.

The container health check detects unresponsive health requests and bridge work
that remains blocked for more than two minutes, including notification and
command callbacks. Home Assistant can then restart the App. A quiet home or a
disconnected MQTT broker alone does not count as a hang. This check cannot
detect every possible device or protocol failure.

## Automatic updates

After the first complete startup, the regular App enables Home Assistant's
automatic updates once. Home Assistant can then install published App versions
without another confirmation. If you later switch automatic updates off, the
App leaves them off, including after a manual update or restart. Locally built
Apps, including the PR test copy, do not enable automatic updates themselves.
Uninstalling the App and deleting its data resets this first-start state.

## Migrating an existing bridge

1. Record any non-default MQTT topic, discovery prefix, or name normalization
   settings used by the existing installation.
2. Stop the existing bridge before starting this App.
3. Install and configure the App, then verify the discovered devices and a few
   commands in Home Assistant.
4. Keep the old installation stopped until the App has also survived a restart.
5. Remove the old installation and revoke its dSS API key when it is no longer
   needed.

The App currently uses the normal `digitalstrom-mqtt` MQTT and Discovery
defaults. An installation with custom topic prefixes cannot be migrated
identically through the App options yet.

## API key recovery

If the API key is revoked in the dSS, enter the dSS password again, enable
**Regenerate API key**, and start the App. The option resets automatically after
a replacement key has been stored and the temporary password has been removed.

## Removing the App

Stop the App before uninstalling it. Uninstalling does not automatically revoke
the dedicated API key in the dSS or remove retained MQTT Discovery messages.
Revoke the integration key named `digitalstrom-mqtt-home-assistant` in the dSS
authorization management when it is no longer used. Remove retained Discovery
messages only after confirming that no other bridge depends on them.

## Troubleshooting

- **MQTT service unavailable**: start Mosquitto Broker, or select **Manual
  configuration** for your existing broker. An external broker configured only
  in Home Assistant's MQTT integration is not automatically available to the App.
- **API key creation fails**: check the dSS address, username and password. The
  App does not replace the existing key when key creation fails.
- **Home Assistant can switch a device, but manual changes are not reflected**:
  check that the dSS notification WebSocket on port `8090` is reachable from
  Home Assistant. Commands use the dSS HTTPS API, normally port `8080`.
- **No devices appear**: wait for the App log to report connections to the dSS
  and MQTT. Confirm that no second bridge publishes the same MQTT entities.
- **MQTT stays disconnected**: verify the broker and MQTT credentials. Once the
  bridge is running, its container health check stays independent of MQTT, so a
  broker outage does not mark the container unhealthy. The App supports the
  Mosquitto App's default internal non-TLS service; a service that requires a
  custom TLS certificate is rejected with an explicit log message.
- **The App stops after changing options**: inspect the App log. Configuration
  and connection errors are reported without requiring container access.

When reporting an issue, include the App version, dSS version, relevant App log
lines, and whether the problem also occurs after restarting the App. Remove IP
addresses or device identifiers if you do not want to publish them.

## License

This App and `digitalstrom-mqtt` are licensed under the
[GNU Affero General Public License v3.0 or later](https://github.com/gaetancollaud/digitalstrom-mqtt/blob/master/LICENSE).
