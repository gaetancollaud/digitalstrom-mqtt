# DigitalSTROM MQTT

This application allows you to set and react to any DigitalSTROM devices using MQTT.

You can set the output values using the command topic and get the current value using the state topic.

![](./docs/images/mqtt-explorer.png)

## Migrating from version 1.x to version 2.x

// TODO

## Motivation

[DigitalSTROM](https://www.digitalstrom.com/en/) system is built upon scenes. You press a button, and a scene starts.
The scene can trigger as many output devices as you want. While this is fine for a standalone system, it’s really
difficult to integrate with a more complex automation system. Basically, if you want the master of your automation to be
an external system, you will have a bad time.

DigitalSTROM provides a [REST api](https://developer.digitalstrom.org/api/), but it’s not that easy to use since the
structures of the element can be quite complex. There is also a notification endpoint that uses websocket to alert you
of any value change. This app uses latest version of the REST API and the notification endpoint to provide a simple MQTT
interface.

Currently, digitalSTROM integrations with home automation systems are rare and sometimes limited. The intent of this app
is to solve this issue as all of them support MQTT.

## Concept

The main goal of this application is to have direct access to the output devices (light, blinds,...) and be notified
when something changes. All of this using MQTT as it’s widely used in home automation systems.

This application will not reflect the internal functioning of digitalSTROM. It will rather try to make an abstraction of
it.

### Technical

This app use the [Smarthome API from digitalSTROM](https://developer.digitalstrom.org/api/). It doesn't use the old
`json/device/` api. The documentation is sometimes lacking, so I had to browser various forum and discussion groups to
find all the relevant information.

## Configuration

You have two ways of configuring the app. Either using a `config.yaml` file next to the executable or with environment
variables.

| required | property                               | description                                                                      | default         | example                     |
|----------|----------------------------------------|----------------------------------------------------------------------------------|-----------------|-----------------------------|
| *        | DIGITALSTROM_HOST                      | Ip address of the digitalstrom system                                            |                 | 192.168.1.10                |
|          | DIGITALSTROM_PORT                      | Secure port of the rest API                                                      | 8080            |                             |
| *        | DIGITALSTROM_API_KEY                   | DigitalSTROM API key                                                             |                 | 782f...6075d                |
| *        | MQTT_URL                               | MQTT url                                                                         |                 | tcp://192.168.1.20:1883     |
|          | MQTT_USERNAME                          | MQTT username                                                                    |                 | myUser                      |
|          | MQTT_PASSWORD                          | MQTT password                                                                    |                 | 9TyVg74e5S                  |
|          | MQTT_TOPIC_PREFIX                      | Topic prefix                                                                     | digitalstrom    |                             |
|          | MQTT_NORMALIZE_DEVICE_NAME             | Remove special chars from device name                                            | true            |                             |
|          | MQTT_RETAIN                            | Retain MQTT messages                                                             | true            |                             |
|          | REFRESH_AT_START                       | should the states be refreshed at start                                          | true            |                             |
|          | LOG_LEVEL                              | log level                                                                        | INFO            | TRACE,DEBUG,INFO,WARN,ERROR |
|          | INVERT_BLINDS_POSITION                 | 100% is fully close                                                              | false           |                             |
|          | HOME_ASSISTANT_DISCOVERY_ENABLED       | Whether or not publish MQTT Discovery messages for Home Assistant                | true            |                             |
|          | HOME_ASSISTANT_DISCOVERY_PREFIX        | Topic prefix where to publish the MQTT Discovery messaged for Home Assistant     | `homeassistant` |                             |
|          | HOME_ASSISTANT_REMOVE_REGEXP_FROM_NAME | Regular expression to remove from device names when announcing to Home Assistant |                 | `"(light\|cover)"`          

## Obtaining the API key

// TODO

## Minimal config file

config.yaml

```yaml
DIGITALSTROM_HOST: 192.168.1.x
DIGITALSTROM_API_KEY: XXX
MQTT_URL: tcp://192.168.1.X:1883
```

### MQTT topic format

The topic format is as follows for the devices:

`{prefix}/devices/{deviceName}/{channel}/{commandState}`

The topic format is as follows for the meterings:

`{prefix}/meterings/{deviceName}/{channel}/state`

The server status topic is

`{prefix}/server/status`

## How to run

### Using the binary

Go to [Releases](https://github.com/gaetancollaud/digitalstrom-mqtt/releases), download and unzip the latest version for
your OS. Create the config file as shown above.

Start the executable

```shell
./digitalstrom-mqtt
```

### Using docker

```shell
docker run \
  -e DIGITALSTROM_HOST=192.168.1.x \
  -e DIGITALSTROM_API_KEY=XXX \
  -e MQTT_URL=tcp://192.168.1.X:1883 \
  gaetancollaud/digitalstrom-mqtt
```

### Home automation integration examples

* [Home Assistant](./docs/home-assistant/README.md)
* [OpenHAB](./docs/openhab/README.md)

### MQTT-Explorer

We recommend using https://mqtt-explorer.com/ if you want a simple interface for MQTT.

## Topics

### GE devices (lights)

```
digitalstrom/devices/DEVICE_NAME/brightness/state
digitalstrom/devices/DEVICE_NAME/brightness/command
```

### GR devices (blinds)

```
digitalstrom/devices/DEVICE_NAME/shadePositionOutside/state
digitalstrom/devices/DEVICE_NAME/shadePositionOutside/command
digitalstrom/devices/DEVICE_NAME/shadeOpeningAngleOutside/state
digitalstrom/devices/DEVICE_NAME/shadeOpeningAngleOutside/command
```

### dSS20 (meterings)

```
digitalstrom/meterings/chambres/consumptionW/state
digitalstrom/meterings/chambres/energyWs/state
```

## Tested devices

digitalSTROM-MQTT was tested successfully with these devices:

* dSM12
* dSS20
* GE-KM200
* GE-TKM210
* GE-UMV200 (see [#22](https://github.com/gaetancollaud/digitalstrom-mqtt/issues/22))
* GE-KL200 (see [#23](https://github.com/gaetancollaud/digitalstrom-mqtt/issues/23))
* SW-TKM200
* SW-TKM210
* GR-KL200
* GR-KL210
* GR-KL220
* GN-KM200 (see [#21](https://github.com/gaetancollaud/digitalstrom-mqtt/issues/21))

Some devices are known to have issues or limitations:

* BL-KM300 (see [#7](https://github.com/gaetancollaud/digitalstrom-mqtt/issues/7) [#19](https://github.com/gaetancollaud/digitalstrom-mqtt/issues/19))
* GE-UMv200 (see [#22](https://github.com/gaetancollaud/digitalstrom-mqtt/issues/22))

Feel free to create an issue or to directly edit this file if you have tested this software with your devices.

## Development

See [CONTRIBUTION.md](./CONTRIBUTION.md)
