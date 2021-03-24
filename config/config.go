package config

import (
	"bytes"
	"fmt"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
)

type ConfigDigitalstrom struct {
	Ip       string
	Port     int
	Username string
	Password string
}
type ConfigMqtt struct {
	MqttUrl string
	// TODO username/password

}
type Config struct {
	DigitalStrom   ConfigDigitalstrom
	Mqtt           ConfigMqtt
	RefreshAtStart bool
}

const (
	Undefined                  string = ""
	configFile                 string = "config.yaml"
	envKeyDigitalstromIp       string = "DIGITALSTROM_IP"
	envKeyDigitalstromPort     string = "DIGITALSTROM_PORT"
	envKeyDigitalstromUsername string = "DIGITALSTROM_USERNAME"
	envKeyDigitalstromPassword string = "DIGITALSTROM_PASSWORD"
	envKeyMqttUrl              string = "MQTT_URL"
	envKeyRefreshAtStart       string = "REFRESH_AT_START"
)

func check(e error) {
	if e != nil {
		panic(fmt.Errorf("Error when reading config: %v\n", e))
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
		envKeyDigitalstromIp:       Undefined,
		envKeyDigitalstromPort:     8080,
		envKeyDigitalstromUsername: Undefined,
		envKeyDigitalstromPassword: Undefined,
		envKeyMqttUrl:              Undefined,
		envKeyRefreshAtStart:       false,
	})
	check(err)

	c := &Config{
		DigitalStrom: ConfigDigitalstrom{
			Ip:       v.GetString(envKeyDigitalstromIp),
			Port:     v.GetInt(envKeyDigitalstromPort),
			Username: v.GetString(envKeyDigitalstromUsername),
			Password: v.GetString(envKeyDigitalstromPassword),
		},
		Mqtt: ConfigMqtt{
			MqttUrl: v.GetString(envKeyMqttUrl),
		},
		RefreshAtStart: v.GetBool(envKeyRefreshAtStart),
	}

	return c
}

func (c *Config) String() string {
	return fmt.Sprintf("%+v\n", c)
}
