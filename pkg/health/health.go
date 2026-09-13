package health

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/config"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/monitor"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/mqtt"
	"github.com/go-chi/chi/v5"
	healthgo "github.com/hellofresh/health-go/v5"
	"github.com/rs/zerolog/log"
)

type Health interface {
	Start() error
	Stop() error
}

type health struct {
	progress   interface{ Check() error }
	config     config.HealthCheckConfig
	mqttClient mqtt.Client
	health     *healthgo.Health

	server *http.Server
}

func NewHealth(config config.HealthCheckConfig, mqttClient mqtt.Client, progress ...*monitor.Monitor) Health {
	h, _ := healthgo.New(healthgo.WithComponent(healthgo.Component{
		Name:    "digitalstrom-mqtt",
		Version: "v1.0",
	}),
	)

	// and then add some more if needed
	err := h.Register(healthgo.Config{
		Name:      "mqtt",
		Timeout:   time.Second * 2,
		SkipOnErr: false,
		Check: func(ctx context.Context) error {
			if mqttClient.RawClient().IsConnectionOpen() {
				log.Trace().Msg("MQTT client is connected")
				return nil
			}
			return errors.New("MQTT client is not connected")
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("Unable to register MQTT healthcheck")
		return nil
	}

	result := &health{
		config:     config,
		mqttClient: mqttClient,
		health:     h,
	}
	if len(progress) > 0 {
		result.progress = progress[0]
	}
	return result
}

func (h *health) Start() error {
	listenAddr := fmt.Sprintf("0.0.0.0:%d", h.config.Port)
	h.server = &http.Server{Addr: listenAddr, Handler: h.service()}
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}
	go func() {
		log.Info().Msgf("Starting health check server on %s", listenAddr)
		err := h.server.Serve(listener)
		if err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("Unable to start health check server")
		}
	}()
	return nil
}

func (h *health) Stop() error {
	err := shutdownHTTPServer(h.server, 30*time.Second)
	if err != nil {
		return err
	}
	log.Info().Msg("Health check server stopped")
	return nil
}

func shutdownHTTPServer(server *http.Server, timeout time.Duration) error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}

func (h *health) service() http.Handler {
	r := chi.NewRouter()
	r.Get("/health", h.health.HandlerFunc)
	r.Get("/health/started", h.health.HandlerFunc)
	r.Get("/health/ready", h.health.HandlerFunc)
	r.Get("/health/live", h.liveHandler)
	return r
}

func (h *health) liveHandler(writer http.ResponseWriter, _ *http.Request) {
	if h.progress != nil {
		if err := h.progress.Check(); err != nil {
			http.Error(writer, err.Error(), http.StatusServiceUnavailable)
			return
		}
	}
	writer.WriteHeader(http.StatusOK)
}
