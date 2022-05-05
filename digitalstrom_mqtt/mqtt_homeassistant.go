package digitalstrom_mqtt

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gaetancollaud/digitalstrom-mqtt/digitalstrom/client"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/config"
	"github.com/gaetancollaud/digitalstrom-mqtt/utils"
)

// Define Home Assistant components.
type Component string

const (
	Light  Component = "light"
	Cover  Component = "cover"
	Sensor Component = "sensor"
)

type HomeAssistantMqtt struct {
	mqtt   *DigitalstromMqtt
	config *config.ConfigHomeAssistant
}

func (hass *HomeAssistantMqtt) Start() {
	if !hass.config.DiscoveryEnabled {
		return
	}
	hass.publishDiscoveryMessages()
}

// Publish the current binary status into the MQTT topic.
func (hass *HomeAssistantMqtt) publishDiscoveryMessages() {
	for _, device := range hass.mqtt.digitalstrom.GetAllDevices() {
		messages, err := hass.deviceToHomeAssistantDiscoveryMessage(device)
		if utils.CheckNoErrorAndPrint(err) {
			for _, discovery_message := range messages {
				hass.mqtt.client.Publish(discovery_message.topic, 0, hass.config.Retain, discovery_message.message)
			}
		}
	}
	for _, circuit := range hass.mqtt.digitalstrom.GetAllCircuits() {
		messages, err := hass.circuitToHomeAssistantDiscoveryMessage(circuit)
		if utils.CheckNoErrorAndPrint(err) {
			for _, discovery_message := range messages {
				hass.mqtt.client.Publish(discovery_message.topic, 0, hass.config.Retain, discovery_message.message)
			}
		}
	}
}

// Define a Home Assistant discovery message as its MQTT topic and the message
// to be published.
type HassDiscoveryMessage struct {
	topic   string
	message []byte
}

// Generates the discovery topic where to publish the message following the
// Home Assistant convention.
func (hass *HomeAssistantMqtt) discoveryTopic(component Component, deviceId string, objectId string) string {
	return hass.config.DiscoveryTopicPrefix + "/" + string(component) + "/" + deviceId + "/" + objectId + "/config"
}

// Return the definition of the light and cover entities coming from a device.
func (hass *HomeAssistantMqtt) deviceToHomeAssistantDiscoveryMessage(device client.Device) ([]HassDiscoveryMessage, error) {
	// Check for device instances where the discovery message can not be created.
	if device.Name == "" {
		return nil, fmt.Errorf("empty device name, skipping discovery message")
	}
	if (device.DeviceType() != client.Light) && (device.DeviceType() != client.Blind) {
		return nil, fmt.Errorf("device type not supported %s", device.DeviceType())
	}
	deviceConfig := map[string]interface{}{
		"configuration_url": "https://" + hass.config.DigitalStromHost,
		"identifiers":       []interface{}{device.Dsid, device.Dsuid},
		"manufacturer":      "DigitalStrom",
		"model":             device.HwInfo,
		"name":              device.Name,
	}
	availability := []interface{}{
		map[string]interface{}{
			"topic":                 hass.mqtt.getStatusTopic(),
			"payload_available":     Online,
			"payload_not_available": Offline,
		},
	}
	var message map[string]interface{}
	var topic string
	if device.DeviceType() == client.Light {
		// Setup configuration for a MQTT Cover in Home Assistant:
		// https://www.home-assistant.io/integrations/light.mqtt/
		nodeId := "light"
		topic = hass.discoveryTopic(Light, device.Dsid, nodeId)
		message = map[string]interface{}{
			"device": deviceConfig,
			"name": utils.RemoveRegexp(
				device.Name,
				hass.config.RemoveRegexpFromName),
			"unique_id":         device.Dsid + "_" + nodeId,
			"retain":            hass.config.Retain,
			"availability":      availability,
			"availability_mode": "all",
			"command_topic": hass.mqtt.getTopic(
				"devices",
				device.Dsid,
				device.Name,
				device.OutputChannels[0].Name,
				"command"),
			"state_topic": hass.mqtt.getTopic(
				"devices",
				device.Dsid,
				device.Name,
				device.OutputChannels[0].Name,
				"state"),
			"payload_on":  "100.00",
			"payload_off": "0.00",
			"qos":         0,
		}
		if device.Properties().Dimmable {
			message["on_command_type"] = "brightness"
			message["brightness_scale"] = 100
			message["brightness_state_topic"] = hass.mqtt.getTopic(
				"devices",
				device.Dsid,
				device.Name,
				device.OutputChannels[0].Name,
				"state")
			message["brightness_command_topic"] = hass.mqtt.getTopic(
				"devices",
				device.Dsid,
				device.Name,
				device.OutputChannels[0].Name,
				"command")

		}
	} else if device.DeviceType() == client.Blind {
		// Setup configuration for a MQTT Cover in Home Assistant:
		// https://www.home-assistant.io/integrations/cover.mqtt/

		// Covers expose up to two output channels:
		// * to control the position, and
		// * to control the tilt
		// For that reason let's extract the correspondent channel for each
		// action.
		positionChannel := ""
		tiltChannel := ""
		for _, channel := range device.OutputChannels {
			channelName := channel.Name
			if strings.Contains(channelName, "Angle") {
				tiltChannel = channelName
			}
			if strings.Contains(channelName, "Position") {
				positionChannel = channelName
			}
		}
		// If position channel has not been found raise error.
		if positionChannel == "" {
			return nil, fmt.Errorf("position channel could not be found for device " + device.Name)
		}

		nodeId := "cover"
		topic = hass.discoveryTopic(Cover, device.Dsid, nodeId)
		message = map[string]interface{}{
			"device": deviceConfig,
			"name": utils.RemoveRegexp(
				device.Name,
				hass.config.RemoveRegexpFromName),
			"unique_id":         device.Dsid + "_" + nodeId,
			"device_class":      "blind",
			"retain":            hass.config.Retain,
			"availability":      availability,
			"availability_mode": "all",
			"state_topic": hass.mqtt.getTopic(
				"devices",
				device.Dsid,
				device.Name,
				positionChannel,
				"state"),
			"state_closed": "0.00",
			"state_open":   "100.00",
			"command_topic": hass.mqtt.getTopic(
				"devices",
				device.Dsid,
				device.Name,
				positionChannel,
				"command"),
			"payload_close": "0.00",
			"payload_open":  "100.00",
			"payload_stop":  "STOP",
			"position_topic": hass.mqtt.getTopic(
				"devices",
				device.Dsid,
				device.Name,
				positionChannel,
				"state"),
			"set_position_topic": hass.mqtt.getTopic(
				"devices",
				device.Dsid,
				device.Name,
				positionChannel,
				"command"),
			"position_template": "{{ value | int }}",
			"qos":               0,
		}
		// In case the cover supports tilting.
		if tiltChannel != "" {
			message["tilt_status_topic"] = hass.mqtt.getTopic(
				"devices",
				device.Dsid,
				device.Name,
				tiltChannel,
				"state")
			message["tilt_command_topic"] = hass.mqtt.getTopic(
				"devices",
				device.Dsid,
				device.Name,
				tiltChannel,
				"command")
			message["tilt_status_template"] = "{{ value | int }}"
		}
	} else {
		return nil, fmt.Errorf("device type is not supported to be announce to Home Assistant discovery")
	}
	json, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}
	return []HassDiscoveryMessage{
		{
			topic:   topic,
			message: json,
		},
	}, nil
}

