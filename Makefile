
GO ?= go
PROTOC ?= protoc

# Build all files.
build:
	@echo "==> Building ./dist/sdm"
	@$(GO) build -o dist/digitalstrom-mqtt ./main.go
.PHONY: build

# Install from source.
install:
	@echo "==> Installing digitalstrom-mqtt ${GOPATH}/bin/digitalstrom-mqtt"
	@$(GO) install ./...
.PHONY: install

# Install the development dependencies.
install.deps:
	@echo "==> Installing dev dependencies"
	@$(GO) get -u github.com/golang/dep/cmd/dep
.PHONY: install.deps

# Run all tests.
test:
	@$(GO) test -timeout 2m ./... && echo "\n==>\033[32m Ok\033[m\n"
.PHONY: test

# Clean.
clean:
	@rm -fr \
		dist \
.PHONY: clean
