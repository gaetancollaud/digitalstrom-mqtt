package main

import (
	"fmt"
	"github.com/gaetancollaud/digitalstrom-mqtt/config"
	"github.com/gaetancollaud/digitalstrom-mqtt/digitalstrom"
)

func main() {
	fmt.Println("Hello, World!")

	config := config.FromEnv()
	tm := digitalstrom.NewTokenManager(config)
	fmt.Printf("Config  %s\n", config.String())

	status := tm.RefreshToken()

	fmt.Printf("status %s\n", status)

	//resp, err := http.Get("http://example.com/")
}
