# Changelog

## 2.4.0-haos.1

- Initial native Home Assistant App package.
- Published as a stable, unofficial community App.
- Uses the Home Assistant MQTT service and existing MQTT Discovery support.
- Supports a manually configured MQTT broker without requiring the Mosquitto App.
- Keeps the digitalSTROM password field visible and clears its value after setup.
- Enables automatic updates once for the regular App and preserves later manual choices.
- Creates and persists a dedicated digitalSTROM API key in the App data directory.
- Recovers interrupted or incomplete API-key setup without overwriting a valid key.
- Keeps passwords, API keys, and Supervisor option payloads out of App logs.
- Supports a configurable digitalSTROM HTTPS API port.
- Adds actionable API-key bootstrap and recovery logs without exposing credentials.
- Includes Home Assistant store icon and logo artwork.
- Runs under a custom AppArmor profile based on an enforce-mode profile validated
  on Home Assistant OS; CI parses the final packaged profile.
- Adds a Docker health check through a dependency-independent liveness endpoint.
- Enables the watchdog after successful setup and detects blocked bridge work.
- Pauses automatic restarts for rejected credentials and preserves manual watchdog choices.
- Retries an unavailable MQTT service and restores paused watchdog protection after startup crashes.
- Rejects unsupported MQTT service TLS instead of starting with an unusable connection.
- Includes complete English and German setup, migration, and removal guidance.
