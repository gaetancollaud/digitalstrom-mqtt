package digitalstrom_mqtt

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gaetancollaud/digitalstrom-mqtt/config"
	"github.com/gaetancollaud/digitalstrom-mqtt/digitalstrom"
	"github.com/gaetancollaud/digitalstrom-mqtt/utils"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

const (
	Online            string = "online"
	Offline           string = "offline"
	DisconnectTimeout uint   = 1000 // 1 second
)

type DigitalstromMqtt struct {
	config       *config.ConfigMqtt
	ds_config    *config.ConfigDigitalstrom
	client       mqtt.Client
	digitalstrom *digitalstrom.Digitalstrom
}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.Debug().
		Str("topic", msg.Topic()).
		Bytes("payload", msg.Payload()).
		Msg("Message received")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Error().
		Err(err).
		Msg("MQTT connection lost")
	time.Sleep(5 * time.Second)
}

func New(config *config.ConfigMqtt, ds_config *config.ConfigDigitalstrom, digitalstrom *digitalstrom.Digitalstrom) *DigitalstromMqtt {
	inst := new(DigitalstromMqtt)
	inst.config = config
	inst.ds_config = ds_config
	u, err := uuid.NewRandom()
	clientPostfix := "-"
	if utils.CheckNoErrorAndPrint(err) {
		clientPostfix = "-" + u.String()
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf(config.MqttUrl))
	opts.SetClientID("digitalstrom-mqtt" + clientPostfix)
	if len(config.Username) > 0 {
		opts.SetUsername(config.Username)
	}
	if len(config.Password) > 0 {
		opts.SetPassword(config.Password)
	}
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = func(client mqtt.Client) {
		log.Info().Msg("MQTT Connected")
		inst.subscribeToAllDevicesCommands()
	}
	opts.OnConnectionLost = connectLostHandler
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Panic().
			Err(token.Error()).
			Str("url", config.MqttUrl).
			Msg("Unable to connect to the mqtt broken")
	}

	inst.client = client
	inst.digitalstrom = digitalstrom

	return inst
}

func (dm *DigitalstromMqtt) Start() {
	go dm.ListenSceneEvent(dm.digitalstrom.GetSceneEventsChannel())
	go dm.ListenForDeviceState(dm.digitalstrom.GetDeviceChangeChannel())
	go dm.ListenForCircuitValues(dm.digitalstrom.GetCircuitChangeChannel())

	// Notify that digitalstrom-mqtt is connected and online.
	dm.publishServerStatus(Online)
	if dm.config.HomeAssistantDiscoveryEnabled {
		dm.publishDiscoveryMessages()
	}
	dm.subscribeToAllDevicesCommands()
}

// Perform cleanup operations when Stopping DigitalstromMqtt.
func (dm *DigitalstromMqtt) Stop() {
	// Notify that difitalstrom-mqtt is not longer online.
	dm.publishServerStatus(Offline)
	// Gracefully close the connection to the MQTT server.
	log.Info().Msg("Stopping MQTT client.")
	dm.client.Disconnect(DisconnectTimeout)
	log.Info().Msg("Disconnected from MQTT server.")
}

func (dm *DigitalstromMqtt) ListenSceneEvent(changes chan digitalstrom.SceneEvent) {
	for event := range changes {
		dm.publishSceneEvent(event)
	}
}

func (dm *DigitalstromMqtt) ListenForDeviceState(changes chan digitalstrom.DeviceStateChanged) {
	for event := range changes {
		dm.publishDevice(event)
	}
}

func (dm *DigitalstromMqtt) ListenForCircuitValues(changes chan digitalstrom.CircuitValueChanged) {
	for event := range changes {
		dm.publishCircuit(event)
	}
}

// Publish the current binary status into the MQTT topic.
func (dm *DigitalstromMqtt) publishServerStatus(message string) {
	topic := dm.getStatusTopic()
	log.Info().Str("status", message).Str("topic", topic).Msg("Updating server status topic")
	dm.client.Publish(topic, 0, dm.config.Retain, message)
}

