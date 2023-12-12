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
	circuits         string = "circuits"
	powerConsumption string = "consumptionW"
	energyMeter      string = "EnergyWs"
)

// Circuits Module encapsulates all the logic regarding the controllers. The logic
// is the following: every 30 seconds the controllers meter values are being checked and
// pushed to the corresponding topic in the MQTT server.
type CircuitsModule struct {
	mqttClient mqtt.Client
	dsClient   digitalstrom.Client
	dsRegistry digitalstrom.Registry

	ticker     *time.Ticker
	tickerDone chan struct{}
}

func (c *CircuitsModule) Start() error {
	c.ticker = time.NewTicker(30 * time.Second)
	c.tickerDone = make(chan struct{})

	go func() {
		for {
			select {
			case <-c.tickerDone:
				return
			case <-c.ticker.C:
				c.updateCircuitValues()
			}
		}
	}()
	return nil
}

func (c *CircuitsModule) Stop() error {
	c.ticker.Stop()
	c.tickerDone <- struct{}{}
	c.ticker = nil
	return nil
}

func (c *CircuitsModule) updateCircuitValues() {
	log.Info().Msg("Updating metering values.")

	// Prefetch the list of circuits available in DigitalStrom.
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
		controller, err := c.dsRegistry.GetControllerById(metering.Attributes.Origin.MeteringOriginId)
		if err != nil {
			// This is expected sometimes, for example for the "apartment"
			log.Trace().
				Err(err).
				Str("controllerId", metering.Attributes.Origin.MeteringOriginId).
				Str("meteringId", metering.MeteringId).
				Msg("No controller found for metering ")
			continue
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
		if err := c.mqttClient.Publish(circuitTopic(controller.Attributes.Name, measurement), valueStr); err != nil {
			log.Error().
				Err(err).
				Str("controller", controller.Attributes.Name).
				Str("unit", metering.Attributes.Unit).
				Msg("Error updating metering of circuit")
			continue
		}
	}
}

func circuitTopic(circuitName string, measurement string) string {
	return path.Join(circuits, circuitName, measurement, mqtt.State)
}

func (c *CircuitsModule) GetHomeAssistantEntities() ([]homeassistant.DiscoveryConfig, error) {
	configs := []homeassistant.DiscoveryConfig{}

	controllers, err := c.dsRegistry.GetControllers()
	if err != nil {
		return nil, err
	}

	for _, circuit := range controllers {
		powerConfig := homeassistant.DiscoveryConfig{
			Domain:   homeassistant.Sensor,
			DeviceId: circuit.ControllerId,
			ObjectId: "power",
			Config: &homeassistant.SensorConfig{
				BaseConfig: homeassistant.BaseConfig{
					Device: homeassistant.Device{
						Identifiers: []string{circuit.ControllerId},
						Model:       circuit.Attributes.TechName,
						Name:        circuit.Attributes.Name,
					},
					Name:     "Power " + circuit.Attributes.Name,
					UniqueId: circuit.ControllerId + "_power",
				},
				StateTopic: c.mqttClient.GetFullTopic(
					circuitTopic(circuit.Attributes.Name, powerConsumption)),
				UnitOfMeasurement: "W",
				DeviceClass:       "power",
				Icon:              "mdi:flash",
			},
		}
		configs = append(configs, powerConfig)
		energyConfig := homeassistant.DiscoveryConfig{
			Domain:   homeassistant.Sensor,
			DeviceId: circuit.ControllerId,
			ObjectId: "energy",
			Config: &homeassistant.SensorConfig{
				BaseConfig: homeassistant.BaseConfig{
					Device: homeassistant.Device{
						Identifiers: []string{circuit.ControllerId},
						Model:       circuit.Attributes.TechName,
						Name:        circuit.Attributes.Name,
					},
					Name:     "Energy " + circuit.Attributes.Name,
					UniqueId: circuit.ControllerId + "_energy",
				},
				StateTopic: c.mqttClient.GetFullTopic(
					circuitTopic(circuit.Attributes.Name, energyMeter)),
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
	return &CircuitsModule{
		mqttClient: mqttClient,
		dsClient:   dsClient,
		dsRegistry: dsRegistry,
	}
}

func init() {
	Register("meterings", NewMeteringsModule)
}
