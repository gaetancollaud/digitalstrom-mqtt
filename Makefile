
PLATFORM=local

# Build all files.
build:
	@echo "==> Building..."
	env GOOS=linux GOARCH=amd64 go build -o dist/digitalstrom-mqtt-amd64 ./main.go
.PHONY: build

build-arm:
	@echo "==> Building..."
	env GOOS=linux GOARCH=arm GOARM=5 go build -o dist/digitalstrom-mqtt-arm ./main.go
.PHONY: build

# Install from source.
install:
	@echo "==> Installing digitalstrom-mqtt ${GOPATH}/bin/digitalstrom-mqtt"
	go install ./...
.PHONY: install

# Run all tests.
test:
	go test -timeout 2m ./... && echo "\n==>\033[32m Ok\033[m\n"
.PHONY: test

.PHONY: docker
docker:
	@docker build . --target bin \
	--output bin/ \
	--platform ${PLATFORM}

# Clean.
clean:
	@rm -fr \
		dist \
.PHONY: clean
