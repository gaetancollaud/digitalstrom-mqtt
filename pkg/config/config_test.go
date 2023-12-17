package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadConfig(t *testing.T) {
	os.Setenv("DIGITALSTROM_HOST", "test_ip")
	os.Setenv("DIGITALSTROM_API_KEY", "foo")

	c, err := ReadConfig()
	if err != nil {
		t.Fail()
		t.Logf("Error found: %s", err.Error())
	}

	assert.Equal(t, "test_ip", c.Digitalstrom.Host, "DigitalStrom host is wrong.")
	assert.Equal(t, "foo", c.Digitalstrom.ApiKey, "DigitalStrom api key is wrong.")
	assert.Equal(t, "mqtt", c.Mqtt.Username, "MQTT username is wrong.")
	assert.Equal(t, "digitalstrom", c.Mqtt.TopicPrefix, "MQTT prefix is wrong.")
}

func TestReadConfigWithDeprecatedFields(t *testing.T) {
	os.Setenv("MQTT_TOPIC_FORMAT", "foo")
	_, err := ReadConfig()
	assert.EqualError(t, err, "deprecated field found in config: mqtt_topic_format")
	os.Clearenv()

	os.Setenv("DIGITALSTROM_USERNAME", "foo")
	_, err = ReadConfig()
	assert.EqualError(t, err, "deprecated field found in config: digitalstrom_username")
	os.Clearenv()

	os.Setenv("DIGITALSTROM_PASSWORD", "foo")
	_, err = ReadConfig()
	assert.EqualError(t, err, "deprecated field found in config: digitalstrom_password")
	os.Clearenv()
}
