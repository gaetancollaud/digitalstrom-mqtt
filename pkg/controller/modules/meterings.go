package modules

import (
	"fmt"
	"path"
	"time"

	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/config"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/digitalstrom"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/homeassistant"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/mqtt"
	"github.com/rs/zerolog/log"
)

const (
	meterings        string = "meterings"
	powerConsumption string = "consumptionW"
	energyMeter      string = "energyWh"
)

// Circuits Module encapsulates all the logic regarding the controllers. The logic
// is the following: every 30 seconds the controllers meter values are being checked and
// pushed to the corresponding topic in the MQTT server.
type MeteringsModule struct {
	mqttClient mqtt.Client
	dsClient   digitalstrom.Client
	dsRegistry digitalstrom.Registry

	ticker     *time.Ticker
	tickerDone chan struct{}
}

func (c *MeteringsModule) Start() error {
	c.ticker = time.NewTicker(10 * time.Second)
	c.tickerDone = make(chan struct{})

	go func() {
		for {
			select {
			case <-c.tickerDone:
				return
			case <-c.ticker.C:
				c.updateMeteringValues()
			}
		}
	}()
	return nil
}

func (c *MeteringsModule) Stop() error {
	c.ticker.Stop()
	c.tickerDone <- struct{}{}
	c.ticker = nil
	return nil
}

func (c *MeteringsModule) updateMeteringValues() {
	log.Debug().Msg("Updating metering values.")

	meterings, err := c.dsRegistry.GetMeterings()
	if err != nil {
		log.Panic().Err(err).Msg("Error fetching the meterings in the apartment.")
	}

	meteringStatus, err := c.dsClient.GetMeteringStatus()
	if err != nil {
		log.Error().Err(err).Msg("Error fetching metering status")
		return
	}

	meteringStatusLookup := make(map[string]digitalstrom.MeteringValue)
	for _, value := range meteringStatus.Values {
		meteringStatusLookup[value.Id] = value
	}

	for _, metering := range meterings {

		var itemName string

		if metering.Attributes.Origin.Type == digitalstrom.MeteringTypeController {
			controller, err := c.dsRegistry.GetControllerById(metering.Attributes.Origin.MeteringOriginId)
			if err != nil {
				log.Error().
					Err(err).
					Str("controllerId", metering.Attributes.Origin.MeteringOriginId).
					Str("meteringId", metering.MeteringId).
					Msg("No controller found for metering ")
				continue
			}
			itemName = controller.Attributes.Name
		} else {
			// apartment
			itemName = "apartment"
		}

		meteringValue := meteringStatusLookup[metering.MeteringId]

		var measurement = ""
		if metering.Attributes.Unit == "W" {
			measurement = powerConsumption
		} else if metering.Attributes.Unit == "Wh" {
			measurement = energyMeter
		} else {
			log.Warn().Str("unit", metering.Attributes.Unit).Msg("Unknown unit")
		}

		valueStr := fmt.Sprintf("%.0f", meteringValue.Attributes.Value)
		if err := c.mqttClient.Publish(meteringTopic(itemName, measurement), valueStr); err != nil {
			log.Error().
				Err(err).
				Str("itemName", itemName).
				Str("unit", metering.Attributes.Unit).
				Msg("Error updating metering")
			continue
		}
	}
}

func meteringTopic(itemName string, measurement string) string {
	return path.Join(meterings, itemName, measurement, mqtt.State)
}

func (c *MeteringsModule) GetHomeAssistantEntities() ([]homeassistant.DiscoveryConfig, error) {
	configs := []homeassistant.DiscoveryConfig{}

	controllers, err := c.dsRegistry.GetControllers()
	if err != nil {
		return nil, err
	}

	// manually add apartment
	controllers = append(controllers, digitalstrom.Controller{
		ControllerId: "apartment",
		Attributes: digitalstrom.ControllerAttributes{
			Name:     "apartment",
			TechName: "apartment",
		},
	})

	for _, controller := range controllers {
		powerConfig := homeassistant.DiscoveryConfig{
			Domain:   homeassistant.Sensor,
			DeviceId: controller.ControllerId,
			ObjectId: "power",
			Config: &homeassistant.SensorConfig{
				BaseConfig: homeassistant.BaseConfig{
					Device: homeassistant.Device{
						Identifiers: []string{controller.ControllerId},
						Model:       controller.Attributes.TechName,
						Name:        controller.Attributes.Name,
					},
					Name:     "Power " + controller.Attributes.Name,
					UniqueId: controller.ControllerId + "_power",
				},
				StateTopic: c.mqttClient.GetFullTopic(
					meteringTopic(controller.Attributes.Name, powerConsumption)),
				UnitOfMeasurement: "W",
				DeviceClass:       "power",
				Icon:              "mdi:flash",
			},
		}
		configs = append(configs, powerConfig)
		energyConfig := homeassistant.DiscoveryConfig{
			Domain:   homeassistant.Sensor,
			DeviceId: controller.ControllerId,
			ObjectId: "energy",
			Config: &homeassistant.SensorConfig{
				BaseConfig: homeassistant.BaseConfig{
					Device: homeassistant.Device{
						Identifiers: []string{controller.ControllerId},
						Model:       controller.Attributes.TechName,
						Name:        controller.Attributes.Name,
					},
					Name:     "Energy " + controller.Attributes.Name,
					UniqueId: controller.ControllerId + "_energy",
				},
				StateTopic: c.mqttClient.GetFullTopic(
					meteringTopic(controller.Attributes.Name, energyMeter)),
				UnitOfMeasurement: "kWh",
				DeviceClass:       "energy",
				StateClass:        "total_increasing",
				ValueTemplate:     "{{ (value | float / (3600*1000)) | round(3) }}",
				Icon:              "mdi:lightning-bolt",
			},
		}
		configs = append(configs, energyConfig)
	}
	return configs, nil
}

func NewMeteringsModule(mqttClient mqtt.Client, dsClient digitalstrom.Client, dsRegistry digitalstrom.Registry, config *config.Config) Module {
	return &MeteringsModule{
		mqttClient: mqttClient,
		dsClient:   dsClient,
		dsRegistry: dsRegistry,
	}
}

func init() {
	Register("meterings", NewMeteringsModule)
}
