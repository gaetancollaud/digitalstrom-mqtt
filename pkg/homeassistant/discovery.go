package homeassistant

import (
	"encoding/json"
	"fmt"
	"path"

	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/config"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/mqtt"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/utils"
)

type Domain string

const (
	Sensor           Domain = "sensor"
	Light            Domain = "light"
	DeviceAutomation Domain = "device_automation"
	Cover            Domain = "cover"
	Scene            Domain = "scene"
	DeviceTrigger    Domain = "device_automation"
)

type DiscoveryConfig struct {
	Domain   Domain
	DeviceId string
	ObjectId string
	Config   MqttConfig
}

type HomeAssistantDiscoveryInterface interface {
	// Returns the list of Home Assitant MQTT entities that each module would
	// be exporting for discovery.
	// This will be run after the method Start is called and therefore it can
	// assume that the logic there will be run.
	GetHomeAssistantEntities() ([]DiscoveryConfig, error)
}

type HomeAssistantDiscovery struct {
	mqttClient mqtt.Client
	config     *config.ConfigHomeAssistant

	discoveryConfigs []DiscoveryConfig
}

func NewHomeAssistantDiscovery(mqttClient mqtt.Client, config *config.ConfigHomeAssistant) *HomeAssistantDiscovery {
	return &HomeAssistantDiscovery{
		mqttClient:       mqttClient,
		config:           config,
		discoveryConfigs: []DiscoveryConfig{},
	}
}

func (hass *HomeAssistantDiscovery) AddConfigs(configs []DiscoveryConfig) {
	systemAvailability := Availability{
		Topic:               hass.mqttClient.ServerStatusTopic(),
		PayloadAvailable:    mqtt.Online,
		PayloadNotAvailable: mqtt.Offline,
	}
	for _, config := range configs {
		entityName := config.Config.GetName()
		config.Config.
			SetName(
				utils.RemoveRegexp(
					entityName,
					hass.config.RemoveRegexpFromName)).
			SetRetain(hass.config.Retain).
			AddAvailability(systemAvailability).
			SetAvailabilityMode("all")
		// Update the config with some generic attributes for all
		// configurations.
		device := config.Config.GetDevice()
		device.Manufacturer = "DigitalStrom"
		device.ConfigurationUrl = "https://" + hass.config.DigitalStromHost

		hass.discoveryConfigs = append(hass.discoveryConfigs, config)
	}
}

func (hass *HomeAssistantDiscovery) PublishDiscoveryMessages() error {
	if !hass.config.DiscoveryEnabled {
		return nil
	}

	for _, config := range hass.discoveryConfigs {
		topic := path.Join(
			hass.config.DiscoveryTopicPrefix,
			string(config.Domain),
			config.DeviceId,
			config.ObjectId,
			"config")
		json, err := json.Marshal(config.Config)
		if err != nil {
			return fmt.Errorf("error serializing dicovery config to JSON: %w", err)
		}
		t := hass.mqttClient.RawClient().Publish(topic, 0, true, json)
		<-t.Done()
		if t.Error() != nil {
			return fmt.Errorf("error publishing discovery message to MQTT: %w", err)
		}
	}
	return nil
}
