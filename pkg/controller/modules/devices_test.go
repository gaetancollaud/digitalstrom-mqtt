package modules

import (
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"
	"testing"

	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/digitalstrom"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/homeassistant"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/mqtt"
)

type deviceMQTTClientStub struct {
	mqtt.Client
	prefix string
}

func (c *deviceMQTTClientStub) Publish(string, interface{}) error {
	return nil
}

func (c *deviceMQTTClientStub) GetFullTopic(topic string) string {
	return path.Join(c.prefix, topic)
}

type deviceSetOutputCall struct {
	deviceID        string
	functionBlockID string
	outputID        string
	value           float64
}

type deviceDigitalstromClientStub struct {
	digitalstrom.Client
	setOutputCalls []deviceSetOutputCall
	scenarioIDs    []string
	scenarioError  error
}

func (c *deviceDigitalstromClientStub) DeviceSetOutputValue(deviceID string, functionBlockID string, outputID string, value float64) error {
	c.setOutputCalls = append(c.setOutputCalls, deviceSetOutputCall{
		deviceID:        deviceID,
		functionBlockID: functionBlockID,
		outputID:        outputID,
		value:           value,
	})
	return nil
}

func (c *deviceDigitalstromClientStub) InvokeScenarioByID(scenarioID string) error {
	c.scenarioIDs = append(c.scenarioIDs, scenarioID)
	return c.scenarioError
}

type deviceRegistryStub struct {
	digitalstrom.Registry
	devices        map[string]digitalstrom.Device
	functionBlocks map[string]digitalstrom.FunctionBlock
}

func (r *deviceRegistryStub) GetDevices() ([]digitalstrom.Device, error) {
	devices := make([]digitalstrom.Device, 0, len(r.devices))
	for _, device := range r.devices {
		devices = append(devices, device)
	}
	return devices, nil
}

func (r *deviceRegistryStub) GetDevice(deviceID string) (digitalstrom.Device, error) {
	device, ok := r.devices[deviceID]
	if !ok {
		return digitalstrom.Device{}, fmt.Errorf("device %s not found", deviceID)
	}
	return device, nil
}

func (r *deviceRegistryStub) GetFunctionBlockForDevice(deviceID string) (digitalstrom.FunctionBlock, error) {
	functionBlock, ok := r.functionBlocks[deviceID]
	if !ok {
		return digitalstrom.FunctionBlock{}, fmt.Errorf("function block for device %s not found", deviceID)
	}
	return functionBlock, nil
}

func (r *deviceRegistryStub) GetOutputsOfDevice(deviceID string) ([]digitalstrom.Output, error) {
	functionBlock, err := r.GetFunctionBlockForDevice(deviceID)
	if err != nil {
		return nil, err
	}
	return functionBlock.Attributes.Outputs, nil
}

func TestDeviceModuleInvokesAdvertisedStopScenarioCaseInsensitively(t *testing.T) {
	for _, payload := range []string{" STOP ", "stop"} {
		t.Run(payload, func(t *testing.T) {
			dsClient := &deviceDigitalstromClientStub{}
			module := testDeviceModule(dsClient, "blind-1", "Office Blind", testBlindFunctionBlock())

			err := module.onMqttMessage("blind-1", "shadePositionOutside", payload)

			if err != nil {
				t.Fatalf("expected blind stop payload to invoke device action, got error: %v", err)
			}
			if len(dsClient.setOutputCalls) != 0 {
				t.Fatalf("expected no output writes for stop payload, got %d", len(dsClient.setOutputCalls))
			}
			if len(dsClient.scenarioIDs) != 1 {
				t.Fatalf("expected one device scenario, got %d", len(dsClient.scenarioIDs))
			}
			if dsClient.scenarioIDs[0] != "advertised-blind-stop-std.stop" {
				t.Fatalf("unexpected device scenario ID: %q", dsClient.scenarioIDs[0])
			}
		})
	}
}

func TestDeviceModuleReturnsStopScenarioError(t *testing.T) {
	expectedError := errors.New("device action failed")
	dsClient := &deviceDigitalstromClientStub{scenarioError: expectedError}
	module := testDeviceModule(dsClient, "blind-1", "Office Blind", testBlindFunctionBlock())

	err := module.onMqttMessage("blind-1", "shadePositionOutside", "STOP")

	if !errors.Is(err, expectedError) {
		t.Fatalf("expected scenario error, got %v", err)
	}
	if len(dsClient.scenarioIDs) != 1 {
		t.Fatalf("expected one scenario invocation, got %d", len(dsClient.scenarioIDs))
	}
}

func TestDeviceModuleReturnsErrorWhenClientCannotInvokeScenarios(t *testing.T) {
	dsClient := &deviceClientWithoutScenarioStub{}
	module := testDeviceModule(dsClient, "blind-1", "Office Blind", testBlindFunctionBlock())

	err := module.onMqttMessage("blind-1", "shadePositionOutside", "STOP")

	if err == nil || !strings.Contains(err.Error(), "does not support scenario invocation") {
		t.Fatalf("expected unsupported scenario error, got %v", err)
	}
}

