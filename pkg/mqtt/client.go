package mqtt

import (
	"fmt"
	"path"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

const QOS byte = 0

const (
	Online  string = "online"
	Offline string = "offline"
)

// Topics.
const (
	State        string = "state"
	Command      string = "command"
	Event        string = "event"
	serverStatus string = "server/status"
)

type SubscriptionHandler struct {
	Topic          string
	MessageHandler mqtt.MessageHandler
}

type Client interface {
	// Connect to the MQTT server.
	Connect() error
	// Disconnect from the MQTT server.
	Disconnect() error

	// Publishes a message under the prefix topic of DigitalStrom.
	Publish(topic string, message interface{}) error
	// Same as publish but force the retain flag regardless of what is in the config
	PublishAndRetain(topic string, message interface{}) error
	// Subscribe to a topic and calls the given handler when a message is
	// received.
	Subscribe(topic string, messageHandler mqtt.MessageHandler) error

	// Return the full topic for a given subpath.
	GetFullTopic(topic string) string
	// Returns the topic used to publish the server status.
	ServerStatusTopic() string

	RawClient() mqtt.Client
}

type client struct {
	mqttClient    mqtt.Client
	options       ClientOptions
	subscriptions *Subscriptions
}

type Subscriptions struct {
	shouldReconnect bool
	list            []SubscriptionHandler
}

func NewClient(options *ClientOptions) Client {
	subscriptions := Subscriptions{
		list: []SubscriptionHandler{},
	}
	mqttOptions := mqtt.NewClientOptions().
		AddBroker(options.MqttUrl).
		SetClientID("digitalstrom-mqtt-" + uuid.New().String()).
		SetOrderMatters(false).
		SetUsername(options.Username).
		SetPassword(options.Password).
		SetAutoReconnect(true).
		SetReconnectingHandler(func(client mqtt.Client, opts *mqtt.ClientOptions) {
			log.Info().Str("url", options.MqttUrl).Msg("Reconnecting to MQTT server.")
			subscriptions.shouldReconnect = true
		}).
		SetOnConnectHandler(func(client mqtt.Client) {
			log.Info().Str("url", options.MqttUrl).Msg("Connected to MQTT server.")

			if subscriptions.shouldReconnect {
				subscriptions.shouldReconnect = false
				log.Info().Int("count", len(subscriptions.list)).Msg("Re-subscribing to topics")
				for _, sub := range subscriptions.list {
					log.Debug().Str("topic", sub.Topic).Msg("Re-subscribing to topic")
					t := client.Subscribe(
						sub.Topic,
						QOS,
						sub.MessageHandler)
					<-t.Done()
					if t.Error() != nil {
						log.Error().Err(t.Error()).Str("topic", sub.Topic).Msg("Error re-subscribing to topic")
					}
				}
			}
		})

	return &client{
		mqttClient:    mqtt.NewClient(mqttOptions),
		options:       *options,
		subscriptions: &subscriptions,
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

func (c *client) publish(topic string, message interface{}, forceRetain bool) error {
	t := c.mqttClient.Publish(
		path.Join(c.options.TopicPrefix, topic),
		QOS,
		c.options.Retain || forceRetain,
		message)
	<-t.Done()
	return t.Error()
}

func (c *client) Publish(topic string, message interface{}) error {
	return c.publish(topic, message, false)
}

func (c *client) PublishAndRetain(topic string, message interface{}) error {
	return c.publish(topic, message, true)
}

func (c *client) Subscribe(topic string, messageHandler mqtt.MessageHandler) error {
	topic = path.Join(c.options.TopicPrefix, topic)
	c.subscriptions.list = append(c.subscriptions.list, SubscriptionHandler{
		Topic:          topic,
		MessageHandler: messageHandler,
	})
	log.Debug().Int("count", len(c.subscriptions.list)).Str("topic", topic).Msg("Subscribing to topic")
	t := c.mqttClient.Subscribe(
		topic,
		QOS,
		messageHandler)
	<-t.Done()
	return t.Error()
}

// Publish the current binary status into the MQTT topic.
func (c *client) publishServerStatus(message string) error {
	log.Info().Str("status", message).Str("topic", serverStatus).Msg("Updating server status topic")
	return c.PublishAndRetain(serverStatus, message)
}

func (c *client) ServerStatusTopic() string {
	return path.Join(c.options.TopicPrefix, serverStatus)
}

func (c *client) GetFullTopic(topic string) string {
	return path.Join(c.options.TopicPrefix, topic)
}

func (c *client) RawClient() mqtt.Client {
	return c.mqttClient
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
