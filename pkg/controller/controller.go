package controller

import (
	"fmt"

	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/config"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/controller/modules"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/digitalstrom"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/mqtt"
	"github.com/rs/zerolog/log"
)

type Controller struct {
	dsClient   digitalstrom.Client
	mqttClient mqtt.Client

	modules map[string]modules.Module
}

func NewController(config *config.Config) *Controller {
	// Create Digitalstrom client.
	dsOptions := digitalstrom.NewClientOptions().
		SetHost(config.Digitalstrom.Host).
		SetPort(config.Digitalstrom.Port).
		SetUsername(config.Digitalstrom.Username).
		SetPassword(config.Digitalstrom.Password)
	dsClient := digitalstrom.NewClient(dsOptions)

	mqttOptions := mqtt.NewClientOptions().
		SetMqttUrl(config.Mqtt.MqttUrl).
		SetUsername(config.Mqtt.Username).
		SetPassword(config.Mqtt.Password).
		SetTopicPrefix(config.Mqtt.TopicPrefix).
		SetRetain(config.Mqtt.Retain)
	mqttClient := mqtt.NewClient(mqttOptions)
	controller := Controller{
		dsClient:   dsClient,
		mqttClient: mqttClient,
		modules:    map[string]modules.Module{},
	}

	for name, builder := range modules.Modules {
		module := builder(mqttClient, dsClient, config)
		controller.modules[name] = module
	}

	return &controller
}

func (c *Controller) Start() error {
	log.Info().Msg("Starting controller.")
	if err := c.mqttClient.Connect(); err != nil {
		return fmt.Errorf("error connecting to MQTT client: %w", err)
	}
	if err := c.dsClient.Connect(); err != nil {
		return fmt.Errorf("error connecting to DigitalStrom client: %w", err)
	}

	for name, module := range c.modules {
		log.Info().Str("module", name).Msg("Starting module.")
		if err := module.Start(); err != nil {
			return fmt.Errorf("error starting module '%s': %w", name, err)
		}
	}

	return nil
}

func (c *Controller) Stop() error {
	log.Info().Msg("Stopping controller.")

	for name, module := range c.modules {
		log.Info().Str("module", name).Msg("Stopping module.")
		if err := module.Stop(); err != nil {
			return fmt.Errorf("error stopping module '%s': %w", name, err)
		}
	}

	if err := c.mqttClient.Disconnect(); err != nil {
		return fmt.Errorf("error disconnecting to MQTT client: %w", err)
	}
	if err := c.dsClient.Disconnect(); err != nil {
		return fmt.Errorf("error disconnecting to DigitalStrom client: %w", err)
	}

	return nil
}
