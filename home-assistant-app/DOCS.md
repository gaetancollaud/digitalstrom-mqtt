# digitalSTROM MQTT

This Home Assistant App runs the existing `digitalstrom-mqtt` bridge. It uses Home Assistant's MQTT service and its existing MQTT Discovery support, so digitalSTROM devices appear automatically in Home Assistant.

The first App release is marked experimental until it has completed the real
Home Assistant OS installation and restart checks described below.

## Before you start

- A reachable digitalSTROM Server (dSS) is required.
- Install and start the **Mosquitto Broker** app in Home Assistant.
- Confirm the Home Assistant MQTT integration is connected.
- Do not run this App and another `digitalstrom-mqtt` instance against the same dSS and MQTT topic prefix at the same time.

## First start

1. Enter the dSS address, normally an IP address or local hostname.
2. Keep the default username `dssadmin` unless your dSS uses another account.
3. Enter the dSS password and start the App.

The App creates a dedicated dSS API key and stores it in its private persistent `/data` directory. The temporary password is removed from the App options after a successful setup. Normal restarts use the stored API key and do not need the password again.

## API key recovery

If the API key is revoked in the dSS, enter the dSS password again, enable **Regenerate API key**, and start the App. The old key is kept until creating a replacement succeeds.

## Troubleshooting

- **MQTT service unavailable**: install and start Mosquitto Broker, then verify the MQTT integration in Home Assistant.
- **API key creation fails**: check the dSS address, username and password. The App does not write a replacement key when key creation fails.
- **Home Assistant can switch a device, but manual changes are not reflected**: check that the dSS notification WebSocket on port `8090` is reachable from Home Assistant. Commands use the dSS HTTPS API, normally port `8080`.
- **No devices appear**: wait for the App logs to report a connection to MQTT, then restart the App once. MQTT Discovery is enabled by this App.

## Development and releases

The App manifest references a pre-built multi-architecture image. This is intentional: Home Assistant's local App builder only sees the App directory, while this image must be built from the repository root to include the Go bridge. The release workflow builds the image for `amd64` and `aarch64`; users should receive the published image rather than compiling the bridge on their Home Assistant host.

Before changing the App stage to stable, verify a fresh installation on Home
Assistant OS with a real dSS:

1. Configure only the dSS address, username and password, then confirm the API
   key is created and the password disappears from the App options.
2. Confirm existing MQTT Discovery entities appear in Home Assistant and can
   both send commands and receive dSS state changes.
3. Restart the App and confirm it reconnects using the stored key without a
   password.
4. Try a regeneration with an invalid password and confirm a failed replacement
   leaves the existing key file in place; then regenerate successfully with the
   correct password.
5. Revoke the stored key in the dSS and confirm a successful regeneration
   restores the connection.
6. Stop Mosquitto and confirm the App reports the missing MQTT service without
   exposing credentials in its log.
