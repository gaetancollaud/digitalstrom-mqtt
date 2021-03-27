package digitalstrom_mqtt

import (
	"testing"
)

func TestTopicGeneration(t *testing.T) {

	format := "digitalstrom/{deviceType}/{deviceName}/{channel}/{commandStatus}"

	if getTopic(format, "circuits", "abc", "chan", "test") != "digitalstrom/circuits/abc/chan/test" {
		t.Errorf("wrong topic")
	}
}

func TestTopicExtraction(t *testing.T) {

	format := "digitalstrom/{deviceType}/{deviceName}/{channel}/{commandStatus}"
	topic := "digitalstrom/circuits/abc/chan/test"

	err, deviceType, deviceName, channel, statusCommand := extractFromTopic(format, topic)

	if err != nil {
		t.Errorf("parsing error")
	}

	if deviceType != "circuits" {
		t.Errorf("wrong deviceType")
	}

	if deviceName != "abc" {
		t.Errorf("wrong deviceName")
	}

	if channel != "chan" {
		t.Errorf("wrong channel")
	}

	if statusCommand != "test" {
		t.Errorf("wrong statusCommand")
	}
}
