# Home assistant integration

## Example of configuration

```yaml
- platform: mqtt
  device_class: shutter
  name: "Ρολό Κουζίνας"
  command_topic: "digitalstrom/devices/Roller_Shutter_Kitchen/shadePositionOutside/command"
  position_topic: "digitalstrom/devices/Roller_Shutter_Kitchen/shadePositionOutside/state"
  set_position_topic: "digitalstrom/devices/Roller_Shutter_Kitchen/shadePositionOutside/command"
  payload_open: "100"
  payload_close: "0"
  payload_stop: "STOP"
  state_open: "open"
  state_opening: "opening"
  state_closed: "closed"
  state_closing: "closing"
  qos: 0
  retain: true
  optimistic: false
```

## References

* [comment in #20](https://github.com/gaetancollaud/digitalstrom-mqtt/issues/20#issuecomment-1013740593)