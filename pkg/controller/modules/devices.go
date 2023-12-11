package modules

import (
	"fmt"
	mqtt_base "github.com/eclipse/paho.mqtt.golang"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/config"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/digitalstrom"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/homeassistant"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/mqtt"
	"github.com/rs/zerolog/log"
	"path"
	"strconv"
	"strings"
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
	dsRegistry digitalstrom.Registry

	normalizeDeviceName  bool
	refreshAtStart       bool
	invertBlindsPosition bool

	//devices              []digitalstrom.Device
	//functionBlocks       []digitalstrom.FunctionBlock
	//deviceLookup         map[string]digitalstrom.Device
	//functionBlocksLookup map[string]digitalstrom.FunctionBlock
	//zoneGroupLookup map[string]map[int][]string
}

func (c *DeviceModule) Start() error {

	//
	//// Prefetch the list of circuits available in DigitalStrom.
	//responseDevices, err := c.dsClient.ApartmentGetDevices()
	//if err != nil {
	//	log.Panic().Err(err).Msg("Error fetching the devices in the apartment.")
	//}
	//c.devices = *responseDevices
	//
	//responseFunctionBlocks, err := c.dsClient.ApartmentGetFunctionBlocks()
	//if err != nil {
	//	log.Panic().Err(err).Msg("Error fetching the function blocks in the apartment.")
	//}
	//c.functionBlocks = *responseFunctionBlocks
	//
	//// Create maps regarding Devices for fast lookup when a new Event is
	//// received.
	//for _, functionBlock := range c.functionBlocks {
	//	c.functionBlocksLookup[functionBlock.DeviceId] = functionBlock
	//}
	//for _, device := range c.devices {
	//	c.deviceLookup[device.DeviceId] = device
	//	//_, ok := c.zoneGroupLookup[device.Attributes.Zone]
	//	//if !ok {
	//	//	c.zoneGroupLookup[device.Attributes.Zone] = map[int][]string{}
	//	//}
	//	//
	//	//for _, groupId := range device.Groups {
	//	//	_, ok := c.zoneGroupLookup[device.ZoneId][groupId]
	//	//	if !ok {
	//	//		c.zoneGroupLookup[device.ZoneId][groupId] = []string{}
	//	//	}
	//	//	c.zoneGroupLookup[device.ZoneId][groupId] = append(c.zoneGroupLookup[device.ZoneId][groupId], device.Dsid)
	//	//}
	//}
	devices, err := c.dsRegistry.GetDevices()

	if err != nil {

		// TODO refresh values in registry directly

		// Refresh devices values.
		if c.refreshAtStart {
			go func() {
				for _, device := range devices {
					if err := c.updateDevice(device.DeviceId); err != nil {
						log.Error().Err(err).Msgf("Error updating device '%s'", device.Attributes.Name)
					}
				}
			}()
		}
	}

	// TODO handle that in registry
	// Subscribe to DigitalStrom events.
	if err := c.dsClient.EventSubscribe(digitalstrom.EventTypeCallScene, func(client digitalstrom.Client, event digitalstrom.Event) error {
		return c.onDsEvent(event)
	}); err != nil {
		return err
	}

	// Subscribe to MQTT events.
	for _, device := range devices {
		outputs, err := c.dsRegistry.GetOutputsOfDevice(device.DeviceId)
		if err == nil {
			for _, output := range outputs {
				deviceName := device.Attributes.Name
				outputName := output.OutputId
				topic := c.deviceCommandTopic(deviceName, outputName)
				log.Trace().
					Str("topic", topic).
					Str("deviceName", deviceName).
					Str("outputName", outputName).
					Msg("Subscribing for topic.")
				c.mqttClient.Subscribe(topic, func(client mqtt_base.Client, message mqtt_base.Message) {
					payload := string(message.Payload())
					log.Trace().
						Str("topic", topic).
						Str("deviceName", deviceName).
						Str("outputName", outputName).
						Str("payload", payload).
						Msg("Message Received.")
					if err := c.onMqttMessage(device.DeviceId, outputName, payload); err != nil {
						log.Error().
							Str("topic", topic).
							Err(err).
							Msg("Error handling MQTT Message.")
					}
				})
			}
		}
	}
	return nil
}

