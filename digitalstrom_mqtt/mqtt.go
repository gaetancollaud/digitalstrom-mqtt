package digitalstrom_mqtt

import (
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gaetancollaud/digitalstrom-mqtt/config"
	"github.com/gaetancollaud/digitalstrom-mqtt/digitalstrom"
	"github.com/gaetancollaud/digitalstrom-mqtt/utils"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"strconv"
	"strings"
	"time"
)

type DigitalstromMqtt struct {
	config *config.ConfigMqtt
	client mqtt.Client

	digitalstrom *digitalstrom.Digitalstrom
}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.Debug().
		Str("topic", msg.Topic()).
		Bytes("payload", msg.Payload()).
		Msg("Message received")
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	log.Info().Msg("MQTT Connected")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Error().
		Err(err).
		Msg("MQTT connection lost")
	time.Sleep(5 * time.Second)
}

func New(config *config.ConfigMqtt, digitalstrom *digitalstrom.Digitalstrom) *DigitalstromMqtt {
	inst := new(DigitalstromMqtt)
	inst.config = config
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
	opts.OnConnect = connectHandler
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
	go dm.ListenForDeviceState(dm.digitalstrom.GetDeviceChangeChannel())
	go dm.ListenForCircuitValues(dm.digitalstrom.GetCircuitChangeChannel())

	dm.subscribeToAllDevicesCommands()
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
	value, err := strconv.ParseFloat(payloadStr, 64)
	if utils.CheckNoErrorAndPrint(err) {
		log.Info().Msg("MQTT message to set device '" + deviceName + "' and channel '" + channel + " to '" + payloadStr + "'")
		dm.digitalstrom.SetDeviceValue(digitalstrom.DeviceCommand{
			DeviceName: deviceName,
			Channel:    channel,
			NewValue:   value,
		})
	} else {
		log.Error().Err(err).Str("payload", payloadStr).Msg("Unable to parse payload")
	}
}

func (dm *DigitalstromMqtt) subscribeToAllDevicesCommands() {
	for _, device := range dm.digitalstrom.GetAllDevices() {
		for _, channel := range device.Channels {
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
