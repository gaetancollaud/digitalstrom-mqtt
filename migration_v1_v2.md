# Migrating from version 1.x to version 2.x

## Reason for a version 2

DigitalSTROM introduced a [new API called Smarthome](https://developer.digitalstrom.org/api/#auth). This API was a
complete rewrite of the previous API. It's more modern, but also different. Some of the features of the old API
don't exists anymore or where renamed. This is the reason of the rewrite of this application and the version 2.0.

## Authentication

Previously you needed to provide the username and password of the digitalSTROM user. Now you need to provide an API key.
This is more safe in a sense that digitalstrom-mqtt will not be aware of your credentials anymore. You can create
a new key using the script provided (see README). You will have to provide a new config called `DIGITALSTROM_API_KEY`.
If you provide the username or password in the config, the app will now complain.

You can now manage the api-key in the digitalSTROM web api under System -> Access Authorization.

## Circuits

Circuits don't exist anymore. They are replaced by the concept of "controllers". In addition, the power consumption
has been moved to a new entity called "metering". Thus, the MQTT interface don't expose circuits anymore but
"meterings". On top of that there is one metering for the "apartment", which basically just sum up all the meterings of
thecontrollers.

## Metering interval

Power and energy consumption are now updated every 10 seconds instead of 30 seconds.

## Scenes

Scenes event are not propagated anymore, so this was removed from the MQTT interface.

## Buttons

Buttons were kind of a hack in the previous version. Since scenes are not exposed anyway it's not possible to redo this
hack and expose button presses anymore. This was also removed

## Devices

Devices are 100% compatible, but if you used home assistant, maybe the ids may have changed. Indeed the new API exposes
"deviceId", "outputId", etc. so it's possible that the mapping of the devices in home assistant has to be redo. The
previous implementation used `dsid` which is fully accessible in the new API anymore.

## Home Assistant

Since home assistant is the de-facto standard for home automation nowadays, it is enabled by default. To better
help home assistant define the state of the devices the MQTT retain flag will also be enabled by default. If you don't
want to use home assistant you can simply disable it. Although having it enabled should not affect you. The
home assistant topic can just be ignored. The rest of the MQTT interface is still the same.

```yaml
MQTT_RETAIN: false
HOME_ASSISTANT_DISCOVERY_ENABLED: false
```