func (dm *DigitalstromMqtt) publishSceneEvent(sceneEvent digitalstrom.SceneEvent) {
	sceneNameOrId := sceneEvent.SceneName
	if len(sceneNameOrId) == 0 {
		// no name for the scene we take the id instead
		sceneNameOrId = strconv.Itoa(sceneEvent.SceneId)
	}

	topic := dm.getTopic("scenes", strconv.Itoa(sceneEvent.ZoneId), sceneEvent.ZoneName, sceneNameOrId, "event")

	json, err := json.Marshal(sceneEvent)
	if utils.CheckNoErrorAndPrint(err) {
		dm.client.Publish(topic, 0, dm.config.Retain, json)
	}
}

func (dm *DigitalstromMqtt) publishDevice(changed digitalstrom.DeviceStateChanged) {
	topic := dm.getTopic("devices", changed.Device.Dsid, changed.Device.Name, changed.Channel, "state")

	dm.client.Publish(topic, 0, dm.config.Retain, fmt.Sprintf("%.2f", changed.NewValue))
}

func (dm *DigitalstromMqtt) publishCircuit(changed digitalstrom.CircuitValueChanged) {
	//log.Info().Msg("Updating meter", changed.Circuit.Name, changed.ConsumptionW, changed.EnergyWs)

	if changed.ConsumptionW != -1 {
		topic := dm.getTopic("circuits", changed.Circuit.Dsid, changed.Circuit.Name, "consumptionW", "state")
		dm.client.Publish(topic, 0, dm.config.Retain, fmt.Sprintf("%d", changed.ConsumptionW))
	}

	if changed.EnergyWs != -1 {
		topic := dm.getTopic("circuits", changed.Circuit.Dsid, changed.Circuit.Name, "EnergyWs", "state")
		dm.client.Publish(topic, 0, dm.config.Retain, fmt.Sprintf("%d", changed.EnergyWs))
	}
}

func (dm *DigitalstromMqtt) deviceReceiverHandler(deviceName string, channel string, msg mqtt.Message) {
	payloadStr := string(msg.Payload())
	log.Info().
		Str("device", deviceName).
		Str("channel", channel).
		Str("payload", payloadStr).
		Msg("MQTT message received")
	if strings.ToLower(payloadStr) == "stop" {
		dm.digitalstrom.SetDeviceValue(digitalstrom.DeviceCommand{
			Action:     digitalstrom.Stop,
			DeviceName: deviceName,
			Channel:    channel,
		})
	} else {
		value, err := strconv.ParseFloat(payloadStr, 64)
		if utils.CheckNoErrorAndPrint(err) {
			dm.digitalstrom.SetDeviceValue(digitalstrom.DeviceCommand{
				Action:     digitalstrom.Set,
				DeviceName: deviceName,
				Channel:    channel,
				NewValue:   value,
			})
		} else {
			log.Error().Err(err).Str("payload", payloadStr).Msg("Unable to parse payload")
		}
	}
}

func (dm *DigitalstromMqtt) subscribeToAllDevicesCommands() {
	for _, device := range dm.digitalstrom.GetAllDevices() {
		for _, channel := range device.OutputChannels {
			deviceName := device.Name // deep copy
			deviceId := device.Dsid   // deep copy
			channelCopy := channel    // deep copy
			topic := dm.getTopic("devices", deviceId, deviceName, channelCopy, "command")
			log.Trace().Str("topic", topic).Str("deviceName", deviceName).Str("channel", channelCopy).Msg("Subscribing for topic")
			dm.client.Subscribe(topic, 0, func(client mqtt.Client, message mqtt.Message) {
				log.Debug().Str("topic", topic).Str("deviceName", deviceName).Str("channel", channelCopy).Msg("Message received")
				dm.deviceReceiverHandler(deviceName, channelCopy, message)
			})
		}
	}
}

