package main

import (
	"github.com/gaetancollaud/digitalstrom-mqtt/config"
	"github.com/gaetancollaud/digitalstrom-mqtt/digitalstrom"
	"github.com/gaetancollaud/digitalstrom-mqtt/digitalstrom_mqtt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"time"
)

func main() {

	// TODO put in config
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	log.Info().Msg("String digitalstrom MQTT!")

	config := config.FromEnv()

	ds := digitalstrom.New(config)
	mqtt := digitalstrom_mqtt.New(&config.Mqtt, ds)

	ds.Start()
	mqtt.Start()

	time.Sleep(100 * 365 * 24 * time.Hour)
}
