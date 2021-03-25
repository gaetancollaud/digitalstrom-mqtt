FROM golang:alpine AS builder

#ENV GO111MODULE=off
#ENV GOFLAGS=-mod=vendor
ENV APP_HOME /go/src/github.com/gaetancollaud/digitalstrom-mqtt

WORKDIR $APP_HOME

COPY . $APP_HOME/


RUN go get -d -v ./...

#RUN dep ensure
RUN go build -o /dist/digitalstrom-mqtt ./main.go
RUN ls -lh /dist/

FROM alpine

WORKDIR /go/bin/

COPY --from=builder /dist/digitalstrom-mqtt /go/bin/digitalstrom-mqtt
COPY config.yaml.example /go/bin/config.yaml

ENTRYPOINT ["/go/bin/digitalstrom-mqtt"]
