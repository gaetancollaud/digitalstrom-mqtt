FROM golang:1.15-buster as builder

ENV GO111MODULE=on
ENV GOFLAGS=-mod=vendor
ENV APP_HOME /go/src/github.com/gaetancollaud/digitalstrom-mqtt

RUN go get -u github.com/golang/dep/cmd/dep

WORKDIR $APP_HOME

COPY . $APP_HOME/


RUN go get -d -v ./...
RUN go install -v ./...

#RUN dep ensure
RUN go build -o dist/digitalstrom-mqtt ./main.go
RUN ls -lh dist/

FROM scratch as bin

COPY --from=builder /go/src/github.com/gaetancollaud/digitalstrom-mqtt/dist/ /

CMD ["/digitalstrom-mqtt"]
