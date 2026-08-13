ARG BUILD_FROM=ghcr.io/home-assistant/base:3.23

FROM --platform=$BUILDPLATFORM golang:1.25.0-alpine AS build

ARG TARGETOS
ARG TARGETARCH

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /out/digitalstrom-mqtt ./main.go

FROM ${BUILD_FROM}

ARG BUILD_ARCH
ARG BUILD_VERSION

COPY --from=build /out/digitalstrom-mqtt /digitalstrom-mqtt
COPY home-assistant-app/run.sh /run.sh

RUN chmod 0755 /digitalstrom-mqtt /run.sh

LABEL \
  io.hass.type="app" \
  io.hass.name="digitalSTROM MQTT" \
  io.hass.description="Unofficial digitalSTROM MQTT bridge for Home Assistant" \
  io.hass.url="https://github.com/gaetancollaud/digitalstrom-mqtt/tree/master/home-assistant-app" \
  io.hass.version="${BUILD_VERSION}" \
  io.hass.arch="${BUILD_ARCH}" \
  org.opencontainers.image.title="digitalSTROM MQTT" \
  org.opencontainers.image.description="digitalSTROM MQTT bridge for Home Assistant" \
  org.opencontainers.image.source="https://github.com/gaetancollaud/digitalstrom-mqtt" \
  org.opencontainers.image.licenses="AGPL-3.0-or-later"

CMD ["/run.sh"]
