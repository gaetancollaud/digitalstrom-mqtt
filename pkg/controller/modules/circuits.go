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

// Circuit Module encapsulates all the logic regarding the circuits. The logic
// is the following: every 30 seconds the circuit values are being checked and
// pushed to the corresponding topic in the MQTT server.
type CircuitModule struct {
	mqttClient mqtt.Client
	dsClient   digitalstrom.Client

	circuits   []digitalstrom.Circuit
	ticker     *time.Ticker
	tickerDone chan struct{}
}

func (c *CircuitModule) Start() error {
	// Prefetch the list of circuits available in DigitalStrom.
	response, err := c.dsClient.ApartmentGetCircuits()
	if err != nil {
		log.Panic().Err(err).Msg("Error fetching the circuits in the apartment.")
	}
	c.circuits = response.Circuits

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

func (c *CircuitModule) Stop() error {
	c.ticker.Stop()
	c.tickerDone <- struct{}{}
	c.ticker = nil
	return nil
}

func (c *CircuitModule) updateCircuitValues() {
	log.Info().Msg("Updating circuit values.")
	for _, circuit := range c.circuits {
		if !circuit.HasMetering {
			continue
		}

		powerResponse, err := c.dsClient.CircuitGetConsumption(circuit.DsId)
		if err != nil {
			log.Error().Err(err).Msgf("Error fetching power consumption of circuit '%s'", circuit.Name)
			continue
		}
		consumptionW := int64(powerResponse.Consumption)
		if err := c.mqttClient.Publish(circuitTopic(circuit.Name, powerConsumption), fmt.Sprintf("%d", consumptionW)); err != nil {
			log.Error().Err(err).Msgf("Error updating power consumption of circuit '%s'", circuit.Name)
			continue
		}

		energyResponse, err := c.dsClient.CircuitGetEnergyMeterValue(circuit.DsId)
		if err != nil {
			log.Error().Err(err).Msgf("Error fetching energy meter of circuit '%s'", circuit.Name)
			continue
		}
		energyWs := int64(energyResponse.MeterValue)
		if err := c.mqttClient.Publish(circuitTopic(circuit.Name, energyMeter), fmt.Sprintf("%d", energyWs)); err != nil {
			log.Error().Err(err).Msgf("Error updating energy meter of circuit '%s'", circuit.Name)
			continue
		}
	}
}

func circuitTopic(circuitName string, measurement string) string {
	return path.Join(circuits, circuitName, measurement, mqtt.State)
}

func (c *CircuitModule) GetHomeAssistantEntities() ([]homeassistant.DiscoveryConfig, error) {
	configs := []homeassistant.DiscoveryConfig{}

	for _, circuit := range c.circuits {
		powerConfig := homeassistant.DiscoveryConfig{
			Domain:   homeassistant.Sensor,
			DeviceId: circuit.DsId,
			ObjectId: "power",
			Config: &homeassistant.SensorConfig{
				BaseConfig: homeassistant.BaseConfig{
					Device: homeassistant.Device{
						Identifiers: []string{circuit.DsId},
						Model:       circuit.HwName,
						Name:        circuit.Name,
					},
					Name:     "Power " + circuit.Name,
					UniqueId: circuit.DsId + "_power",
				},
				StateTopic: c.mqttClient.GetFullTopic(
					circuitTopic(circuit.Name, powerConsumption)),
				UnitOfMeasurement: "W",
				DeviceClass:       "power",
				Icon:              "mdi:flash",
			},
		}
		configs = append(configs, powerConfig)
		energyConfig := homeassistant.DiscoveryConfig{
			Domain:   homeassistant.Sensor,
			DeviceId: circuit.DsId,
			ObjectId: "energy",
			Config: &homeassistant.SensorConfig{
				BaseConfig: homeassistant.BaseConfig{
					Device: homeassistant.Device{
						Identifiers: []string{circuit.DsId},
						Model:       circuit.HwName,
						Name:        circuit.Name,
					},
					Name:     "Energy " + circuit.Name,
					UniqueId: circuit.DsId + "_energy",
				},
				StateTopic: c.mqttClient.GetFullTopic(
					circuitTopic(circuit.Name, energyMeter)),
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

func NewCircuitModule(mqttClient mqtt.Client, dsClient digitalstrom.Client, config *config.Config) Module {
	return &CircuitModule{
		mqttClient: mqttClient,
		dsClient:   dsClient,
		circuits:   []digitalstrom.Circuit{},
	}
}

func init() {
	Register("circuits", NewCircuitModule)
}
