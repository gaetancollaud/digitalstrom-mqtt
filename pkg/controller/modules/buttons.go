package modules

import (
	"fmt"
	"path"

	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/config"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/digitalstrom"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/homeassistant"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/mqtt"
	"github.com/rs/zerolog/log"
)

const (
	buttons string = "buttons"
)

// Circuit Module encapsulates all the logic regarding the circuits. The logic
// is the following: every 30 seconds the circuit values are being checked and
// pushed to the corresponding topic in the MQTT server.
type ButtonModule struct {
	mqttClient mqtt.Client
	dsClient   digitalstrom.Client

	normalizeTopicName bool

	buttons      []digitalstrom.Device
	buttonLookup map[string]digitalstrom.Device
}

func (c *ButtonModule) Start() error {
	// Prefetch the list of circuits available in DigitalStrom.
	response, err := c.dsClient.ApartmentGetDevices()
	if err != nil {
		log.Panic().Err(err).Msg("Error fetching the circuits in the apartment.")
	}
	// Store only the devices that are actually joker buttons.
	for _, device := range *response {
		if len(device.Groups) == 1 && device.Groups[0] == 8 { // Joker device.
			c.buttons = append(c.buttons, device)
		}
	}

	// Create maps regarding Buttons for fast lookup when a new Event is
	// received.
	for _, device := range c.buttons {
		c.buttonLookup[device.Dsuid] = device
	}

	// Subscribe to DigitalStrom events.
	if err := c.dsClient.EventSubscribe(digitalstrom.EventButtonClick, func(client digitalstrom.Client, event digitalstrom.Event) error {
		return c.onDsEvent(event)
	}); err != nil {
		return err
	}
	return nil
}

func (c *ButtonModule) Stop() error {
	if err := c.dsClient.EventUnsubscribe(digitalstrom.EventButtonClick); err != nil {
		return err
	}
	return nil
}

func (c *ButtonModule) onDsEvent(event digitalstrom.Event) error {
	fmt.Printf("button click event: %+v\n", event)
	button := c.buttonLookup[event.Source.Dsid]
	fmt.Printf("button device: %+v\n", button)

	return c.publishButtonClick(&button, event.Properties.ClickType)
	// Click type:
	// 0 simple click
	// 1 double click
	// 6 long click
}

func (c *ButtonModule) publishButtonClick(device *digitalstrom.Device, clickType int) error {
	message := getClickType(clickType)
	return c.mqttClient.Publish(c.buttonClickTopic(device.Name), message)
}

func (c *ButtonModule) buttonClickTopic(deviceName string) string {
	if c.normalizeTopicName {
		deviceName = normalizeForTopicName(deviceName)
	}
	return path.Join(buttons, deviceName, mqtt.Event)
}

var clickTypeMapping = map[int]string{
	0: "1-push",
	1: "2-push",
	2: "3-push",
	3: "4-push",
	6: "long-push",
}

func getClickType(clickType int) string {
	value, ok := clickTypeMapping[clickType]
	if !ok {
		return "unknown"
	}
	return value
}

func (c *ButtonModule) GetHomeAssistantEntities() ([]homeassistant.DiscoveryConfig, error) {
	configs := []homeassistant.DiscoveryConfig{}

	hassTypeMapping := map[int]string{
		0: "button_short_press",
		1: "button_double_press",
		2: "button_triple_press",
		3: "button_quadruple_press",
		6: "button_long_press",
	}

	for _, device := range c.buttons {
		for clickType, clickName := range clickTypeMapping {
			config := homeassistant.DiscoveryConfig{
				Domain:   homeassistant.DeviceTrigger,
				DeviceId: device.Dsid,
				ObjectId: clickName,
				Config: &homeassistant.DeviceTriggerConfig{
					BaseConfig: homeassistant.BaseConfig{
						Device: homeassistant.Device{
							Identifiers: []string{
								device.Dsid,
							},
							Name: device.Name,
						},
					},
					AutomationType: "trigger",
					Type:           hassTypeMapping[clickType],
					Subtype:        clickName,
					Topic:          c.mqttClient.GetFullTopic(c.buttonClickTopic(device.Name)),
					Payload:        getClickType(clickType),
				},
			}
			configs = append(configs, config)
		}
	}
	return configs, nil
}

func NewButtonModule(mqttClient mqtt.Client, dsClient digitalstrom.Client, config *config.Config) Module {
	return &ButtonModule{
		mqttClient:         mqttClient,
		dsClient:           dsClient,
		normalizeTopicName: config.Mqtt.NormalizeDeviceName,
		buttons:            []digitalstrom.Device{},
		buttonLookup:       map[string]digitalstrom.Device{},
	}
}

func init() {
	Register("buttons", NewButtonModule)
}
