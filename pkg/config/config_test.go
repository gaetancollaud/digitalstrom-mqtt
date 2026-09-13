package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadConfig(t *testing.T) {
	useTestWorkingDirectory(t, "---\nmqtt_username: mqtt\n")
	setRequiredConfig(t)

	c, err := ReadConfig()
	require.NoError(t, err)

	assert.Equal(t, "test_ip", c.Digitalstrom.Host, "DigitalStrom host is wrong.")
	assert.Equal(t, "foo", c.Digitalstrom.ApiKey, "DigitalStrom API key is wrong.")
	assert.Equal(t, "mqtt", c.Mqtt.Username, "MQTT username is wrong.")
	assert.Equal(t, "digitalstrom", c.Mqtt.TopicPrefix, "MQTT prefix is wrong.")
	assert.True(t, c.MeteringsEnabled, "Meterings should be enabled by default.")
	assert.Equal(t, 10, c.MeteringsInterval, "Meterings interval is wrong.")
}

func TestReadConfigWithMeteringsEnv(t *testing.T) {
	resetViper(t)
	setRequiredConfig(t)
	t.Setenv("METERINGS_ENABLED", "false")
	t.Setenv("METERINGS_INTERVAL_SECONDS", "300")

	c, err := ReadConfig()
	require.NoError(t, err)

	assert.False(t, c.MeteringsEnabled, "Meterings enabled setting is wrong.")
	assert.Equal(t, 300, c.MeteringsInterval, "Meterings interval setting is wrong.")
}

func TestReadConfigWithInvalidMeteringsInterval(t *testing.T) {
	resetViper(t)
	setRequiredConfig(t)
	t.Setenv("METERINGS_ENABLED", "true")
	t.Setenv("METERINGS_INTERVAL_SECONDS", "0")

	_, err := ReadConfig()
	assert.EqualError(t, err, "meterings_interval_seconds must be at least 1")
}

func TestReadConfigWithInvalidMeteringsIntervalWhenDisabled(t *testing.T) {
	resetViper(t)
	setRequiredConfig(t)
	t.Setenv("METERINGS_ENABLED", "false")
	t.Setenv("METERINGS_INTERVAL_SECONDS", "0")

	_, err := ReadConfig()
	assert.EqualError(t, err, "meterings_interval_seconds must be at least 1")
}

func TestReadConfigWithDeprecatedFields(t *testing.T) {
	resetViper(t)
	setRequiredConfig(t)
	t.Setenv("MQTT_TOPIC_FORMAT", "foo")
	_, err := ReadConfig()
	assert.EqualError(t, err, "deprecated field found in config: mqtt_topic_format")
	os.Unsetenv("MQTT_TOPIC_FORMAT")
	viper.Reset()
	setRequiredConfig(t)

	t.Setenv("DIGITALSTROM_USERNAME", "foo")
	_, err = ReadConfig()
	assert.EqualError(t, err, "deprecated field found in config: digitalstrom_username")
	os.Unsetenv("DIGITALSTROM_USERNAME")
	viper.Reset()
	setRequiredConfig(t)

	t.Setenv("DIGITALSTROM_PASSWORD", "foo")
	_, err = ReadConfig()
	assert.EqualError(t, err, "deprecated field found in config: digitalstrom_password")
}

func resetViper(t *testing.T) {
	t.Helper()
	viper.Reset()
	t.Cleanup(viper.Reset)
}

func setRequiredConfig(t *testing.T) {
	t.Helper()
	t.Setenv("DIGITALSTROM_HOST", "test_ip")
	t.Setenv("DIGITALSTROM_API_KEY", "foo")
	t.Setenv("MQTT_URL", "tcp://mqtt:1883")
}

func useTestWorkingDirectory(t *testing.T, configContent string) {
	t.Helper()
	resetViper(t)

	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, "config.yaml"), []byte(configContent), 0o600))

	previousDirectory, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(directory))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(previousDirectory))
	})
}
