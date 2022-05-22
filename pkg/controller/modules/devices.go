package modules

import (
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	mqtt_base "github.com/eclipse/paho.mqtt.golang"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/config"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/digitalstrom"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/homeassistant"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/mqtt"
	"github.com/rs/zerolog/log"
)

const (
	devices string = "devices"
	stop    string = "stop"
)

// Circuit Module encapsulates all the logic regarding the circuits. The logic
// is the following: every 30 seconds the circuit values are being checked and
// pushed to the corresponding topic in the MQTT server.
type DeviceModule struct {
	mqttClient mqtt.Client
	dsClient   digitalstrom.Client

	normalizeDeviceName  bool
	refreshAtStart       bool
	invertBlindsPosition bool

	devices         []digitalstrom.Device
	deviceLookup    map[string]digitalstrom.Device
	zoneGroupLookup map[int]map[int][]string
}

func (c *DeviceModule) Start() error {
	// Prefetch the list of circuits available in DigitalStrom.
	response, err := c.dsClient.ApartmentGetDevices()
	if err != nil {
		log.Panic().Err(err).Msg("Error fetching the circuits in the apartment.")
	}
	c.devices = *response

	// Create maps regarding Devices for fast lookup when a new Event is
	// received.
	for _, device := range c.devices {
		c.deviceLookup[device.Dsid] = device
		_, ok := c.zoneGroupLookup[device.ZoneId]
		if !ok {
			c.zoneGroupLookup[device.ZoneId] = map[int][]string{}
		}

		for _, groupId := range device.Groups {
			_, ok := c.zoneGroupLookup[device.ZoneId][groupId]
			if !ok {
				c.zoneGroupLookup[device.ZoneId][groupId] = []string{}
			}
			c.zoneGroupLookup[device.ZoneId][groupId] = append(c.zoneGroupLookup[device.ZoneId][groupId], device.Dsid)
		}
	}

	// Refresh devices values.
	if c.refreshAtStart {
		go func() {
			for _, device := range c.devices {
				if err := c.updateDevice(&device); err != nil {
					log.Error().Err(err).Msgf("Error updating device '%s'", device.Name)
				}
			}
		}()
	}

	// Subscribe to DigitalStrom events.
	if err := c.dsClient.EventSubscribe(digitalstrom.EventCallScene, func(client digitalstrom.Client, event digitalstrom.Event) error {
		return c.onDsEvent(event)
	}); err != nil {
		return err
	}

	// Subscribe to MQTT events.
	for _, device := range c.devices {
		for _, channel := range device.OutputChannels {
			deviceCopy := device
			deviceName := deviceCopy.Name
			channelName := channel.Name
			topic := c.deviceCommandTopic(deviceName, channelName)
			log.Trace().
				Str("topic", topic).
				Str("deviceName", deviceName).
				Str("channel", channelName).
				Msg("Subscribing for topic.")
			c.mqttClient.Subscribe(topic, func(client mqtt_base.Client, message mqtt_base.Message) {
				payload := string(message.Payload())
				log.Trace().
					Str("topic", topic).
					Str("deviceName", deviceName).
					Str("channel", channelName).
					Str("payload", payload).
					Msg("Message Received.")
				if err := c.onMqttMessage(&deviceCopy, channelName, payload); err != nil {
					log.Error().
						Str("topic", topic).
						Err(err).
						Msg("Error handling MQTT Message.")
				}
			})
		}
	}
	return nil
}

func (c *DeviceModule) Stop() error {
	if err := c.dsClient.EventUnsubscribe(digitalstrom.EventCallScene); err != nil {
		return err
	}
	return nil
}

