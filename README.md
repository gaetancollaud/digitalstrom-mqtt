# Digitalstrom MQTT

## THIS IS A WORK IN PROGRESS

The goal of this tool is to be able to sync a digitalstrom installation with MQTT.

## Config file
config.yaml
```yaml
DIGITALSTROM_IP: 192.168.1.x
DIGITALSTROM_USERNAME: dssadmin
DIGITALSTROM_PASSWORD: XXX
MQTT_URL: tcp://192.168.1.X:1883

```

## Development

```shell
Make install.deps
dep ensure
go run .
```

### Compile

```shell
make build-amd64
make build-arm
```
