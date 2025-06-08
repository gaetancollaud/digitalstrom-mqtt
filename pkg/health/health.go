package health

import (
	"context"
	"errors"
	"fmt"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/config"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/mqtt"
	"github.com/rs/zerolog/log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	healthgo "github.com/hellofresh/health-go/v5"
)

type Health interface {
	Start() error
	Stop() error
}

type health struct {
	config     config.HealthCheckConfig
	mqttClient mqtt.Client
	health     *healthgo.Health

	server        *http.Server
	serverCtx     context.Context
	serverStopCtx context.CancelFunc
	shutdownCtx   context.Context
}

func NewHealth(config config.HealthCheckConfig, mqttClient mqtt.Client) Health {
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
				log.Info().Msg("MQTT client is connected")
				return nil
			}
			return errors.New("MQTT client is not connected")
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("Unable to register MQTT healthcheck")
		return nil
	}

	return &health{
		config:     config,
		mqttClient: mqttClient,
		health:     h,
	}
}

func (h *health) Start() error {
	listenAddr := fmt.Sprintf("0.0.0.0:%d", h.config.Port)
	h.server = &http.Server{Addr: listenAddr, Handler: h.service()}
	h.serverCtx, h.serverStopCtx = context.WithCancel(context.Background())
	h.shutdownCtx, _ = context.WithTimeout(h.serverCtx, 30*time.Second)
	go func() {
		log.Info().Msgf("Starting health check server on %s", listenAddr)
		err := h.server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("Unable to start health check server")
		}
	}()
	return nil
}

func (h *health) Stop() error {
	err := h.server.Shutdown(h.shutdownCtx)
	if err != nil {
		return err
	}
	h.serverStopCtx()
	log.Info().Msg("Health check server stopped")
	return nil
}

func (h *health) service() http.Handler {
	r := chi.NewRouter()
	r.Get("/health", h.health.HandlerFunc)
	r.Get("/health/ready", h.health.HandlerFunc)
	r.Get("/health/live", h.health.HandlerFunc)
	return r
}
