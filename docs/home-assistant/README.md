# Home Assistant Integration

`digitalstrom-mqtt` supports [MQTT Discovery from Home Assistant](https://www.home-assistant.io/docs/mqtt/discovery/) but it is not activated by default. In order to enable it, make sure you set the following environmental variable:

```yaml
HOME_ASSISTANT_DISCOVERY_ENABLED: true
# You can also customize the prefix for the MQTT discovery topic:
HOME_ASSISTANT_DISCOVERY_PREFIX: "homeassistant"
# In case you would like to remove some parts of the name that gets published
# into Home Assistant, there is an option to provice a regex that will be use
# to remove it from the entity name. This way "Location Light" could be
# translated in Home Assistant as `light.location` rather than
# `light.location_light`.
HOME_ASSISTANT_REMOVE_REGEXP_FROM_NAME: "(light|cover|blind)"
```

## Example of configuration

If you still want to configure manually the entities, here there is an example:

```yaml
- platform: mqtt
  device_class: shutter
  name: "Roller Shutter Kitchen"
  state_topic: "digitalstrom/devices/Roller_Shutter_Kitchen/shadePositionOutside/state"
  command_topic: "digitalstrom/devices/Roller_Shutter_Kitchen/shadePositionOutside/command"
  position_topic: "digitalstrom/devices/Roller_Shutter_Kitchen/shadePositionOutside/state"
  set_position_topic: "digitalstrom/devices/Roller_Shutter_Kitchen/shadePositionOutside/command"
  payload_open: "100"
  payload_close: "0"
  payload_stop: "STOP"
  state_open: "100.00"
  state_closed: "0.00"
  qos: 0
  retain: true
```

## References

* [comment in #20](https://github.com/gaetancollaud/digitalstrom-mqtt/issues/20#issuecomment-1013740593)