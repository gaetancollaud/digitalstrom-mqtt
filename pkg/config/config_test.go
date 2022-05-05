package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadConfig(t *testing.T) {
	os.Setenv("DIGITALSTROM_HOST", "test_ip")
	os.Setenv("DIGITALSTROM_USERNAME", "foo")

	c, err := ReadConfig()
	if err != nil {
		t.Fail()
		t.Logf("Error found: %s", err.Error())
	}

	assert.Equal(t, "test_ip", c.Digitalstrom.Host, "DigitalStrom host wrong.")
	assert.Equal(t, "foo", c.Digitalstrom.Username, "DigitalStrom username wrong.")
	assert.Equal(t, "mqtt", c.Mqtt.Username, "MQTT username wrong.")
	assert.Equal(t, "digitalstrom", c.Mqtt.TopicPrefix, "MQTT prefix wrong.")
}

func TestReadConfigWithDeprecatedFields(t *testing.T) {
	os.Setenv("MQTT_TOPIC_FORMAT", "foo")
	_, err := ReadConfig()
	assert.EqualError(t, err, "deprecated field found in config: mqtt_topic_format")
}
