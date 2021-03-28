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

//const BASE_TOPIC = "digitalstrom/"
//const BASE_DEVICE_TOPIC = BASE_TOPIC + "devices/"
//const BASE_CIRCUIT_TOPIC = BASE_TOPIC + "circuits/"
//const COMMAND_SUFFIX = "command"

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
		panic(token.Error())
	}

	inst.client = client
	inst.digitalstrom = digitalstrom

	return inst
}

func (dm *DigitalstromMqtt) Start() {
	go dm.ListenForDeviceStatus(dm.digitalstrom.GetDeviceChangeChannel())
	go dm.ListenForCircuitValues(dm.digitalstrom.GetCircuitChangeChannel())

	dm.subscribeToAllDevicesCommands()
}

func (dm *DigitalstromMqtt) ListenForDeviceStatus(changes chan digitalstrom.DeviceStatusChanged) {
	for event := range changes {
		dm.publishDevice(event)
	}
}

func (dm *DigitalstromMqtt) ListenForCircuitValues(changes chan digitalstrom.CircuitValueChanged) {
	for event := range changes {
		dm.publishCircuit(event)
	}
}

func (dm *DigitalstromMqtt) publishDevice(changed digitalstrom.DeviceStatusChanged) {
	topic := getTopic(dm.config.TopicFormat, "devices", changed.Device.Name, changed.Channel, "status")

	dm.client.Publish(topic, 0, false, fmt.Sprintf("%.2f", changed.NewValue))
}

func (dm *DigitalstromMqtt) publishCircuit(changed digitalstrom.CircuitValueChanged) {
	//log.Info().Msg("Updating meter", changed.Circuit.Name, changed.ConsumptionW, changed.EnergyWs)

	if changed.ConsumptionW != -1 {
		topic := getTopic(dm.config.TopicFormat, "circuits", changed.Circuit.Name, "consumptionW", "status")
		dm.client.Publish(topic, 0, false, fmt.Sprintf("%d", changed.ConsumptionW))
	}

	if changed.EnergyWs != -1 {
		topic := getTopic(dm.config.TopicFormat, "circuits", changed.Circuit.Name, "EnergyWs", "status")
		dm.client.Publish(topic, 0, false, fmt.Sprintf("%d", changed.EnergyWs))
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
			topic := getTopic(dm.config.TopicFormat, "devices", deviceName, channel, "command")
			log.Trace().Str("topic", topic).Str("deviceName", deviceName).Str("channel", channel).Msg("Subscribing for topic")
			dm.client.Subscribe(topic, 0, func(client mqtt.Client, message mqtt.Message) {
				log.Debug().Str("topic", topic).Str("deviceName", deviceName).Str("channel", channel).Msg("Message received")
				dm.deviceReceiverHandler(deviceName, channel, message)
			})
		}

	}
}

func getTopic(format string, deviceType string, deviceName string, channel string, commandStatus string) string {
	topic := format
	topic = strings.ReplaceAll(topic, "{deviceType}", deviceType)
	topic = strings.ReplaceAll(topic, "{deviceName}", deviceName)
	topic = strings.ReplaceAll(topic, "{channel}", channel)
	topic = strings.ReplaceAll(topic, "{commandStatus}", commandStatus)

	return topic
}
