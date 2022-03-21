package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gaetancollaud/digitalstrom-mqtt/config"
	"github.com/gaetancollaud/digitalstrom-mqtt/digitalstrom"
	"github.com/gaetancollaud/digitalstrom-mqtt/digitalstrom_mqtt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {

	// TODO put in config
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	config := config.FromEnv()

	if config.LogLevel == "TRACE" {
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	} else if config.LogLevel == "DEBUG" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else if config.LogLevel == "INFO" {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	} else if config.LogLevel == "WARN" {
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	} else if config.LogLevel == "ERROR" {
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	}

	log.Info().Msg("String digitalstrom MQTT!")

	ds := digitalstrom.New(config)
	mqtt := digitalstrom_mqtt.New(config, ds)

	ds.Start()
	mqtt.Start()

	// Subscribe for interruption happening during execution.
	exitSignal := make(chan os.Signal)
	signal.Notify(exitSignal, os.Interrupt, syscall.SIGTERM)
	<-exitSignal

	// Gracefulle stop the connections.
	mqtt.Stop()
}
