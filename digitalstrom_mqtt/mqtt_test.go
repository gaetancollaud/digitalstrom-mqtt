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
