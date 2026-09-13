package mqtt

import (
	"errors"
	"fmt"
	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/monitor"
	"path"
	"time"

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
		SetProtocolVersion(4).
		SetConnectTimeout(30*time.Second).
		SetClientID("digitalstrom-mqtt-"+uuid.New().String()).
		SetOrderMatters(false).
		SetUsername(options.Username).
		SetPassword(options.Password).
		SetAutoReconnect(true).
		SetConnectionNotificationHandler(func(c mqtt.Client, notification mqtt.ConnectionNotification) {
			if failed, ok := notification.(mqtt.ConnectionNotificationFailed); ok {
				err := connectionError(failed.Reason)
				if monitor.IsPermanent(err) && options.Monitor != nil {
					options.Monitor.Report(err)
					go c.Disconnect(0)
				}
			}
		}).
		SetWill(serverStatus, Offline, QOS, true).
		SetReconnectingHandler(func(client mqtt.Client, opts *mqtt.ClientOptions) {
			log.Info().Str("url", options.MqttUrl).Msg("Reconnecting to MQTT server.")
			subscriptions.shouldReconnect = true
		}).
		SetOnConnectHandler(func(client mqtt.Client) {
			defer options.Monitor.Begin("MQTT reconnect subscriptions")()
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
					if err := waitToken(t); err != nil {
						log.Error().Err(err).Str("topic", sub.Topic).Msg("Error re-subscribing to topic")
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
	defer c.options.Monitor.Begin("MQTT connect")()
	t := c.mqttClient.Connect()
	if err := waitToken(t); err != nil {
		return fmt.Errorf("error connecting to MQTT broker: %w", connectionError(err))
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
	defer c.options.Monitor.Begin("MQTT publish")()
	t := c.mqttClient.Publish(
		path.Join(c.options.TopicPrefix, topic),
		QOS,
		c.options.Retain || forceRetain,
		message)
	return waitToken(t)
}

func (c *client) Publish(topic string, message interface{}) error {
	return c.publish(topic, message, false)
}

func (c *client) PublishAndRetain(topic string, message interface{}) error {
	return c.publish(topic, message, true)
}

func (c *client) Subscribe(topic string, messageHandler mqtt.MessageHandler) error {
	defer c.options.Monitor.Begin("MQTT subscribe")()
	handler := messageHandler
	messageHandler = func(client mqtt.Client, message mqtt.Message) {
		defer c.options.Monitor.Begin("MQTT command callback")()
		handler(client, message)
	}
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
	return waitToken(t)
}

func waitToken(token mqtt.Token) error {
	if !token.WaitTimeout(30 * time.Second) {
		return errors.New("MQTT operation timed out")
	}
	return token.Error()
}

func connectionError(err error) error {
	for _, permanent := range []error{packets.ErrorRefusedBadUsernameOrPassword,
		packets.ErrorRefusedNotAuthorised, packets.ErrorRefusedBadProtocolVersion, packets.ErrorRefusedIDRejected} {
		if errors.Is(err, permanent) {
			return monitor.Permanent(err)
		}
	}
	return err
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