// Return the definition of the power and energy sensors coming from the circuit
// devices.
func (hass *HomeAssistantMqtt) circuitToHomeAssistantDiscoveryMessage(circuit client.Circuit) ([]HassDiscoveryMessage, error) {
	deviceConfig := map[string]interface{}{
		"configuration_url": "https://" + hass.config.DigitalStromHost,
		"identifiers":       []interface{}{circuit.DsId},
		"manufacturer":      "DigitalStrom",
		"model":             circuit.HwName,
		"name":              circuit.Name,
	}
	availability := []interface{}{
		map[string]interface{}{
			"topic":                 hass.mqtt.getStatusTopic(),
			"payload_available":     Online,
			"payload_not_available": Offline,
		},
	}
	// Setup configuration for a MQTT Cover in Home Assistant:
	// https://www.home-assistant.io/integrations/sensor.mqtt/
	// Define sensor for power consumption. This is a straightforward
	// definition.
	powerNodeId := "power"
	powerTopic := hass.discoveryTopic(Sensor, circuit.DsId, powerNodeId)
	powerMessage := map[string]interface{}{
		"device":            deviceConfig,
		"name":              "Power " + circuit.Name,
		"unique_id":         circuit.DsId + "_" + powerNodeId,
		"retain":            hass.config.Retain,
		"availability":      availability,
		"availability_mode": "all",
		"state_topic": hass.mqtt.getTopic(
			"circuits",
			circuit.DsId,
			circuit.Name,
			"consumptionW",
			"state"),
		"unit_of_measurement": "W",
		"device_class":        "power",
		"icon":                "mdi:flash",
		"qos":                 0,
	}
	// Define the energy sensor. We need to define the state class in order to
	// make sure statistics are bing computed and stored. We also use the
	// `value_template` field to make the conversion from Ws reported in the
	// MQTT topic, to kWh which is the default unit of measurement of energy in
	// Home Assistant.
	energyNodeId := "power"
	energyTopic := hass.discoveryTopic(Sensor, circuit.DsId, energyNodeId)
	energyMessage := map[string]interface{}{
		"device":            deviceConfig,
		"name":              "Energy " + circuit.Name,
		"unique_id":         circuit.DsId + "_" + energyNodeId,
		"retain":            hass.config.Retain,
		"availability":      availability,
		"availability_mode": "all",
		"state_topic": hass.mqtt.getTopic(
			"circuits",
			circuit.DsId,
			circuit.Name,
			"EnergyWs",
			"state"),
		"unit_of_measurement": "kWh",
		"device_class":        "energy",
		"state_class":         "total_increasing",
		// Convert the vaue from Ws to kWh which is the default energy unit in
		// Home Assistant
		"value_template": "{{ (value | float / (3600*1000)) | round(3) }}",
		"icon":           "mdi:lightning-bolt",
		"qos":            0,
	}
	powerJson, err := json.Marshal(powerMessage)
	if err != nil {
		return nil, err
	}
	enegryJson, err := json.Marshal(energyMessage)
	if err != nil {
		return nil, err
	}
	return []HassDiscoveryMessage{
		{
			topic:   powerTopic,
			message: powerJson,
		},
		{
			topic:   energyTopic,
			message: enegryJson,
		},
	}, nil
}
