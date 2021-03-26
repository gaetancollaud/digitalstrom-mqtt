FROM alpine
ENTRYPOINT ["/digitalstrom-mqtt"]
COPY digitalstrom-mqtt config.yaml.example /
