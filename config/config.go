package config

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type ConfigDigitalstrom struct {
	Host     string
	Port     int
	Username string
	Password string
}
type ConfigMqtt struct {
	MqttUrl             string
	Username            string
	Password            string
	TopicFormat         string
	TopicPrefix         string
	NormalizeDeviceName bool
	Retain              bool
}
type ConfigHomeAssistant struct {
	DiscoveryEnabled     bool
	DiscoveryTopicPrefix string
	RemoveRegexpFromName string
	DigitalStromHost     string
	Retain               bool
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
	Undefined                               string = ""
	configFile                              string = "config.yaml"
	envKeyDigitalstromHost                  string = "DIGITALSTROM_HOST"
	envKeyDigitalstromPort                  string = "DIGITALSTROM_PORT"
	envKeyDigitalstromUsername              string = "DIGITALSTROM_USERNAME"
	envKeyDigitalstromPassword              string = "DIGITALSTROM_PASSWORD"
	envKeyMqttUrl                           string = "MQTT_URL"
	envKeyMqttUsername                      string = "MQTT_USERNAME"
	envKeyMqttPassword                      string = "MQTT_PASSWORD"
	envKeyMqttTopicFormat                   string = "MQTT_TOPIC_FORMAT"
	envKeyMqttTopicPrefix                   string = "MQTT_TOPIC_PREFIX"
	envKeyMqttNormalizeTopicName            string = "MQTT_NORMALIZE_DEVICE_NAME"
	envKeyMqttRetain                        string = "MQTT_RETAIN"
	envKeyInvertBlindsPosition              string = "INVERT_BLINDS_POSITION"
	envKeyRefreshAtStart                    string = "REFRESH_AT_START"
	envKeyLogLevel                          string = "LOG_LEVEL"
	envKeyHomeAssistantDiscoveryEnabled     string = "HOME_ASSISTANT_DISCOVERY_ENABLED"
	envKeyHomeAssistantDiscoveryPrefix      string = "HOME_ASSISTANT_DISCOVERY_PREFIX"
	envKeyHomeAssistantRemoveRegexpFromName string = "HOME_ASSISTANT_REMOVE_REGEXP_FROM_NAME"
)

func check(e error) {
	if e != nil {
		log.Panic().
			Err(e).Msg("Error when reading config")
	}
}

func readConfig(defaults map[string]interface{}) (*viper.Viper, error) {
	v := viper.New()
	for key, value := range defaults {
		v.SetDefault(key, value)
	}
	f, err := os.OpenFile(configFile, os.O_RDONLY|os.O_CREATE, 0600)
	check(err)
	f.Close()
	d, err := ioutil.ReadFile(configFile)
	check(err)
	v.SetConfigType("yaml")
	v.AutomaticEnv()
	err = v.ReadConfig(bytes.NewBuffer(d))
	return v, err
}

// FromEnv returns a Config from env variables
func FromEnv() *Config {
	v, err := readConfig(map[string]interface{}{
		envKeyDigitalstromHost:                  Undefined,
		envKeyDigitalstromPort:                  8080,
		envKeyDigitalstromUsername:              Undefined,
		envKeyDigitalstromPassword:              Undefined,
		envKeyMqttUrl:                           Undefined,
		envKeyMqttUsername:                      Undefined,
		envKeyMqttPassword:                      Undefined,
		envKeyMqttTopicPrefix:                   "digitalstrom",
		envKeyMqttTopicFormat:                   "deprecated",
		envKeyMqttNormalizeTopicName:            true,
		envKeyMqttRetain:                        false,
		envKeyRefreshAtStart:                    true,
		envKeyLogLevel:                          "INFO",
		envKeyInvertBlindsPosition:              false,
		envKeyHomeAssistantDiscoveryEnabled:     false,
		envKeyHomeAssistantDiscoveryPrefix:      "homeassistant",
		envKeyHomeAssistantRemoveRegexpFromName: "",
	})
	check(err)

	c := &Config{
		Digitalstrom: ConfigDigitalstrom{
			Host:     v.GetString(envKeyDigitalstromHost),
			Port:     v.GetInt(envKeyDigitalstromPort),
			Username: v.GetString(envKeyDigitalstromUsername),
			Password: v.GetString(envKeyDigitalstromPassword),
		},
		Mqtt: ConfigMqtt{
			MqttUrl:             v.GetString(envKeyMqttUrl),
			Username:            v.GetString(envKeyMqttUsername),
			Password:            v.GetString(envKeyMqttPassword),
			TopicFormat:         v.GetString(envKeyMqttTopicFormat),
			TopicPrefix:         v.GetString(envKeyMqttTopicPrefix),
			NormalizeDeviceName: v.GetBool(envKeyMqttNormalizeTopicName),
			Retain:              v.GetBool(envKeyMqttRetain),
		},
		HomeAssistant: ConfigHomeAssistant{
			DiscoveryEnabled:     v.GetBool(envKeyHomeAssistantDiscoveryEnabled),
			DiscoveryTopicPrefix: v.GetString(envKeyHomeAssistantDiscoveryPrefix),
			RemoveRegexpFromName: v.GetString(envKeyHomeAssistantRemoveRegexpFromName),
			DigitalStromHost:     v.GetString(envKeyDigitalstromHost),
			Retain:               v.GetBool(envKeyMqttRetain),
		},
		RefreshAtStart:       v.GetBool(envKeyRefreshAtStart),
		LogLevel:             v.GetString(envKeyLogLevel),
		InvertBlindsPosition: v.GetBool(envKeyInvertBlindsPosition),
	}

	return c
}

func (c *Config) String() string {
	return fmt.Sprintf("%+v\n", c.Digitalstrom)
}
