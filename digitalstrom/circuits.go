package digitalstrom

import (
	"github.com/gaetancollaud/digitalstrom-mqtt/digitalstrom/client"
	"github.com/gaetancollaud/digitalstrom-mqtt/utils"
	"github.com/rs/zerolog/log"
)

type CircuitValueChanged struct {
	Circuit      client.Circuit
	ConsumptionW int64
	EnergyWs     int64
}

type CircuitsManager struct {
	httpClient        client.DigitalStromClient
	circuits          []client.Circuit
	circuitValuesChan chan CircuitValueChanged
}

func NewCircuitManager(httpClient client.DigitalStromClient) *CircuitsManager {
	dm := new(CircuitsManager)
	dm.httpClient = httpClient
	dm.circuitValuesChan = make(chan CircuitValueChanged)

	return dm
}

func (dm *CircuitsManager) Start() {
	dm.reloadAllCircuits()
}

func (dm *CircuitsManager) reloadAllCircuits() {
	log.Info().Msg("Reloading circuits")
	response, err := dm.httpClient.ApartmentGetCircuits()
	if err != nil {
		log.Error().
			Err(err).
			Msg("Unable to load circuit list")
	} else {
		dm.circuits = response.Circuits
		log.Debug().
			Str("circuits", utils.PrettyPrint(dm.circuits)).
			Msg("Circuits loaded")
	}
}

func (dm *CircuitsManager) UpdateCircuitsValue() {
	for _, circuit := range dm.circuits {
		if circuit.HasMetering {
			consumptionW := int64(-1)
			energyWs := int64(-1)

			powerResponse, err := dm.httpClient.CircuitGetConsumption(circuit.DsId)
			if utils.CheckNoErrorAndPrint(err) {
				consumptionW = int64(powerResponse.Consumption)
			}

			energyResponse, err := dm.httpClient.CircuitGetEnergyMeterValue(circuit.DsId)
			if utils.CheckNoErrorAndPrint(err) {
				energyWs = int64(energyResponse.MeterValue)
			}

			dm.updateValue(circuit, consumptionW, energyWs)
		}
	}
}

func (dm *CircuitsManager) updateValue(circuit client.Circuit, newConsumptionW int64, newEnergyWs int64) {
	dm.circuitValuesChan <- CircuitValueChanged{
		Circuit:      circuit,
		ConsumptionW: newConsumptionW,
		EnergyWs:     newEnergyWs,
	}
}
