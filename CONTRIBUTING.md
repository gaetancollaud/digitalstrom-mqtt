## Development

### Checkout

``` bash
git@github.com:gaetancollaud/digitalstrom-mqtt.git
```

### Config file
Copy and adapt the config file

```shell
cp config.yaml.example config.yaml
```

### Run the go program

```shell
go install
go run .
```

### Build for docker

```shell
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build
docker compose build
```

### Test the Home Assistant App

Pull requests do not publish the Home Assistant App image. To generate a local
App from an exact Git commit and test it on Home Assistant OS, follow
[`home-assistant-app/DEVELOPMENT.md`](home-assistant-app/DEVELOPMENT.md#test-a-pull-request-on-home-assistant-os).

