FROM scratch
ENTRYPOINT ["/digitalstrom-mqtt"]
COPY digitalstrom-mqtt /
COPY config.yaml.example /config.yaml
