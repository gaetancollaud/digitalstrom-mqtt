# Changelog

## 2.4.0

- Initial native Home Assistant App package.
- Published as a stable, unofficial community App.
- Uses the Home Assistant MQTT service and existing MQTT Discovery support.
- Creates and persists a dedicated digitalSTROM API key in the App data directory.
- Resumes interrupted API-key option cleanup without generating another key.
- Supports a configurable digitalSTROM HTTPS API port.
- Adds actionable API-key bootstrap and recovery logs without exposing credentials.
- Includes Home Assistant store icon and logo artwork.
- Runs under a custom AppArmor profile based on an enforce-mode profile validated
  on Home Assistant OS; CI parses the final packaged profile.
- Adds Supervisor process monitoring through a dependency-independent liveness endpoint.
- Includes complete English and German setup, migration, and removal guidance.
