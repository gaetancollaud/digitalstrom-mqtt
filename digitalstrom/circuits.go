package digitalstrom

import (
	"github.com/gaetancollaud/digitalstrom-mqtt/utils"
	"github.com/rs/zerolog/log"
	"strings"
)

type CircuitValueChanged struct {
	Circuit      Circuit
	ConsumptionW int64
	EnergyWs     int64
}

type Circuit struct {
	Name         string
	Dsid         string
	HwName       string
	HasMetering  bool
	IsValid      bool
	consumptionW int64
	energyWs     int64
}

type CircuitsManager struct {
	httpClient        *HttpClient
	circuits          []Circuit
	circuitValuesChan chan CircuitValueChanged
}

func NewCircuitManager(httpClient *HttpClient) *CircuitsManager {
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
	response, err := dm.httpClient.get("json/apartment/getCircuits")
	if err != nil {
		log.Error().
			Err(err).
			Msg("Unable to load circuit list")
	} else {
		circuits := response.mapValue["circuits"].([]interface{})
		for _, s := range circuits {
			m := s.(map[string]interface{})
			if dm.supportedCircuit(m) {
				dm.circuits = append(dm.circuits, Circuit{
					Dsid:         m["dsid"].(string),
					Name:         m["name"].(string),
					HwName:       m["hwName"].(string),
					HasMetering:  m["hasMetering"].(bool),
					IsValid:      m["isValid"].(bool),
					consumptionW: -1,
					energyWs:     -1,
				})
			}
		}

		log.Debug().
			Str("circuits", utils.PrettyPrintArray(dm.circuits)).
			Msg("Circuits loaded")
	}
}

func (dm *CircuitsManager) supportedCircuit(m map[string]interface{}) bool {
	name := ""
	if m["name"] != nil {
		name = m["name"].(string)
	}
	if len(strings.TrimSpace(name)) == 0 {
		log.Warn().Msg("Circuit not supported because it has no name. Enable debug to see the complete devices")
		log.Debug().Str("device", utils.PrettyPrintMap(m)).Msg("Circuit not supported because it has no name")
		return false
	}
	if m["dsid"] == nil || len(m["dsid"].(string)) == 0 {
		log.Info().Str("name", name).Msg("Circuit not supported because it has no dsid. Enable debug to see the complete devices")
		log.Debug().Str("device", utils.PrettyPrintMap(m)).Msg("Circuit not supported because it has no dsid")
		return false
	}
	if !m["isValid"].(bool) {
		log.Info().Str("name", name).Msg("Circuit is not valid. Enable debug to see the complete devices")
		log.Debug().Str("device", utils.PrettyPrintMap(m)).Msg("Circuit is not valid")
		return false
	}
	return true
}

func (dm *CircuitsManager) UpdateCircuitsValue() {
	for _, circuit := range dm.circuits {
		if circuit.HasMetering {
			consumptionW := int64(-1)
			energyWs := int64(-1)

			response, err := dm.httpClient.get("json/circuit/getConsumption?id=" + circuit.Dsid)
			if utils.CheckNoErrorAndPrint(err) {
				consumptionW = int64(response.mapValue["consumption"].(float64))
			}

			response, err = dm.httpClient.get("json/circuit/getEnergyMeterValue?id=" + circuit.Dsid)
			if utils.CheckNoErrorAndPrint(err) {
				energyWs = int64(response.mapValue["meterValue"].(float64))
			}

			dm.updateValue(circuit, consumptionW, energyWs)
		}
	}
}

func (dm *CircuitsManager) updateValue(circuit Circuit, newConsumptionW int64, newEnergyWs int64) {
	dm.circuitValuesChan <- CircuitValueChanged{
		Circuit:      circuit,
		ConsumptionW: newConsumptionW,
		EnergyWs:     newEnergyWs,
	}
	circuit.consumptionW = newConsumptionW
	circuit.energyWs = newEnergyWs
}
