package digitalstrom_mqtt

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gaetancollaud/digitalstrom-mqtt/digitalstrom"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/config"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/utils"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

const (
	Online            string = "online"
	Offline           string = "offline"
	DisconnectTimeout uint   = 1000 // 1 second
)

type DigitalstromMqtt struct {
	config         *config.ConfigMqtt
	client         mqtt.Client
	digitalstrom   *digitalstrom.Digitalstrom
	home_assistant *HomeAssistantMqtt
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

func New(config *config.Config, digitalstrom *digitalstrom.Digitalstrom) *DigitalstromMqtt {
	inst := new(DigitalstromMqtt)
	inst.config = &config.Mqtt
	u, err := uuid.NewRandom()
	clientPostfix := "-"
	if utils.CheckNoErrorAndPrint(err) {
		clientPostfix = "-" + u.String()
	}
	inst.home_assistant = &HomeAssistantMqtt{
		mqtt:   inst,
		config: &config.HomeAssistant,
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf(config.Mqtt.MqttUrl))
	opts.SetClientID("digitalstrom-mqtt" + clientPostfix)
	// Set the recommended value as we don't care about the order of the
	// messages received from the broker.
	opts.SetOrderMatters(false)
	if len(config.Mqtt.Username) > 0 {
		opts.SetUsername(config.Mqtt.Username)
	}
	if len(config.Mqtt.Password) > 0 {
		opts.SetPassword(config.Mqtt.Password)
	}
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		log.Info().Msg("MQTT Connected")
		inst.subscribeToAllDevicesCommands()
	})
	opts.SetConnectionLostHandler(connectLostHandler)
	client := mqtt.NewClient(opts)

	inst.client = client
	inst.digitalstrom = digitalstrom

	return inst
}

func (dm *DigitalstromMqtt) Start() {
	if token := dm.client.Connect(); token.Wait() && token.Error() != nil {
		log.Panic().
			Err(token.Error()).
			Str("url", dm.config.MqttUrl).
			Msg("Unable to connect to the mqtt broken")
	}

	go dm.ListenSceneEvent(dm.digitalstrom.GetSceneEventsChannel())
	go dm.ListenForDeviceState(dm.digitalstrom.GetDeviceChangeChannel())
	go dm.ListenForCircuitValues(dm.digitalstrom.GetCircuitChangeChannel())

	// Notify that digitalstrom-mqtt is connected and online.
	dm.publishServerStatus(Online)
	dm.home_assistant.Start()
	dm.subscribeToAllDevicesCommands()
}

// Perform cleanup operations when Stopping DigitalstromMqtt.
func (dm *DigitalstromMqtt) Stop() {
	// Notify that difitalstrom-mqtt is not longer online.
	dm.publishServerStatus(Offline)
	// Gracefully close the connection to the MQTT server.
	log.Info().Msg("Stopping MQTT digitalstrom.")
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
		topic := dm.getTopic("circuits", changed.Circuit.DsId, changed.Circuit.Name, "consumptionW", "state")
		dm.client.Publish(topic, 0, dm.config.Retain, fmt.Sprintf("%d", changed.ConsumptionW))
	}

	if changed.EnergyWs != -1 {
		topic := dm.getTopic("circuits", changed.Circuit.DsId, changed.Circuit.Name, "EnergyWs", "state")
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
			Action:     digitalstrom.CommandStop,
			DeviceName: deviceName,
			Channel:    channel,
		})
	} else {
		value, err := strconv.ParseFloat(payloadStr, 64)
		if utils.CheckNoErrorAndPrint(err) {
			dm.digitalstrom.SetDeviceValue(digitalstrom.DeviceCommand{
				Action:     digitalstrom.CommandSet,
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
			deviceName := device.Name   // deep copy
			deviceId := device.Dsid     // deep copy
			channelCopy := channel.Name // deep copy
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
	topic := dm.config.TopicPrefix

	if dm.config.NormalizeDeviceName {
		deviceName = normalizeForTopicName(deviceName)
	}

	topic += "/" + deviceType
	topic += "/" + deviceName
	topic += "/" + channel
	topic += "/" + commandState

	return topic
}

// Returns MQTT topic to publish the Server status.
func (dm *DigitalstromMqtt) getStatusTopic() string {
	return dm.config.TopicPrefix + "/server/state"
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
