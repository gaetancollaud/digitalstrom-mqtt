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

// Device Module encapsulates all the logic regarding the devices. The logic
// is the following: devices output values can be changed from mqtt and forwarded to digitalstrom on the opposite
// side, when an event is received from digitalstrom, the new value is pushed to mqtt.
type DeviceModule struct {
	mqttClient mqtt.Client
	dsClient   digitalstrom.Client
	dsRegistry digitalstrom.Registry

	normalizeDeviceName  bool
	refreshAtStart       bool
	invertBlindsPosition bool
}

func (c *DeviceModule) Start() error {
	devices, err := c.dsRegistry.GetDevices()

	for _, device := range devices {
		err := c.dsRegistry.DeviceChangeSubscribe(device.DeviceId, func(deviceId string, outputId string, oldValue float64, newValue float64) {
			err := c.updateDevice(deviceId)
			if err != nil {
				log.Error().Err(err).Str("deviceid", deviceId).Msg("Error updating device ")
			}
		})
		if err != nil {
			return err
		}
	}

	if err == nil {
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

	// Subscribe to MQTT events.
	for _, device := range devices {
		outputs, err := c.dsRegistry.GetOutputsOfDevice(device.DeviceId)
		if err == nil {
			for _, output := range outputs {
				deviceId := device.DeviceId          // deep copy
				deviceName := device.Attributes.Name // deep copy
				outputName := output.OutputId        // deep copy
				topic := c.deviceCommandTopic(deviceName, outputName)
				log.Trace().
					Str("topic", topic).
					Str("deviceName", deviceName).
					Str("outputName", outputName).
					Msg("Subscribing for topic.")
				err := c.mqttClient.Subscribe(topic, func(client mqtt_base.Client, message mqtt_base.Message) {
					payload := string(message.Payload())
					log.Trace().
						Str("topic", topic).
						Str("deviceName", deviceName).
						Str("outputName", outputName).
						Str("payload", payload).
						Msg("Message Received.")
					if err := c.onMqttMessage(deviceId, outputName, payload); err != nil {
						log.Error().
							Str("topic", topic).
							Err(err).
							Msg("Error handling MQTT Message.")
					}
				})
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (c *DeviceModule) Stop() error {
	if devices, err := c.dsRegistry.GetDevices(); err != nil {
		for _, device := range devices {
			_ = c.dsRegistry.DeviceChangeUnsubscribe(device.DeviceId)
		}
	}

	return nil
}

func (c *DeviceModule) onMqttMessage(deviceId string, outputId string, message string) error {
	device, err := c.dsRegistry.GetDevice(deviceId)
	if err != nil {
		return err
	}

	value, err := strconv.ParseFloat(strings.TrimSpace(message), 64)
	if err != nil {
		return fmt.Errorf("error parsing message as float value: %w", err)
	}
	value = c.invertValueIfNeeded(outputId, value)
	log.Info().
		Str("device", device.Attributes.Name).
		Str("outputId", outputId).
		Float64("value", value).
		Msg("Setting value.")

	functionBlock, err := c.dsRegistry.GetFunctionBlockForDevice(deviceId)
	if err != nil {
		return fmt.Errorf("no function block found for device %s: %w", deviceId, err)
	}

	err = c.dsClient.DeviceSetOutputValue(deviceId, functionBlock.FunctionBlockId, outputId, value)
	if err != nil {
		return err
	}

	// for fast deliveries we confirm the state
	if err := c.publishDeviceValue(&device, outputId, value); err != nil {
		return err
	}

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

	outputValues, err := c.dsRegistry.GetOutputValuesOfDevice(deviceId)
	if err != nil {
		return err
	}
	outputValuesLookup := map[string]digitalstrom.OutputValue{}
	for _, outputValue := range outputValues {
		outputValuesLookup[outputValue.OutputId] = outputValue
	}

	for _, output := range outputs {
		outputValue := outputValuesLookup[output.OutputId]
		value := c.invertValueIfNeeded(output.OutputId, outputValue.TargetValue)
		if err := c.publishDeviceValue(&device, output.OutputId, value); err != nil {
			return fmt.Errorf("error publishing device '%s' value: %w", device.Attributes.Name, err)
		}
	}

	return nil
}

func (c *DeviceModule) publishDeviceValue(device *digitalstrom.Device, outputId string, value float64) error {
	return c.mqttClient.Publish(c.deviceStateTopic(device.Attributes.Name, outputId), fmt.Sprintf("%.2f", value))
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
		var cfg homeassistant.DiscoveryConfig
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
					Name:     "light",
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
				entityConfig.StateValueTemplate = "{% if value|int > 0 %}100.00{% else %}0.00{% endif %}"
			}
			cfg = homeassistant.DiscoveryConfig{
				Domain:   homeassistant.Light,
				DeviceId: device.DeviceId,
				ObjectId: "light",
				Config:   entityConfig,
			}
			configs = append(configs, cfg)
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
					Name:     "cover",
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
			cfg = homeassistant.DiscoveryConfig{
				Domain:   homeassistant.Cover,
				DeviceId: device.DeviceId,
				ObjectId: "cover",
				Config:   entityConfig,
			}
			configs = append(configs, cfg)
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
	}
}

func init() {
	Register("devices", NewDeviceModule)
}
