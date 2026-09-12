package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/appwatchdog"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/config"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/controller"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/digitalstrom"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/monitor"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	mode := flag.String("mode", "standard", "Operation mode (standard, get-api-key, app-watchdog-pause)")

	host := flag.String("host", "test", "DigitalSTROM server host")
	port := flag.Int("port", 8080, "DigitalSTROM server port")
	username := flag.String("username", "dssadmin", "DigitalSTROM user name")
	password := flag.String("password", "", "DigitalSTROM password")
	passwordFile := flag.String("password-file", "", "Path to a file containing the DigitalSTROM password")
	apiKeyFile := flag.String("api-key-file", "", "Path where the generated DigitalSTROM API key is stored")
	integrationName := flag.String("integrationName", "digitalstrom-to-mqtt", "Name of the integration. It will appear in digitalSTROM system panel")

	flag.Parse()

	if *mode == "standard" {
		if err := modeStandard(); err != nil {
			log.Error().Err(err).Msg("Bridge stopped")
			if os.Getenv("HOME_ASSISTANT_APP") == "true" {
				os.Exit(monitor.ExitCode(err))
			}
			os.Exit(1)
		}
	} else if *mode == "app-watchdog-pause" {
		if err := appwatchdog.FromEnvironment().Pause(context.Background()); err != nil {
			log.Error().Err(err).Msg("Cannot pause App watchdog; startup must wait")
			os.Exit(75)
		}
	} else if *mode == "get-api-key" {
		request := apiKeyRequest{
			host:            *host,
			port:            *port,
			username:        *username,
			password:        *password,
			passwordFile:    *passwordFile,
			integrationName: *integrationName,
			apiKeyFile:      *apiKeyFile,
		}
		if err := modeGetApiKey(request); err != nil {
			log.Error().Err(err).Msg("Unable to get API key")
			if os.Getenv("HOME_ASSISTANT_APP") == "true" {
				os.Exit(monitor.ExitCode(err))
			}
			os.Exit(1)
		}
	} else {
		log.Error().Str("mode", *mode).Msg("Unknown mode")
		flag.PrintDefaults()
	}
}

type apiKeyRequest struct {
	host            string
	port            int
	username        string
	password        string
	passwordFile    string
	integrationName string
	apiKeyFile      string
}

func modeGetApiKey(request apiKeyRequest) error {
	password, err := request.readPassword()
	if err != nil {
		return monitor.Permanent(err)
	}

	apiKey, err := digitalstrom.GetApiKey(
		request.host,
		request.port,
		request.username,
		password,
		request.integrationName,
	)
	if err != nil {
		return err
	}

	if request.apiKeyFile != "" {
		if err := writePrivateFile(request.apiKeyFile, apiKey); err != nil {
			return monitor.Permanent(fmt.Errorf("store API key: %w", err))
		}
		log.Info().Str("path", request.apiKeyFile).Msg("API key successfully retrieved and stored")
		return nil
	}

	log.Info().
		Str("DIGITALSTROM_API_KEY", apiKey).
		Msg("API key successfully retrieved. Please save it in the config file, this cannot be retrieved a second time. You will have to create a new API key.")
	return nil
}

func (request apiKeyRequest) readPassword() (string, error) {
	if request.passwordFile == "" {
		return request.password, nil
	}
	if request.password != "" {
		return "", fmt.Errorf("password and password-file cannot be used together")
	}

	password, err := os.ReadFile(request.passwordFile)
	if err != nil {
		return "", fmt.Errorf("read password file: %w", err)
	}
	return strings.TrimSuffix(strings.TrimSuffix(string(password), "\n"), "\r"), nil
}

func writePrivateFile(path string, value string) error {
	directory := filepath.Dir(path)
	temporaryFile, err := os.CreateTemp(directory, ".digitalstrom-mqtt-")
	if err != nil {
		return err
	}
	temporaryPath := temporaryFile.Name()
	defer os.Remove(temporaryPath)

	if err := temporaryFile.Chmod(0o600); err != nil {
		temporaryFile.Close()
		return err
	}
	if _, err := temporaryFile.WriteString(value); err != nil {
		temporaryFile.Close()
		return err
	}
	if err := temporaryFile.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}

func modeStandard() error {
	config, err := config.ReadConfig()
	if err != nil {
		return monitor.Permanent(fmt.Errorf("invalid configuration: %w", err))
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
	exitSignal := make(chan os.Signal, 2)
	signal.Notify(exitSignal, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(exitSignal)
	started := make(chan error, 1)
	go func() { started <- ctrl.Start() }()
	startupTimeout := time.NewTimer(2 * time.Minute)
	defer startupTimeout.Stop()
	select {
	case err := <-started:
		if err != nil {
			return fmt.Errorf("controller startup failed: %w", err)
		}
	case err := <-ctrl.Monitor.Failures():
		return err
	case <-startupTimeout.C:
		return fmt.Errorf("controller startup timed out")
	case <-exitSignal:
		return nil
	}

	var watchdog *appwatchdog.Watchdog
	if os.Getenv("HOME_ASSISTANT_APP") == "true" {
		watchdog = appwatchdog.FromEnvironment()
	}
	arm := func() {
		if watchdog != nil {
			if err := watchdog.Arm(context.Background()); err != nil {
				log.Warn().Err(err).Msg("Watchdog activation pending; will retry")
			}
		}
	}
	arm()
	tick := time.NewTicker(15 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			arm()
		case err := <-ctrl.Monitor.Failures():
			return err
		case <-exitSignal:
			return ctrl.Stop()
		}
	}
}
