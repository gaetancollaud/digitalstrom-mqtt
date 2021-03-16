package main

import (
	"fmt"
	"github.com/gaetancollaud/digitalstrom-mqtt/config"
	"github.com/gaetancollaud/digitalstrom-mqtt/digitalstrom"
	"time"
)

func main() {
	fmt.Println("String digitalstrom MQTT!")

	config := config.FromEnv()

	ds := digitalstrom.New(config)
	ds.Start()

	time.Sleep(100 * 365 * 24 * time.Hour)
}
