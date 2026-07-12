package modules

import (
	"path"
	"testing"

	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/digitalstrom"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/homeassistant"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/mqtt"
)

type weatherMQTTClientStub struct {
	mqtt.Client
	prefix    string
	published map[string]interface{}
}

func (c *weatherMQTTClientStub) Publish(topic string, value interface{}) error {
	c.published[topic] = value
	return nil
}

func (c *weatherMQTTClientStub) GetFullTopic(topic string) string {
	return path.Join(c.prefix, topic)
}

type weatherRegistryStub struct {
	digitalstrom.Registry
	devices        []digitalstrom.Device
	functionBlocks map[string]digitalstrom.FunctionBlock
	status         *digitalstrom.ApartmentStatus
	callback       digitalstrom.ApartmentStatusChangeCallback
}

func (r *weatherRegistryStub) GetDevices() ([]digitalstrom.Device, error) {
	return r.devices, nil
}

func (r *weatherRegistryStub) GetFunctionBlockForDevice(deviceID string) (digitalstrom.FunctionBlock, error) {
	return r.functionBlocks[deviceID], nil
}

func (r *weatherRegistryStub) GetApartmentStatus() (*digitalstrom.ApartmentStatus, error) {
	return r.status, nil
}

func (r *weatherRegistryStub) ApartmentStatusChangeSubscribe(_ string, callback digitalstrom.ApartmentStatusChangeCallback) error {
	r.callback = callback
	return nil
}

func (r *weatherRegistryStub) ApartmentStatusChangeUnsubscribe(_ string) error {
	r.callback = nil
	return nil
}

func TestWeatherModulePublishesModernApartmentStatus(t *testing.T) {
	temperature := 21.5
	brightness := 12345.0
	windSpeed := 2.25
	windGust := 4.75
	rain := true
	status := &digitalstrom.ApartmentStatus{
		Attributes: digitalstrom.ApartmentStatusAttributes{
			Measurements: digitalstrom.ApartmentMeasurements{
				Temperature: &temperature,
				Brightness:  &brightness,
				WindSpeed:   &windSpeed,
				WindGust:    &windGust,
			},
			Weather: digitalstrom.ApartmentWeatherStatus{Rain: &rain},
		},
	}
	mqttClient := &weatherMQTTClientStub{prefix: "digitalstrom", published: map[string]interface{}{}}
	registry := &weatherRegistryStub{
		devices: []digitalstrom.Device{{
			DeviceId: "weather-1",
			Attributes: digitalstrom.DeviceAttributes{
				Name: "Roof Weather",
			},
		}},
		functionBlocks: map[string]digitalstrom.FunctionBlock{
			"weather-1": {Attributes: digitalstrom.FunctionBlockAttributes{TechnicalName: "Weather"}},
		},
		status: status,
	}
	module := &WeatherModule{mqttClient: mqttClient, dsRegistry: registry, normalizeDeviceName: true}

	if err := module.Start(); err != nil {
		t.Fatalf("expected weather module to start: %v", err)
	}
	assertPublishedWeatherValue(t, mqttClient, "weather_station_sensors/Roof_Weather/temperature/state", "21.50")
	assertPublishedWeatherValue(t, mqttClient, "weather_station_sensors/Roof_Weather/illuminance/state", "12345")
	assertPublishedWeatherValue(t, mqttClient, "weather_station_sensors/Roof_Weather/wind_speed_10min_average/state", "2.25")
	assertPublishedWeatherValue(t, mqttClient, "weather_station_sensors/Roof_Weather/wind_gust/state", "4.75")
	assertPublishedWeatherValue(t, mqttClient, "weather_station_sensors/Roof_Weather/rain/state", "ON")
	if registry.callback == nil {
		t.Fatal("expected apartment status subscription")
	}
	updatedWindSpeed := 3.0
	updatedRain := false
	updatedStatus := *status
	updatedStatus.Attributes.Measurements.WindSpeed = &updatedWindSpeed
	updatedStatus.Attributes.Weather.Rain = &updatedRain
	registry.callback(status, &updatedStatus)
	assertPublishedWeatherValue(t, mqttClient, "weather_station_sensors/Roof_Weather/wind_speed_10min_average/state", "3.00")
	assertPublishedWeatherValue(t, mqttClient, "weather_station_sensors/Roof_Weather/rain/state", "OFF")

	configs, err := module.GetHomeAssistantEntities()
	if err != nil {
		t.Fatalf("expected Home Assistant configs: %v", err)
	}
	if len(configs) != 5 {
		t.Fatalf("expected five Home Assistant configs, got %d", len(configs))
	}
	assertWeatherConfig(t, configs, weatherTemperature, homeassistant.Sensor)
	assertWeatherConfig(t, configs, weatherIlluminance, homeassistant.Sensor)
	assertWeatherConfig(t, configs, weatherWindSpeed, homeassistant.Sensor)
	assertWeatherConfig(t, configs, weatherWindGust, homeassistant.Sensor)
	assertWeatherConfig(t, configs, weatherRain, homeassistant.BinarySensor)

	if err := module.Stop(); err != nil {
		t.Fatalf("expected weather module to stop: %v", err)
	}
	if registry.callback != nil {
		t.Fatal("expected apartment status subscription to be removed")
	}
}