func (c *DeviceModule) onMqttMessage(device *digitalstrom.Device, channel string, message string) error {
	// In case stop is being passed as part of the message.
	if strings.ToLower(message) == stop {
		if err := c.dsClient.ZoneCallAction(device.ZoneId, digitalstrom.Stop); err != nil {
			return err
		}
		return nil
	}
	// Alternatively, the actual value is given and must be pushed to
	// DigitalStrom.
	value, err := strconv.ParseFloat(message, 64)
	if err != nil {
		return fmt.Errorf("error parsing message as float value: %w", err)
	}
	value = c.invertValueIfNeeded(channel, value)
	log.Info().
		Str("device", device.Name).
		Str("channel", channel).
		Float64("value", value).
		Msg("Setting value.")
	if err := c.dsClient.DeviceSetOutputChannelValue(device.Dsid, map[string]int{channel: int(value)}); err != nil {
		return err
	}
	if err := c.publishDeviceValue(device, channel, value); err != nil {
		return err
	}
	return nil

}

func (c *DeviceModule) onDsEvent(event digitalstrom.Event) error {
	if event.Source.IsDevice {
		// The event was triggered by a single device, then let's update it.
		device := c.deviceLookup[event.Source.Dsid]
		if err := c.updateDevice(&device); err != nil {
			return fmt.Errorf("error updating device '%s': %w", device.Name, err)
		}
		return nil
	}
	devicesIds, ok := c.zoneGroupLookup[event.Source.ZoneId][event.Source.GroupId]
	if !ok {
		log.Warn().
			Int("zoneId", event.Source.ZoneId).
			Int("groupID", event.Source.GroupId).
			Msg("No devices found for group when event received.")
		return fmt.Errorf("error when retrieving device given a zone and group ID")
	}

	time.Sleep(1 * time.Second)
	for _, dsid := range devicesIds {
		device := c.deviceLookup[dsid]
		if err := c.updateDevice(&device); err != nil {
			return fmt.Errorf("error updating device '%s': %w", device.Name, err)
		}
	}

	return nil
}

func (c *DeviceModule) updateDevice(device *digitalstrom.Device) error {
	if len(device.OutputChannels) == 0 {
		log.Debug().Str("device", device.Name).Msg("Skipping update. No output channels.")
		return nil
	}
	outputChannels := device.OutputChannelsNames()
	log.Debug().
		Str("device", device.Name).
		Str("outputChannels", strings.Join(outputChannels, ";")).
		Msg("Updating device")
	response, err := c.dsClient.DeviceGetOutputChannelValue(device.Dsid, outputChannels)
	if err != nil {
		return err
	}
	for _, channelValue := range response.Channels {
		value := c.invertValueIfNeeded(channelValue.Name, channelValue.Value)
		if err := c.publishDeviceValue(device, channelValue.Name, value); err != nil {
			return fmt.Errorf("error publishing device '%s' value: %w", device.Name, err)
		}
	}

	return nil
}

func (c *DeviceModule) publishDeviceValue(device *digitalstrom.Device, channelName string, value float64) error {
	return c.mqttClient.Publish(c.deviceStateTopic(device.Name, channelName), fmt.Sprintf("%.2f", value))
}

func (c *DeviceModule) invertValueIfNeeded(channel string, value float64) float64 {
	if c.invertBlindsPosition {
		if strings.HasPrefix(strings.ToLower(channel), "shadeposition") {
			return 100 - value
		}
	}

	// nothing to invert
	return value
}

func (c *DeviceModule) deviceStateTopic(deviceName string, channel string) string {
	if c.normalizeDeviceName {
		deviceName = normalizeForTopicName(deviceName)
	}
	return path.Join(devices, deviceName, channel, mqtt.State)
}

func (c *DeviceModule) deviceCommandTopic(deviceName string, channel string) string {
	if c.normalizeDeviceName {
		deviceName = normalizeForTopicName(deviceName)
	}
	return path.Join(devices, deviceName, channel, mqtt.Command)
}

