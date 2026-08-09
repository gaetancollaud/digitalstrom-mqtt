# digitalSTROM MQTT

This Home Assistant App runs the existing `digitalstrom-mqtt` bridge. It uses
Home Assistant's MQTT service and MQTT Discovery, so supported digitalSTROM
devices appear automatically in Home Assistant.

## Before you start

- A reachable digitalSTROM Server (dSS) is required.
- Install and start the **Mosquitto Broker** App in Home Assistant.
- Confirm the Home Assistant MQTT integration is connected.
- Do not run this App and another `digitalstrom-mqtt` instance against the
  same dSS and MQTT topic prefix at the same time.

## First start

1. Enter the dSS address, normally an IP address or local hostname. Keep the
   default HTTPS API port `8080` unless your dSS uses another port.
2. Keep the default username `dssadmin` unless your dSS uses another account.
3. Enter the dSS password and start the App.

The App creates a dedicated dSS API key and stores it in its private persistent
`/data` directory. The temporary password is removed from the App options after
successful setup. If Home Assistant cannot update the options immediately, the
App retries that cleanup on its next start without creating another key. Normal
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

Required only when creating or regenerating the stored API key. The App removes
the password from its options after successful key creation.

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

## API key recovery

If the API key is revoked in the dSS, enter the dSS password again, enable
**Regenerate API key**, and start the App. The option resets automatically after
a replacement key has been stored and the temporary password has been removed.

## Troubleshooting

- **MQTT service unavailable**: install and start Mosquitto Broker, then verify
  the MQTT integration in Home Assistant.
- **API key creation fails**: check the dSS address, username and password. The
  App does not replace the existing key when key creation fails.
- **Home Assistant can switch a device, but manual changes are not reflected**:
  check that the dSS notification WebSocket on port `8090` is reachable from
  Home Assistant. Commands use the dSS HTTPS API, normally port `8080`.
- **No devices appear**: wait for the App log to report connections to the dSS
  and MQTT. Confirm that no second bridge publishes the same MQTT entities.
- **The App stops after changing options**: inspect the App log. Configuration
  and connection errors are reported without requiring access to the container.

When reporting an issue, include the App version, dSS version, relevant App log
lines, and whether the problem also occurs after restarting the App. Remove IP
addresses or device identifiers if you do not want to publish them.

## License

This App and `digitalstrom-mqtt` are licensed under the
[GNU Affero General Public License v3.0 or later](https://github.com/gaetancollaud/digitalstrom-mqtt/blob/master/LICENSE).
