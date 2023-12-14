package main

import (
	"flag"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/digitalstrom"
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

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	mode := flag.String("mode", "standard", "Operation mode (standard, get-api-key)")

	host := flag.String("host", "test", "DigitalSTROM server host")
	port := flag.Int("port", 8080, "DigitalSTROM server port")
	username := flag.String("username", "dssadmin", "DigitalSTROM user name")
	password := flag.String("password", "", "DigitalSTROM password")
	integrationName := flag.String("integrationName", "digitalstrom-to-mqtt", "Name of the integration. It will appear in digitalSTROM system panel")

	flag.Parse()

	if *mode == "standard" {
		modeStandard()
	} else if *mode == "get-api-key" {
		modeGetApiKey(*host, *port, *username, *password, *integrationName)
	} else {
		log.Error().Str("mode", *mode).Msg("Unknown mode")
		flag.PrintDefaults()
	}
}

func modeGetApiKey(host string, port int, user string, password string, integrationName string) {
	apiKey, err := digitalstrom.GetApiKey(host, port, user, password, integrationName)
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to get API key.")
	} else {
		log.Info().
			Str("DIGITALSTROM_API_KEY", apiKey).
			Msg("API key successfully retrieved. Please save it in the config file, this cannot be retrieved a second time. You will have to create a new API key.")
	}
}

func modeStandard() {
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

	// Initialize controller responsible for all the bridge logic.
	ctrl := controller.NewController(config)
	if err := ctrl.Start(); err != nil {
		log.Fatal().Err(err).Msg("Error on starting the controller")
	}

	// Subscribe for interruption happening during execution.
	exitSignal := make(chan os.Signal, 2)
	signal.Notify(exitSignal, os.Interrupt, syscall.SIGTERM)
	<-exitSignal

	// Gracefulle stop all the modules loops and logic.
	if err := ctrl.Stop(); err != nil {
		log.Fatal().Err(err).Msg("Error when stopping the controller")
	}
}
