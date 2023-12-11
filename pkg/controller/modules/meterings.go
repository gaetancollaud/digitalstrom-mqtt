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

// Metering Module encapsulates all the logic regarding the meters. The logic
// is the following: every 30 seconds the meters values are being checked and
// pushed to the corresponding topic in the MQTT server.
type MeteringsModule struct {
	mqttClient mqtt.Client
	dsClient   digitalstrom.Client
	dsRegistry digitalstrom.Registry

	meterings  []digitalstrom.Metering
	ticker     *time.Ticker
	tickerDone chan struct{}
}

func (c *MeteringsModule) Start() error {
	// Prefetch the list of circuits available in DigitalStrom.
	meterings, err := c.dsRegistry.GetMeterings()
	if err != nil {
		log.Panic().Err(err).Msg("Error fetching the meterings in the apartment.")
	}
	c.meterings = meterings

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

func (c *MeteringsModule) Stop() error {
	c.ticker.Stop()
	c.tickerDone <- struct{}{}
	c.ticker = nil
	return nil
}

func (c *MeteringsModule) updateCircuitValues() {
	log.Info().Msg("Updating metering values.")
	meteringStatus, err := c.dsClient.GetMeteringStatus()
	if err != nil {
		log.Error().Err(err).Msg("Error fetching metering status")
		return
	}

	meteringStatusLookup := make(map[string]digitalstrom.MeteringValue)
	for _, value := range meteringStatus.Values {
		meteringStatusLookup[value.Id] = value
	}

	for _, metering := range c.meterings {

		meteringValue := meteringStatusLookup[metering.MeteringId]

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

func (c *MeteringsModule) GetHomeAssistantEntities() ([]homeassistant.DiscoveryConfig, error) {
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

func NewMeteringsModule(mqttClient mqtt.Client, dsClient digitalstrom.Client, dsRegistry digitalstrom.Registry, config *config.Config) Module {
	return &MeteringsModule{
		mqttClient: mqttClient,
		dsClient:   dsClient,
		dsRegistry: dsRegistry,
		circuits:   []digitalstrom.Circuit{},
	}
}

func init() {
	Register("meterings", NewMeteringsModule)
}
