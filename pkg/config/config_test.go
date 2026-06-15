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
	assert.True(t, c.MeteringsEnabled, "Meterings should be enabled by default.")
	assert.Equal(t, 10, c.MeteringsInterval, "Meterings interval is wrong.")
}

func TestReadConfigWithMeteringsEnv(t *testing.T) {
	os.Setenv("DIGITALSTROM_HOST", "test_ip")
	os.Setenv("DIGITALSTROM_API_KEY", "foo")
	os.Setenv("METERINGS_ENABLED", "false")
	os.Setenv("METERINGS_INTERVAL_SECONDS", "300")
	defer os.Clearenv()

	c, err := ReadConfig()
	if err != nil {
		t.Fail()
		t.Logf("Error found: %s", err.Error())
	}

	assert.False(t, c.MeteringsEnabled, "Meterings enabled setting is wrong.")
	assert.Equal(t, 300, c.MeteringsInterval, "Meterings interval setting is wrong.")
}

func TestReadConfigWithInvalidMeteringsInterval(t *testing.T) {
	os.Setenv("DIGITALSTROM_HOST", "test_ip")
	os.Setenv("DIGITALSTROM_API_KEY", "foo")
	os.Setenv("METERINGS_ENABLED", "true")
	os.Setenv("METERINGS_INTERVAL_SECONDS", "0")
	defer os.Clearenv()

	_, err := ReadConfig()
	assert.EqualError(t, err, "meterings_interval_seconds must be at least 1")
}

func TestReadConfigWithInvalidMeteringsIntervalWhenDisabled(t *testing.T) {
	os.Setenv("DIGITALSTROM_HOST", "test_ip")
	os.Setenv("DIGITALSTROM_API_KEY", "foo")
	os.Setenv("METERINGS_ENABLED", "false")
	os.Setenv("METERINGS_INTERVAL_SECONDS", "0")
	defer os.Clearenv()

	_, err := ReadConfig()
	assert.EqualError(t, err, "meterings_interval_seconds must be at least 1")
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
