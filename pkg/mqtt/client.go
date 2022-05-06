package mqtt

import (
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/digitalstrom"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

const (
	Online  string = "online"
	Offline string = "offline"
)

// Topics.
const (
	circuits         string = "circuits"
	devices          string = "devices"
	state            string = "state"
	command          string = "command"
	powerConsumption string = "consumptionW"
	energyMeter      string = "EnergyWs"
)

type Client interface {
	// Connect to the MQTT server.
	Connect() error
	// Disconnect from the MQTT server.
	Disconnect() error

	PublishDevice(device digitalstrom.Device, channelValues []digitalstrom.ChannelValue) error

	PublishCircuit(circuit digitalstrom.Circuit, powerConsumption int64, energyMeter int64) error

	PublishScene(scene digitalstrom.SceneName) error
}

type client struct {
	mqttClient mqtt.Client
	options    ClientOptions
}

func NewClient(options *ClientOptions) Client {
	mqttOptions := mqtt.NewClientOptions().
		AddBroker(options.MqttUrl).
		SetClientID("digitalstrom-mqtt-" + uuid.New().String()).
		SetOrderMatters(false).
		SetUsername(options.Username).
		SetPassword(options.Password).
		SetDefaultPublishHandler(options.MessageHandler).
		SetOnConnectHandler(options.OnConnectHandler)

	return &client{
		mqttClient: mqtt.NewClient(mqttOptions),
		options:    *options,
	}
}

func (c *client) Connect() error {
	t := c.mqttClient.Connect()
	<-t.Done()
	if t.Error() != nil {
		return fmt.Errorf("error connecting to MQTT broker: %w", t.Error())
	}

	if err := c.publishServerStatus(Online); err != nil {
		return err
	}
	return nil
}

func (c *client) Disconnect() error {
	log.Info().Msg("Publishing Offline status to MQTT server.")
	if err := c.publishServerStatus(Offline); err != nil {
		return err
	}
	c.mqttClient.Disconnect(uint(c.options.DisconnectTimeout.Milliseconds()))
	log.Info().Msg("Disconnected from MQTT server.")
	return nil
}

func (c *client) PublishDevice(device digitalstrom.Device, channelValues []digitalstrom.ChannelValue) error {
	for _, channelValue := range channelValues {
		topic := c.getTopic(devices, device.Name, channelValue.Name, state)
		if err := c.publish(topic, fmt.Sprintf("%.2f", channelValue.Value)); err != nil {
			return err
		}

	}
	return nil
}

func (c *client) PublishCircuit(circuit digitalstrom.Circuit, powerConsumptionValue int64, energyMeterValue int64) error {
	var topic string
	topic = c.getTopic(circuits, circuit.Name, powerConsumption, state)
	if err := c.publish(topic, fmt.Sprintf("%d", powerConsumptionValue)); err != nil {
		return err
	}
	topic = c.getTopic(circuits, circuit.Name, energyMeter, state)
	if err := c.publish(topic, fmt.Sprintf("%d", energyMeterValue)); err != nil {
		return err
	}
	return nil
}

func (c *client) PublishScene(scene digitalstrom.SceneName) error {
	return nil
}

func (c *client) publish(topic string, message interface{}) error {
	t := c.mqttClient.Publish(topic, c.options.QoS, c.options.Retain, message)
	<-t.Done()
	return t.Error()
}

// Publish the current binary status into the MQTT topic.
func (c *client) publishServerStatus(message string) error {
	topic := c.getStatusTopic()
	log.Info().Str("status", message).Str("topic", topic).Msg("Updating server status topic")
	return c.publish(topic, message)
}

func (c *client) getTopic(deviceType string, deviceName string, channel string, commandState string) string {
	if c.options.NormalizeDeviceName {
		deviceName = normalizeForTopicName(deviceName)
	}

	topic := c.options.TopicPrefix
	topic += "/" + deviceType
	topic += "/" + deviceName
	topic += "/" + channel
	topic += "/" + commandState
	return topic
}

// Returns MQTT topic to publish the Server status.
func (c *client) getStatusTopic() string {
	return c.options.TopicPrefix + "/server/state"
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
