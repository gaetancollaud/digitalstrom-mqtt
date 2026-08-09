# Home Assistant App: digitalSTROM MQTT

Connect digitalSTROM devices to Home Assistant through MQTT Discovery.

## About

This App runs `digitalstrom-mqtt` directly on Home Assistant OS. It uses the
Home Assistant MQTT service, creates its own digitalSTROM API key, and publishes
supported devices through MQTT Discovery.

No separate Docker host or command-line setup is required.