func TestWeatherModuleUsesTechnicalNameInsteadOfDisplayName(t *testing.T) {
	module := &WeatherModule{
		mqttClient: &weatherMQTTClientStub{published: map[string]interface{}{}},
		dsRegistry: &weatherRegistryStub{
			devices: []digitalstrom.Device{{DeviceId: "weather-1", Attributes: digitalstrom.DeviceAttributes{Name: "Custom Device Name"}}},
			functionBlocks: map[string]digitalstrom.FunctionBlock{
				"weather-1": {Attributes: digitalstrom.FunctionBlockAttributes{TechnicalName: "Weather"}},
			},
			status: &digitalstrom.ApartmentStatus{},
		},
	}
	if err := module.Start(); err != nil {
		t.Fatalf("expected weather module to recognize technical name: %v", err)
	}
	if module.weatherStation == nil || module.weatherStation.DeviceId != "weather-1" {
		t.Fatalf("expected weather station detection, got %#v", module.weatherStation)
	}
}

func TestWeatherModuleDiscoversAllEntitiesWithPartialInitialStatus(t *testing.T) {
	temperature := 19.25
	status := &digitalstrom.ApartmentStatus{
		Attributes: digitalstrom.ApartmentStatusAttributes{
			Measurements: digitalstrom.ApartmentMeasurements{Temperature: &temperature},
		},
	}
	mqttClient := &weatherMQTTClientStub{prefix: "digitalstrom", published: map[string]interface{}{}}
	registry := &weatherRegistryStub{
		devices: []digitalstrom.Device{{
			DeviceId:   "weather-1",
			Attributes: digitalstrom.DeviceAttributes{Name: "Partial Weather"},
		}},
		functionBlocks: map[string]digitalstrom.FunctionBlock{
			"weather-1": {Attributes: digitalstrom.FunctionBlockAttributes{TechnicalName: "Weather"}},
		},
		status: status,
	}
	module := &WeatherModule{mqttClient: mqttClient, dsRegistry: registry}

	if err := module.Start(); err != nil {
		t.Fatalf("expected weather module to start: %v", err)
	}
	if len(mqttClient.published) != 1 {
		t.Fatalf("expected only the available value to be published, got %#v", mqttClient.published)
	}
	configs, err := module.GetHomeAssistantEntities()
	if err != nil {
		t.Fatalf("expected Home Assistant configs: %v", err)
	}
	if len(configs) != 5 {
		t.Fatalf("expected all five weather entities, got %#v", configs)
	}
}

func assertPublishedWeatherValue(t *testing.T, client *weatherMQTTClientStub, topic string, expected interface{}) {
	t.Helper()
	if actual := client.published[topic]; actual != expected {
		t.Fatalf("expected %s=%v, got %v", topic, expected, actual)
	}
}

func assertWeatherConfig(t *testing.T, configs []homeassistant.DiscoveryConfig, objectID string, domain homeassistant.Domain) {
	t.Helper()
	for _, config := range configs {
		if config.ObjectId == objectID {
			if config.Domain != domain {
				t.Fatalf("expected %s domain %s, got %s", objectID, domain, config.Domain)
			}
			if config.DeviceId != "weather-1" {
				t.Fatalf("expected stable device id, got %s", config.DeviceId)
			}
			return
		}
	}
	t.Fatalf("missing Home Assistant config %s", objectID)
}