func (dm *DigitalstromMqtt) getTopic(deviceType string, deviceId string, deviceName string, channel string, commandState string) string {
	topic := dm.config.TopicFormat

	if dm.config.NormalizeDeviceName {
		deviceName = normalizeForTopicName(deviceName)
	}

	topic = strings.ReplaceAll(topic, "{deviceType}", deviceType)
	topic = strings.ReplaceAll(topic, "{deviceId}", deviceId)
	topic = strings.ReplaceAll(topic, "{deviceName}", deviceName)
	topic = strings.ReplaceAll(topic, "{channel}", channel)
	topic = strings.ReplaceAll(topic, "{commandState}", commandState)

	return topic
}

// Returns MQTT topic to publish the Server status.
func (dm *DigitalstromMqtt) getStatusTopic() string {
	// FIXME: use a topic prefix for all digitalstrom-mqtt messages.
	root_topic := strings.Split(dm.config.TopicFormat, "/")[0]
	return root_topic + "/server/state"
}

// Publish the current binary status into the MQTT topic.
func (dm *DigitalstromMqtt) publishDiscoveryMessages() {
	for _, device := range dm.digitalstrom.GetAllDevices() {
		messages, err := dm.deviceToHomeAssistantDiscoveryMessage(device)
		if utils.CheckNoErrorAndPrint(err) {
			for _, discovery_message := range messages {
				dm.client.Publish(discovery_message.topic, 0, dm.config.Retain, discovery_message.message)
			}
		}
	}
	for _, circuit := range dm.digitalstrom.GetAllCircuits() {
		messages, err := dm.circuitToHomeAssistantDiscoveryMessage(circuit)
		if utils.CheckNoErrorAndPrint(err) {
			for _, discovery_message := range messages {
				dm.client.Publish(discovery_message.topic, 0, dm.config.Retain, discovery_message.message)
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

// Return the definition of the light and cover entities coming from a device.
func (dm *DigitalstromMqtt) deviceToHomeAssistantDiscoveryMessage(device digitalstrom.Device) ([]HassDiscoveryMessage, error) {
	// Check for device instances where the discovery message can not be created.
	if device.Name == "" {
		return nil, fmt.Errorf("empty device name, skipping discovery message")
	}
	if (device.DeviceType != digitalstrom.Light) && (device.DeviceType != digitalstrom.Blind) {
		return nil, fmt.Errorf("device type not supported %s", device.DeviceType)
	}
	device_config := map[string]interface{}{
		"configuration_url": "https://" + dm.ds_config.Host,
		"identifiers":       []interface{}{device.Dsid, device.Dsuid},
		"manufacturer":      "DigitalStrom",
		"model":             device.HwInfo,
		"name":              device.Name,
	}
	availability := []interface{}{
		map[string]interface{}{
			"topic":                 dm.getStatusTopic(),
			"payload_available":     Online,
			"payload_not_available": Offline,
		},
	}
	var message map[string]interface{}
	var topic string
	if device.DeviceType == digitalstrom.Light {
		// Setup configuration for a MQTT Cover in Home Assistant:
		// https://www.home-assistant.io/integrations/light.mqtt/
		topic = dm.config.HomeAssistantDiscoveryPrefix + "/light/" + device.Dsid + "/light/config"
		message = map[string]interface{}{
			"device": device_config,
			"name": utils.RemoveRegexp(
				device.Name,
				dm.config.HomeAssistantRemoveRegexpFromName),
			"unique_id":         device.Dsid + "_light",
			"retain":            dm.config.Retain,
			"availability":      availability,
			"availability_mode": "all",
			"command_topic": dm.getTopic(
				"devices",
				device.Dsid,
				device.Name,
				device.OutputChannels[0],
				"command"),
			"state_topic": dm.getTopic(
				"devices",
				device.Dsid,
				device.Name,
				device.OutputChannels[0],
				"state"),
			"payload_on":  "100.00",
			"payload_off": "0.00",
			"qos":         0,
		}
	} else if device.DeviceType == digitalstrom.Blind {
		// Setup configuration for a MQTT Cover in Home Assistant:
		// https://www.home-assistant.io/integrations/cover.mqtt/
		topic = dm.config.HomeAssistantDiscoveryPrefix + "/cover/" + device.Dsid + "/cover/config"
		message = map[string]interface{}{
			"device": device_config,
			"name": utils.RemoveRegexp(
				device.Name,
				dm.config.HomeAssistantRemoveRegexpFromName),
			"unique_id":         device.Dsid + "_cover",
			"device_class":      "blind",
			"retain":            dm.config.Retain,
			"availability":      availability,
			"availability_mode": "all",
			"state_topic": dm.getTopic(
				"devices",
				device.Dsid,
				device.Name,
				device.OutputChannels[0],
				"state"),
			"state_closed": "0.00",
			"state_open":   "100.00",
			"command_topic": dm.getTopic(
				"devices",
				device.Dsid,
				device.Name,
				device.OutputChannels[0],
				"command"),
			"payload_close": "0.00",
			"payload_open":  "100.00",
			"payload_stop":  "STOP",
			"position_topic": dm.getTopic(
				"devices",
				device.Dsid,
				device.Name,
				device.OutputChannels[0],
				"state"),
			"set_position_topic": dm.getTopic(
				"devices",
				device.Dsid,
				device.Name,
				device.OutputChannels[0],
				"command"),
			"position_template": "{{ value | int }}",
			"qos":               0,
		}
		// In case the cover supports tilting.
		if len(device.OutputChannels) > 1 {
			message["tilt_status_topic"] = dm.getTopic(
				"devices",
				device.Dsid,
				device.Name,
				device.OutputChannels[1],
				"state")
			message["tilt_command_topic"] = dm.getTopic(
				"devices",
				device.Dsid,
				device.Name,
				device.OutputChannels[1],
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
func (dm *DigitalstromMqtt) circuitToHomeAssistantDiscoveryMessage(circuit digitalstrom.Circuit) ([]HassDiscoveryMessage, error) {
	device_config := map[string]interface{}{
		"configuration_url": "https://" + dm.ds_config.Host,
		"identifiers":       []interface{}{circuit.Dsid},
		"manufacturer":      "DigitalStrom",
		"model":             circuit.HwName,
		"name":              circuit.Name,
	}
	availability := []interface{}{
		map[string]interface{}{
			"topic":                 dm.getStatusTopic(),
			"payload_available":     Online,
			"payload_not_available": Offline,
		},
	}
	// Setup configuration for a MQTT Cover in Home Assistant:
	// https://www.home-assistant.io/integrations/sensor.mqtt/
	// Define sensor for power consumption. This is a straightforward
	// definition.
	power_topic := dm.config.HomeAssistantDiscoveryPrefix + "/sensor/" + circuit.Dsid + "/power/config"
	power_message := map[string]interface{}{
		"device":            device_config,
		"name":              "Power " + circuit.Name,
		"unique_id":         circuit.Dsid + "_power",
		"retain":            dm.config.Retain,
		"availability":      availability,
		"availability_mode": "all",
		"state_topic": dm.getTopic(
			"circuits",
			circuit.Dsid,
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
	energy_topic := dm.config.HomeAssistantDiscoveryPrefix + "/sensor/" + circuit.Dsid + "/energy/config"
	energy_message := map[string]interface{}{
		"device":            device_config,
		"name":              "Energy " + circuit.Name,
		"unique_id":         circuit.Dsid + "_energy",
		"retain":            dm.config.Retain,
		"availability":      availability,
		"availability_mode": "all",
		"state_topic": dm.getTopic(
			"circuits",
			circuit.Dsid,
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
	power_json, err := json.Marshal(power_message)
	if err != nil {
		return nil, err
	}
	energy_json, err := json.Marshal(energy_message)
	if err != nil {
		return nil, err
	}
	return []HassDiscoveryMessage{
		{
			topic:   power_topic,
			message: power_json,
		},
		{
			topic:   energy_topic,
			message: energy_json,
		},
	}, nil
}

func normalizeForTopicName(item string) string {
	output := ""
	for i := 0; i < len(item); i++ {
		c := item[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' {
			output += string(c)
		} else if c == ' ' || c == '/' {
			output += "_"
		}
	}
	return output
}
