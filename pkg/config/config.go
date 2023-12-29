package config

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type ConfigDigitalstrom struct {
	Host string
	Port int
	// Deprecated: use apiKey instead
	Username string
	// Deprecated: use apiKey instead
	Password string
	ApiKey   string
}
type ConfigMqtt struct {
	MqttUrl             string
	Username            string
	Password            string
	TopicPrefix         string
	NormalizeDeviceName bool
	Retain              bool
}
type ConfigHomeAssistant struct {
	DiscoveryEnabled     bool
	DiscoveryTopicPrefix string
	RemoveRegexpFromName string
	DigitalStromHost     string
}
type Config struct {
	Digitalstrom         ConfigDigitalstrom
	Mqtt                 ConfigMqtt
	HomeAssistant        ConfigHomeAssistant
	RefreshAtStart       bool
	LogLevel             string
	InvertBlindsPosition bool
}

const (
	undefined                               string = "__undefined__"
	deprecated                              string = "__deprecated__"
	configFile                              string = "config.yaml"
	envKeyDigitalstromHost                  string = "digitalstrom_host"
	envKeyDigitalstromPort                  string = "digitalstrom_port"
	envKeyDigitalstromUsername              string = "digitalstrom_username"
	envKeyDigitalstromPassword              string = "digitalstrom_password"
	envKeyDigitalstromApiKey                string = "digitalstrom_api_key"
	envKeyMqttUrl                           string = "mqtt_url"
	envKeyMqttUsername                      string = "mqtt_username"
	envKeyMqttPassword                      string = "mqtt_password"
	envKeyMqttTopicFormat                   string = "mqtt_topic_format"
	envKeyMqttTopicPrefix                   string = "mqtt_topic_prefix"
	envKeyMqttNormalizeTopicName            string = "mqtt_normalize_device_name"
	envKeyMqttRetain                        string = "mqtt_retain"
	envKeyInvertBlindsPosition              string = "invert_blinds_position"
	envKeyRefreshAtStart                    string = "refresh_at_start"
	envKeyLogLevel                          string = "log_level"
	envKeyHomeAssistantDiscoveryEnabled     string = "home_assistant_discovery_enabled"
	envKeyHomeAssistantDiscoveryPrefix      string = "home_assistant_discovery_prefix"
	envKeyHomeAssistantRemoveRegexpFromName string = "home_assistant_remove_regexp_from_name"
)

var defaultConfig = map[string]interface{}{
	envKeyDigitalstromHost:                  undefined,
	envKeyDigitalstromPort:                  8080,
	envKeyDigitalstromUsername:              deprecated,
	envKeyDigitalstromPassword:              deprecated,
	envKeyDigitalstromApiKey:                undefined,
	envKeyMqttUrl:                           undefined,
	envKeyMqttUsername:                      "",
	envKeyMqttPassword:                      "",
	envKeyMqttTopicPrefix:                   "digitalstrom",
	envKeyMqttTopicFormat:                   deprecated,
	envKeyMqttNormalizeTopicName:            true,
	envKeyMqttRetain:                        true,
	envKeyRefreshAtStart:                    true,
	envKeyLogLevel:                          "INFO",
	envKeyInvertBlindsPosition:              false,
	envKeyHomeAssistantDiscoveryEnabled:     true,
	envKeyHomeAssistantDiscoveryPrefix:      "homeassistant",
	envKeyHomeAssistantRemoveRegexpFromName: "",
}

// FromEnv returns a Config from env variables
func ReadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	// Set the current directory where the binary is being run.
	viper.AddConfigPath(".")
	viper.AutomaticEnv()
	for key, value := range defaultConfig {
		if value != undefined && value != deprecated {
			viper.SetDefault(key, value)
		}
	}

	err := viper.ReadInConfig()
	if err != nil {
		log.Info().Err(err).Msg("No config file found, using environment variables only")
	}

	// Check for deprecated and undefined fields.
	for fieldName, defaultValue := range defaultConfig {
		if defaultValue == deprecated && viper.IsSet(fieldName) {
			return nil, fmt.Errorf("deprecated field found in config: %s", fieldName)
		}
		if defaultValue == undefined && !viper.IsSet(fieldName) {
			return nil, fmt.Errorf("required field not found in config: %s", fieldName)
		}
	}

	config := &Config{
		Digitalstrom: ConfigDigitalstrom{
			Host:     viper.GetString(envKeyDigitalstromHost),
			Port:     viper.GetInt(envKeyDigitalstromPort),
			Username: viper.GetString(envKeyDigitalstromUsername),
			Password: viper.GetString(envKeyDigitalstromPassword),
			ApiKey:   viper.GetString(envKeyDigitalstromApiKey),
		},
		Mqtt: ConfigMqtt{
			MqttUrl:             viper.GetString(envKeyMqttUrl),
			Username:            viper.GetString(envKeyMqttUsername),
			Password:            viper.GetString(envKeyMqttPassword),
			TopicPrefix:         viper.GetString(envKeyMqttTopicPrefix),
			NormalizeDeviceName: viper.GetBool(envKeyMqttNormalizeTopicName),
			Retain:              viper.GetBool(envKeyMqttRetain),
		},
		HomeAssistant: ConfigHomeAssistant{
			DiscoveryEnabled:     viper.GetBool(envKeyHomeAssistantDiscoveryEnabled),
			DiscoveryTopicPrefix: viper.GetString(envKeyHomeAssistantDiscoveryPrefix),
			RemoveRegexpFromName: viper.GetString(envKeyHomeAssistantRemoveRegexpFromName),
			DigitalStromHost:     viper.GetString(envKeyDigitalstromHost),
		},
		RefreshAtStart:       viper.GetBool(envKeyRefreshAtStart),
		LogLevel:             viper.GetString(envKeyLogLevel),
		InvertBlindsPosition: viper.GetBool(envKeyInvertBlindsPosition),
	}

	return config, nil
}

func (c *Config) String() string {
	return fmt.Sprintf("%+v\n", c.Digitalstrom)
}
