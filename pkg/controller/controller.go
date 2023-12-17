package controller

import (
	"fmt"

	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/config"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/controller/modules"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/digitalstrom"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/homeassistant"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/mqtt"
	"github.com/rs/zerolog/log"
)

type Controller struct {
	dsClient      digitalstrom.Client
	dsRegistry    digitalstrom.Registry
	mqttClient    mqtt.Client
	hassDiscovery *homeassistant.HomeAssistantDiscovery

	modules map[string]modules.Module
}

func NewController(config *config.Config) *Controller {
	// Create Digitalstrom client.
	dsOptions := digitalstrom.NewClientOptions().
		SetHost(config.Digitalstrom.Host).
		SetPort(config.Digitalstrom.Port).
		SetApiKey(config.Digitalstrom.ApiKey)
	dsClient := digitalstrom.NewClient(dsOptions)

	dsRegistry := digitalstrom.NewRegistry(dsClient)

	mqttOptions := mqtt.NewClientOptions().
		SetMqttUrl(config.Mqtt.MqttUrl).
		SetUsername(config.Mqtt.Username).
		SetPassword(config.Mqtt.Password).
		SetTopicPrefix(config.Mqtt.TopicPrefix).
		SetRetain(config.Mqtt.Retain)
	mqttClient := mqtt.NewClient(mqttOptions)

	hass := homeassistant.NewHomeAssistantDiscovery(
		mqttClient,
		&config.HomeAssistant)

	controller := Controller{
		dsClient:      dsClient,
		dsRegistry:    dsRegistry,
		mqttClient:    mqttClient,
		hassDiscovery: hass,
		modules:       map[string]modules.Module{},
	}

	for name, builder := range modules.Modules {
		module := builder(mqttClient, dsClient, dsRegistry, config)
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
	if err := c.dsRegistry.Start(); err != nil {
		return fmt.Errorf("error starting DigitalStrom registry: %w", err)
	}

	for name, module := range c.modules {
		log.Info().Str("module", name).Msg("Starting module.")
		if err := module.Start(); err != nil {
			return fmt.Errorf("error starting module '%s': %w", name, err)
		}
	}

	// Retrieve from all the modules the discovery configs to be exported.
	for name, module := range c.modules {
		m, ok := module.(homeassistant.HomeAssistantDiscoveryInterface)
		if !ok {
			continue
		}
		configs, err := m.GetHomeAssistantEntities()
		if err != nil {
			return fmt.Errorf("error getting discovery configs from module '%s': %w", name, err)
		}
		c.hassDiscovery.AddConfigs(configs)
	}
	// Publishes Home Assistant Discovery messages.
	if err := c.hassDiscovery.PublishDiscoveryMessages(); err != nil {
		return err
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
	if err := c.dsRegistry.Stop(); err != nil {
		return fmt.Errorf("error stoping DigitalStrom registry: %w", err)
	}
	if err := c.dsClient.Disconnect(); err != nil {
		return fmt.Errorf("error disconnecting to DigitalStrom client: %w", err)
	}

	return nil
}