func TestDeviceModuleRejectsStopWhenDeviceDoesNotAdvertiseScenario(t *testing.T) {
	dsClient := &deviceDigitalstromClientStub{}
	module := testDeviceModule(dsClient, "blind-1", "Office Blind", testBlindFunctionBlock())
	registry := module.dsRegistry.(*deviceRegistryStub)
	device := registry.devices["blind-1"]
	device.Attributes.Scenarios = nil
	registry.devices["blind-1"] = device

	err := module.onMqttMessage("blind-1", "shadePositionOutside", "STOP")

	if err == nil || !strings.Contains(err.Error(), "does not advertise") {
		t.Fatalf("expected missing stop scenario error, got %v", err)
	}
	if len(dsClient.scenarioIDs) != 0 {
		t.Fatalf("expected no scenario invocation, got %d", len(dsClient.scenarioIDs))
	}
}

func TestDeviceModuleRejectsStopPayloadForNonBlind(t *testing.T) {
	dsClient := &deviceDigitalstromClientStub{}
	module := testDeviceModule(dsClient, "light-1", "Office Light", testLightFunctionBlock())

	err := module.onMqttMessage("light-1", "brightness", "stop")

	if err == nil {
		t.Fatal("expected STOP payload on non-blind device to fail")
	}
	if len(dsClient.setOutputCalls) != 0 {
		t.Fatalf("expected no output writes for invalid STOP payload, got %d", len(dsClient.setOutputCalls))
	}
	if len(dsClient.scenarioIDs) != 0 {
		t.Fatalf("expected no device scenario for invalid STOP payload, got %d", len(dsClient.scenarioIDs))
	}
}

func TestDeviceModuleStillWritesNumericOutputValues(t *testing.T) {
	dsClient := &deviceDigitalstromClientStub{}
	module := testDeviceModule(dsClient, "blind-1", "Office Blind", testBlindFunctionBlock())

	err := module.onMqttMessage("blind-1", "shadePositionOutside", "42.50")

	if err != nil {
		t.Fatalf("expected numeric payload to be written, got error: %v", err)
	}
	if len(dsClient.scenarioIDs) != 0 {
		t.Fatalf("expected no scenario for numeric payload, got %d", len(dsClient.scenarioIDs))
	}
	if len(dsClient.setOutputCalls) != 1 {
		t.Fatalf("expected one output write, got %d", len(dsClient.setOutputCalls))
	}
	call := dsClient.setOutputCalls[0]
	if call.deviceID != "blind-1" || call.functionBlockID != "fb-blind-1" || call.outputID != "shadePositionOutside" || call.value != 42.50 {
		t.Fatalf("unexpected output write: %#v", call)
	}
}

func TestDeviceModuleCoverDiscoveryAdvertisesStopCommand(t *testing.T) {
	dsClient := &deviceDigitalstromClientStub{}
	module := testDeviceModule(dsClient, "blind-1", "Office Blind", testBlindFunctionBlock())

	configs, err := module.GetHomeAssistantEntities()

	if err != nil {
		t.Fatalf("expected discovery config, got error: %v", err)
	}
	if len(configs) != 1 || configs[0].Domain != homeassistant.Cover {
		t.Fatalf("expected one cover discovery config, got %#v", configs)
	}
	coverConfig, ok := configs[0].Config.(*homeassistant.CoverConfig)
	if !ok {
		t.Fatalf("expected cover config, got %T", configs[0].Config)
	}
	if coverConfig.PayloadStop != mqttStopCommand {
		t.Fatalf("expected payload_stop %s, got %q", mqttStopCommand, coverConfig.PayloadStop)
	}
	expectedTopic := "digitalstrom/devices/Office Blind/shadePositionOutside/command"
	if coverConfig.CommandTopic != expectedTopic {
		t.Fatalf("expected stop command topic %q, got %q", expectedTopic, coverConfig.CommandTopic)
	}
	payload, err := json.Marshal(coverConfig)
	if err != nil {
		t.Fatalf("marshal cover discovery config: %v", err)
	}
	var discoveryPayload map[string]interface{}
	if err := json.Unmarshal(payload, &discoveryPayload); err != nil {
		t.Fatalf("decode cover discovery config: %v", err)
	}
	if discoveryPayload["payload_stop"] != mqttStopCommand {
		t.Fatalf("expected serialized payload_stop %s, got %#v", mqttStopCommand, discoveryPayload["payload_stop"])
	}
	if discoveryPayload["command_topic"] != expectedTopic {
		t.Fatalf("expected serialized command_topic %q, got %#v", expectedTopic, discoveryPayload["command_topic"])
	}
}

