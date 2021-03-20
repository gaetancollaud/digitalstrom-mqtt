
# Build all files.
build:
	@echo "==> Building ./dist/sdm"
	go build -o dist/digitalstrom-mqtt-amd64 ./main.go
.PHONY: build

build-arm:
	@echo "==> Building ./dist/sdm"
	env GOOS=linux GOARCH=arm GOARM=5 go build -o dist/digitalstrom-mqtt-arm ./main.go
.PHONY: build

# Install from source.
install:
	@echo "==> Installing digitalstrom-mqtt ${GOPATH}/bin/digitalstrom-mqtt"
	go install ./...
.PHONY: install

# Install the development dependencies.
install.deps:
	@echo "==> Installing dev dependencies"
	go get -u github.com/golang/dep/cmd/dep
.PHONY: install.deps

# Run all tests.
test:
	go test -timeout 2m ./... && echo "\n==>\033[32m Ok\033[m\n"
.PHONY: test

# Clean.
clean:
	@rm -fr \
		dist \
.PHONY: clean
