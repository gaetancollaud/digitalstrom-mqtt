package main

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	_ "net/http/pprof"

	"golang.org/x/net/context"

	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/config"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/controller"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/debug"
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

	// Add profiling server for live profile of the program.
	debugServerExitDone := &sync.WaitGroup{}
	debugServerExitDone.Add(1)
	srv, isReady := debug.StartDebugServer(debugServerExitDone)

	// Initialize controller responsible for all the bridge logic.
	controller := controller.NewController(config)
	if err := controller.Start(); err != nil {
		log.Fatal().Err(err).Msg("Error on starting the controller")
	}
	isReady.Store(true)

	// Subscribe for interruption happening during execution.
	exitSignal := make(chan os.Signal, 2)
	signal.Notify(exitSignal, os.Interrupt, syscall.SIGTERM)
	<-exitSignal

	// Gracefulle stop all the modules loops and logic.
	log.Info().Msg("Shutting down controller...")
	if err := controller.Stop(); err != nil {
		log.Fatal().Err(err).Msg("Error when stopping the controller")
	}

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	log.Info().Msg("Shutting down debug server...")
	if err := srv.Shutdown(ctx); err != nil {
		panic(err) // failure/timeout shutting down the server gracefully.
	}

	debugServerExitDone.Wait()
	log.Info().Msg("Done exiting.")
}