func TestDeviceModuleCoverDiscoveryOmitsStopWithoutAdvertisedScenario(t *testing.T) {
	module := testDeviceModule(&deviceDigitalstromClientStub{}, "blind-1", "Office Blind", testBlindFunctionBlock())
	registry := module.dsRegistry.(*deviceRegistryStub)
	device := registry.devices["blind-1"]
	device.Attributes.Scenarios = nil
	registry.devices["blind-1"] = device

	configs, err := module.GetHomeAssistantEntities()

	if err != nil {
		t.Fatalf("expected discovery config, got error: %v", err)
	}
	coverConfig, ok := configs[0].Config.(*homeassistant.CoverConfig)
	if !ok {
		t.Fatalf("expected cover config, got %T", configs[0].Config)
	}
	if coverConfig.PayloadStop != "" {
		t.Fatalf("expected no payload_stop without advertised scenario, got %q", coverConfig.PayloadStop)
	}
	payload, err := json.Marshal(coverConfig)
	if err != nil {
		t.Fatalf("marshal cover discovery config: %v", err)
	}
	var discoveryPayload map[string]interface{}
	if err := json.Unmarshal(payload, &discoveryPayload); err != nil {
		t.Fatalf("decode cover discovery config: %v", err)
	}
	if _, found := discoveryPayload["payload_stop"]; found {
		t.Fatalf("expected serialized discovery to omit payload_stop: %#v", discoveryPayload)
	}
}

func TestDeviceModuleCoverDiscoveryOmitsStopWhenClientCannotInvokeScenarios(t *testing.T) {
	module := testDeviceModule(&deviceClientWithoutScenarioStub{}, "blind-1", "Office Blind", testBlindFunctionBlock())

	configs, err := module.GetHomeAssistantEntities()

	if err != nil {
		t.Fatalf("expected discovery config, got error: %v", err)
	}
	coverConfig, ok := configs[0].Config.(*homeassistant.CoverConfig)
	if !ok {
		t.Fatalf("expected cover config, got %T", configs[0].Config)
	}
	if coverConfig.PayloadStop != "" {
		t.Fatalf("expected no payload_stop without scenario invocation support, got %q", coverConfig.PayloadStop)
	}
	payload, err := json.Marshal(coverConfig)
	if err != nil {
		t.Fatalf("marshal cover discovery config: %v", err)
	}
	var discoveryPayload map[string]interface{}
	if err := json.Unmarshal(payload, &discoveryPayload); err != nil {
		t.Fatalf("decode cover discovery config: %v", err)
	}
	if _, found := discoveryPayload["payload_stop"]; found {
		t.Fatalf("expected serialized discovery to omit payload_stop: %#v", discoveryPayload)
	}
}

type deviceClientWithoutScenarioStub struct {
	digitalstrom.Client
}

func (c *deviceClientWithoutScenarioStub) DeviceSetOutputValue(string, string, string, float64) error {
	return nil
}

func testDeviceModule(dsClient digitalstrom.Client, deviceID string, name string, functionBlock digitalstrom.FunctionBlock) *DeviceModule {
	scenarios := []string{}
	if functionBlock.DeviceType() == digitalstrom.DeviceTypeBlind {
		scenarios = append(scenarios, "advertised-blind-stop-std.stop")
	}
	return &DeviceModule{
		dsClient:   dsClient,
		mqttClient: &deviceMQTTClientStub{prefix: "digitalstrom"},
		dsRegistry: &deviceRegistryStub{
			devices: map[string]digitalstrom.Device{
				deviceID: {
					DeviceId: deviceID,
					Attributes: digitalstrom.DeviceAttributes{
						Name:      name,
						Zone:      "9",
						Scenarios: scenarios,
					},
				},
			},
			functionBlocks: map[string]digitalstrom.FunctionBlock{
				deviceID: functionBlock,
			},
		},
	}
}

func testBlindFunctionBlock() digitalstrom.FunctionBlock {
	return digitalstrom.FunctionBlock{
		FunctionBlockId: "fb-blind-1",
		Attributes: digitalstrom.FunctionBlockAttributes{
			TechnicalName: "GR-KL200",
			Outputs: []digitalstrom.Output{
				{
					OutputId: "shadePositionOutside",
					Attributes: digitalstrom.OutputAttributes{
						Mode: digitalstrom.OutputModePositional,
					},
				},
			},
		},
	}
}

func testLightFunctionBlock() digitalstrom.FunctionBlock {
	return digitalstrom.FunctionBlock{
		FunctionBlockId: "fb-light-1",
		Attributes: digitalstrom.FunctionBlockAttributes{
			TechnicalName: "GE-KM200",
			Outputs: []digitalstrom.Output{
				{
					OutputId: "brightness",
					Attributes: digitalstrom.OutputAttributes{
						Mode: digitalstrom.OutputModeGradual,
					},
				},
			},
		},
	}
}
