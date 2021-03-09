package main

import (
	"fmt"
	"github.com/gaetancollaud/digitalstrom-mqtt/config"
	"github.com/gaetancollaud/digitalstrom-mqtt/digitalstrom"
)

func main() {
	fmt.Println("String digitalstrom MQTT!")

	config := config.FromEnv()
	tm := digitalstrom.NewTokenManager(config)

	status := tm.RefreshToken()

	fmt.Printf("token %s\n", status)

	//resp, err := http.Get("http://example.com/")
}
