package config

import (
	"bytes"
	"fmt"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
)

type Config struct {
	Ip       string
	Port     int
	Username string
	Password string
	MqttUrl  string
}

const (
	Undefined                  string = ""
	configFile                 string = "config.yaml"
	envKeyDigitalstromIp       string = "DIGITALSTROM_IP"
	envKeyDigitalstromPort     string = "DIGITALSTROM_PORT"
	envKeyDigitalstromUsername string = "DIGITALSTROM_USERNAME"
	envKeyDigitalstromPassword string = "DIGITALSTROM_PASSWORD"
	envKeyMqttUrl              string = "MQTT_URL"
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
	})
	check(err)

	c := &Config{}
	c.Ip = v.GetString(envKeyDigitalstromIp)
	c.Port = v.GetInt(envKeyDigitalstromPort)
	c.Username = v.GetString(envKeyDigitalstromUsername)
	c.Password = v.GetString(envKeyDigitalstromPassword)
	c.MqttUrl = v.GetString(envKeyMqttUrl)

	return c
}

func (c *Config) String() string {
	return fmt.Sprintf("{host: %s:%s, Username: %s, Password: %s}", c.Ip, c.Port, c.Username, c.Password)
}