func (c *DeviceModule) GetHomeAssistantEntities() ([]homeassistant.DiscoveryConfig, error) {
	configs := []homeassistant.DiscoveryConfig{}

	for _, device := range c.devices {
		var config homeassistant.DiscoveryConfig
		if device.DeviceType() == digitalstrom.Light {
			entityConfig := &homeassistant.LightConfig{
				BaseConfig: homeassistant.BaseConfig{
					Device: homeassistant.Device{
						Identifiers: []string{
							device.Dsid,
							device.Dsuid,
						},
						Model: device.HwInfo,
						Name:  device.Name,
					},
					Name:     device.Name,
					UniqueId: device.Dsid + "_light",
				},
				CommandTopic: c.mqttClient.GetFullTopic(
					c.deviceCommandTopic(device.Name, device.OutputChannelsNames()[0])),
				StateTopic: c.mqttClient.GetFullTopic(
					c.deviceStateTopic(device.Name, device.OutputChannelsNames()[0])),
				PayloadOn:  "100.00",
				PayloadOff: "0.00",
			}
			if device.Properties().Dimmable {
				entityConfig.OnCommandType = "brightness"
				entityConfig.BrightnessScale = 100
				entityConfig.BrightnessStateTopic = c.mqttClient.GetFullTopic(
					c.deviceStateTopic(device.Name, device.OutputChannelsNames()[0]))
				entityConfig.BrightnessCommandTopic = c.mqttClient.GetFullTopic(
					c.deviceCommandTopic(device.Name, device.OutputChannelsNames()[0]))
			}
			config = homeassistant.DiscoveryConfig{
				Domain:   homeassistant.Light,
				DeviceId: device.Dsid,
				ObjectId: "light",
				Config:   entityConfig,
			}
			configs = append(configs, config)
		} else if device.DeviceType() == digitalstrom.Blind {
			deviceProperties := device.Properties()
			entityConfig := &homeassistant.CoverConfig{
				BaseConfig: homeassistant.BaseConfig{
					Device: homeassistant.Device{
						Identifiers: []string{
							device.Dsid,
							device.Dsuid,
						},
						Model: device.HwInfo,
						Name:  device.Name,
					},
					Name:     device.Name,
					UniqueId: device.Dsid + "_cover",
				},
				CommandTopic: c.mqttClient.GetFullTopic(
					c.deviceCommandTopic(device.Name, deviceProperties.PositionChannel)),
				PayloadOpen:  "100.00",
				PayloadClose: "0.00",
				PayloadStop:  "STOP",
				StateTopic: c.mqttClient.GetFullTopic(
					c.deviceStateTopic(device.Name, deviceProperties.PositionChannel)),
				StateOpen:        "100.00",
				StateClosed:      "0.00",
				PositionTopic:    c.mqttClient.GetFullTopic(c.deviceStateTopic(device.Name, deviceProperties.PositionChannel)),
				SetPositionTopic: c.mqttClient.GetFullTopic(c.deviceCommandTopic(device.Name, deviceProperties.PositionChannel)),
				PositionTemplate: "{{ value | int }}",
			}
			if deviceProperties.TiltChannel != "" {
				entityConfig.TiltStatusTemplate = "{{ value | int }}"
				entityConfig.TiltStatusTopic = c.mqttClient.GetFullTopic(
					c.deviceStateTopic(device.Name, deviceProperties.TiltChannel))
				entityConfig.TiltCommandTopic = c.mqttClient.GetFullTopic(
					c.deviceCommandTopic(device.Name, deviceProperties.TiltChannel))
			}
			config = homeassistant.DiscoveryConfig{
				Domain:   homeassistant.Cover,
				DeviceId: device.Dsid,
				ObjectId: "blind",
				Config:   entityConfig,
			}
			configs = append(configs, config)
		}
	}
	return configs, nil
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

func NewDeviceModule(mqttClient mqtt.Client, dsClient digitalstrom.Client, config *config.Config) Module {
	return &DeviceModule{
		mqttClient:           mqttClient,
		dsClient:             dsClient,
		normalizeDeviceName:  config.Mqtt.NormalizeDeviceName,
		refreshAtStart:       config.RefreshAtStart,
		invertBlindsPosition: config.InvertBlindsPosition,
		devices:              []digitalstrom.Device{},
		deviceLookup:         map[string]digitalstrom.Device{},
		zoneGroupLookup:      map[int]map[int][]string{},
	}
}

func init() {
	Register("devices", NewDeviceModule)
}
