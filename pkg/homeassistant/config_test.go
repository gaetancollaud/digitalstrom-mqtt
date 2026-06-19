package homeassistant

import (
	"encoding/json"
	"testing"
)

func TestBinarySensorConfig(t *testing.T) {
	config := BinarySensorConfig{
		BaseConfig: BaseConfig{
			Name:     "motion",
			UniqueId: "device_motion",
		},
		StateTopic:    "digitalstrom/motion/state",
		DeviceClass:   "motion",
		PayloadOn:     "ON",
		PayloadOff:    "OFF",
		Icon:          "mdi:motion-sensor",
		ValueTemplate: "{{ value_json.state }}",
	}

	payload, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Expected binary sensor config to marshal: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(payload, &result); err != nil {
		t.Fatalf("Expected binary sensor config to unmarshal: %v", err)
	}

	expectEqual(t, result["state_topic"], "digitalstrom/motion/state")
	expectEqual(t, result["device_class"], "motion")
	expectEqual(t, result["payload_on"], "ON")
	expectEqual(t, result["payload_off"], "OFF")
	expectEqual(t, result["icon"], "mdi:motion-sensor")
	expectEqual(t, result["value_template"], "{{ value_json.state }}")
	expectEqual(t, string(BinarySensor), "binary_sensor")
}

func expectEqual(t *testing.T, got interface{}, want interface{}) {
	t.Helper()

	if want != got {
		t.Errorf("Expected='%v' but got '%v'", want, got)
	}
}
