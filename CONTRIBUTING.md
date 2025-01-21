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
CGO_ENABLED=0 GOOS=linux GOARCH=amd64
docker compose build
```

