package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/config"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/controller"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {

	// TODO put in config
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	config, err := config.ReadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Error found when reading the config.")
	}

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

	log.Info().Msg("Starting DigitalStrom MQTT!")

	// ds := digitalstrom.New(config)
	// mqtt := mqtt.New(config, ds)

	// ds.Start()
	// mqtt.Start()
	controller := controller.NewController(config)
	if err := controller.Start(); err != nil {
		log.Fatal().Err(err).Msg("Error on starting the controller")
	}

	// Subscribe for interruption happening during execution.
	exitSignal := make(chan os.Signal, 2)
	signal.Notify(exitSignal, os.Interrupt, syscall.SIGTERM)
	<-exitSignal

	// Gracefulle stop the connections.
	// ds.Stop()
	// mqtt.Stop()
	if err := controller.Stop(); err != nil {
		log.Fatal().Err(err).Msg("Error when stopping the controller")
	}
}