func (c *DeviceModule) Stop() error {
	// TODO do this in registry
	if err := c.dsClient.EventUnsubscribe(digitalstrom.EventTypeCallScene); err != nil {
		return err
	}
	return nil
}

func (c *DeviceModule) onMqttMessage(deviceId string, channel string, message string) error {
	device, err := c.dsRegistry.GetDevice(deviceId)
	if err != nil {
		return err
	}

	// In case stop is being passed as part of the message.
	if strings.ToLower(message) == stop {
		if err := c.dsClient.ZoneCallAction(device.Attributes.Zone, digitalstrom.ActionStop); err != nil {
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
		Str("device", device.Attributes.Name).
		Str("channel", channel).
		Float64("value", value).
		Msg("Setting value.")
	if err := c.dsClient.DeviceSetOutputChannelValue(device.DeviceId, map[string]int{channel: int(value)}); err != nil {
		return err
	}
	if err := c.publishDeviceValue(&device, channel, value); err != nil {
		return err
	}
	return nil

}

func (c *DeviceModule) onDsEvent(event digitalstrom.Event) error {
	// TODO refresh the all devices and make diff
	//if event.Source.IsDevice {
	//	// The event was triggered by a single device, then let's update it.
	//	device := c.deviceLookup[event.Source.Dsid]
	//	if err := c.updateDevice(&device); err != nil {
	//		return fmt.Errorf("error updating device '%s': %w", device.Name, err)
	//	}
	//	return nil
	//}
	//devicesIds, ok := c.zoneGroupLookup[event.Source.ZoneId][event.Source.GroupId]
	//if !ok {
	//	log.Warn().
	//		Int("zoneId", event.Source.ZoneId).
	//		Int("groupID", event.Source.GroupId).
	//		Msg("No devices found for group when event received.")
	//	return fmt.Errorf("error when retrieving device given a zone and group ID")
	//}
	//
	//time.Sleep(1 * time.Second)
	//for _, dsid := range devicesIds {
	//	device := c.deviceLookup[dsid]
	//	if err := c.updateDevice(&device); err != nil {
	//		return fmt.Errorf("error updating device '%s': %w", device.Name, err)
	//	}
	//}

	return nil
}

func (c *DeviceModule) updateDevice(deviceId string) error {
	device, err := c.dsRegistry.GetDevice(deviceId)
	if err != nil {
		return err
	}
	outputs, err := c.dsRegistry.GetOutputsOfDevice(deviceId)
	if err != nil {
		return err
	}
	if len(outputs) == 0 {
		log.Debug().Str("device", device.Attributes.Name).Msg("Skipping update. No output channels.")
		return nil
	}

	channels := []string{}
	for _, output := range outputs {
		channels = append(channels, output.Attributes.TechnicalName)
	}
	log.Debug().
		Str("device", device.Attributes.Name).
		Str("outputChannels", strings.Join(channels, ";")).
		Msg("Updating device")

	// TODO use registry
	//response, err := c.dsClient.DeviceGetOutputChannelValue(device.DeviceId, outputChannels)
	//if err != nil {
	//	return err
	//}
	//for _, channelValue := range response.Channels {
	//	value := c.invertValueIfNeeded(channelValue.Name, channelValue.Value)
	//	if err := c.publishDeviceValue(device, channelValue.Name, value); err != nil {
	//		return fmt.Errorf("error publishing device '%s' value: %w", device.Name, err)
	//	}
	//}

	return nil
}

func (c *DeviceModule) publishDeviceValue(device *digitalstrom.Device, channelName string, value float64) error {
	return c.mqttClient.Publish(c.deviceStateTopic(device.Attributes.Name, channelName), fmt.Sprintf("%.2f", value))
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

	devices, err := c.dsRegistry.GetDevices()
	if err != nil {
		return nil, err
	}

	for _, device := range devices {
		functionBlock, err := c.dsRegistry.GetFunctionBlockForDevice(device.DeviceId)
		if err != nil {
			return nil, err
		}
		properties := functionBlock.Properties()
		deviceType := functionBlock.DeviceType()
		var config homeassistant.DiscoveryConfig
		if deviceType == digitalstrom.DeviceTypeLight {

			outputs, err := c.dsRegistry.GetOutputsOfDevice(device.DeviceId)
			if err != nil || len(outputs) == 0 {
				log.Info().Str("deviceId", device.DeviceId).Msg("Skipping device without output channels.")
				continue
			}
			lightOutput := outputs[0]

			entityConfig := &homeassistant.LightConfig{
				BaseConfig: homeassistant.BaseConfig{
					Device: homeassistant.Device{
						Identifiers: []string{
							device.DeviceId,
						},
						Model: functionBlock.Attributes.TechnicalName,
						Name:  device.Attributes.Name,
					},
					Name:     device.Attributes.Name,
					UniqueId: device.DeviceId + "_light",
				},
				CommandTopic: c.mqttClient.GetFullTopic(
					c.deviceCommandTopic(device.Attributes.Name, lightOutput.OutputId)),
				StateTopic: c.mqttClient.GetFullTopic(
					c.deviceStateTopic(device.Attributes.Name, lightOutput.OutputId)),
				PayloadOn:  "100.00",
				PayloadOff: "0.00",
			}
			if properties.Dimmable {
				entityConfig.OnCommandType = "brightness"
				entityConfig.BrightnessScale = 100
				entityConfig.BrightnessStateTopic = c.mqttClient.GetFullTopic(
					c.deviceStateTopic(device.Attributes.Name, lightOutput.OutputId))
				entityConfig.BrightnessCommandTopic = c.mqttClient.GetFullTopic(
					c.deviceCommandTopic(device.Attributes.Name, lightOutput.OutputId))
			}
			config = homeassistant.DiscoveryConfig{
				Domain:   homeassistant.Light,
				DeviceId: device.DeviceId,
				ObjectId: "light",
				Config:   entityConfig,
			}
			configs = append(configs, config)
		} else if deviceType == digitalstrom.DeviceTypeBlind {
			entityConfig := &homeassistant.CoverConfig{
				BaseConfig: homeassistant.BaseConfig{
					Device: homeassistant.Device{
						Identifiers: []string{
							device.DeviceId,
						},
						Model: functionBlock.Attributes.TechnicalName,
						Name:  device.Attributes.Name,
					},
					Name:     device.Attributes.Name,
					UniqueId: device.DeviceId + "_cover",
				},
				CommandTopic: c.mqttClient.GetFullTopic(
					c.deviceCommandTopic(device.Attributes.Name, properties.PositionChannel)),
				PayloadOpen:  "100.00",
				PayloadClose: "0.00",
				PayloadStop:  "STOP",
				StateTopic: c.mqttClient.GetFullTopic(
					c.deviceStateTopic(device.Attributes.Name, properties.PositionChannel)),
				StateOpen:        "100.00",
				StateClosed:      "0.00",
				PositionTopic:    c.mqttClient.GetFullTopic(c.deviceStateTopic(device.Attributes.Name, properties.PositionChannel)),
				SetPositionTopic: c.mqttClient.GetFullTopic(c.deviceCommandTopic(device.Attributes.Name, properties.PositionChannel)),
				PositionTemplate: "{{ value | int }}",
			}
			if properties.TiltChannel != "" {
				entityConfig.TiltStatusTemplate = "{{ value | int }}"
				entityConfig.TiltStatusTopic = c.mqttClient.GetFullTopic(
					c.deviceStateTopic(device.Attributes.Name, properties.TiltChannel))
				entityConfig.TiltCommandTopic = c.mqttClient.GetFullTopic(
					c.deviceCommandTopic(device.Attributes.Name, properties.TiltChannel))
			}
			config = homeassistant.DiscoveryConfig{
				Domain:   homeassistant.Cover,
				DeviceId: device.DeviceId,
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

func NewDeviceModule(mqttClient mqtt.Client, dsClient digitalstrom.Client, dsRegistry digitalstrom.Registry, config *config.Config) Module {
	return &DeviceModule{
		mqttClient:           mqttClient,
		dsClient:             dsClient,
		dsRegistry:           dsRegistry,
		normalizeDeviceName:  config.Mqtt.NormalizeDeviceName,
		refreshAtStart:       config.RefreshAtStart,
		invertBlindsPosition: config.InvertBlindsPosition,
		//devices:              []digitalstrom.Device{},
		//deviceLookup:         map[string]digitalstrom.Device{},
		//zoneGroupLookup:      map[int]map[int][]string{},
	}
}

func init() {
	Register("devices", NewDeviceModule)
}
