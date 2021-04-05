package digitalstrom_mqtt

import (
	"github.com/gaetancollaud/digitalstrom-mqtt/config"
	"testing"
)

func TestTopicGeneration(t *testing.T) {

	config := config.ConfigMqtt{
		TopicFormat: "digitalstrom/{deviceType}/{deviceName}/{channel}/{commandState}",
	}

	mqtt := DigitalstromMqtt{
		config: &config,
	}

	if mqtt.getTopic("circuits", "abc", "chan", "test") != "digitalstrom/circuits/abc/chan/test" {
		t.Errorf("wrong topic")
	}
}

func TestNormalize(t *testing.T) {
	expect(t, normalizeForTopicName("test"), "test", "Error with normalize")
	expect(t, normalizeForTopicName("test_test-test"), "test_test-test", "Error with normalize")
	expect(t, normalizeForTopicName("TeSt"), "TeSt", "Error with normalize")
	expect(t, normalizeForTopicName("test test"), "test_test", "Error with normalize")
	expect(t, normalizeForTopicName("test/test"), "test_test", "Error with normalize")
	expect(t, normalizeForTopicName("té$`^'st"), "tst", "Error with normalize")
	expect(t, normalizeForTopicName("test123"), "test123", "Error with normalize")
}

func expect(t *testing.T, result string, expect string, msg string) {
	if expect != result {
		t.Errorf("%s Expected='%s' but got '%s'", msg, expect, result)
	}
}
