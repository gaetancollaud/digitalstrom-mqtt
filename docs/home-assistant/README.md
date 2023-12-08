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
## Example of working configuration for GE-UMV200 based tunable white light device leveraging 3 channels of the GE-UMV200 (first for on/off switch, second for brightness, third for temperature)
```yaml
mqtt:
  - light:
      name: "Spots Arbeitszimmer"
      unique_id: "Spots_Arbeitszimmer"
      state_value_template: "{{ '0' if value == '0' else '100' }}"
      state_topic: "digitalstrom/devices/Spots_An_Aus_Arbeitszimmer/brightness/state"
      on_command_type: first
      payload_on: "100"
      payload_off: "0"
      command_topic: "digitalstrom/devices/Spots_An_Aus_Arbeitszimmer/brightness/command"
      device:
        configuration_url: https://192.168.1.10
        manufacturer: DigitalStrom
        model: GE-UMV200
        name: "Spots Arbeitszimmer"
        identifiers:
          - 302ed89f43f00f00000f4208
          - 302ed89f43f0000000000f00000f420800
      brightness_scale: 100
      brightness_state_topic: "digitalstrom/devices/Spots_Dimmer_Arbeitszimmer/brightness/state"
      brightness_command_topic: "digitalstrom/devices/Spots_Dimmer_Arbeitszimmer/brightness/command"
      color_temp_command_template: "{{ 100-(0.288*(value-153)) }}"
      color_temp_value_template: "{{ value*0.288 + 153}}"
      color_temp_state_topic: "digitalstrom/devices/Spots_Farbe_Arbeitszimmer/brightness/state"
      color_temp_command_topic: "digitalstrom/devices/Spots_Farbe_Arbeitszimmer/brightness/command"
      optimistic: true
```
Some explanations:
- state_value_template makes sure to set correct payload for on/off commands of entity/device
- identifiers are derived from ds for the first UMV output channel
- color_temp_command_template calculates a value between 0 and 100 from the mireds value used in HASS to be send to DS
- color_temp_value_template calculates a mireds value (between 153 and 500) out of the ds channel state of the 3rd UMV output channel (0-100) to meet HASS color_temp handling

- 
## References

* [comment in #20](https://github.com/gaetancollaud/digitalstrom-mqtt/issues/20#issuecomment-1013740593)
