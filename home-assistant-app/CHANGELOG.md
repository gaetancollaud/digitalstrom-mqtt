# Changelog

## 2.3.3-haos.1

- Initial native Home Assistant App package.
- Uses the Home Assistant MQTT service and existing MQTT Discovery support.
- Creates and persists a dedicated digitalSTROM API key in the App data directory.
- Resumes interrupted API-key option cleanup without generating another key.
- Supports a configurable digitalSTROM HTTPS API port.
- Adds actionable API-key bootstrap and recovery logs without exposing credentials.
- Includes Home Assistant store icon and logo artwork.
- Runs under a custom AppArmor profile validated on Home Assistant OS.
