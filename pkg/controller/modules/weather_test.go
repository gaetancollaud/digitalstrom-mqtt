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
		ApartmentId: "apartment-1",
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
	assertPublishedWeatherValue(t, mqttClient, "weather_station_sensors/dS-Weather/temperature/state", "21.50")
	assertPublishedWeatherValue(t, mqttClient, "weather_station_sensors/dS-Weather/illuminance/state", "12345")
	assertPublishedWeatherValue(t, mqttClient, "weather_station_sensors/dS-Weather/wind_speed_10min_average/state", "2.25")
	assertPublishedWeatherValue(t, mqttClient, "weather_station_sensors/dS-Weather/wind_gust/state", "4.75")
	assertPublishedWeatherValue(t, mqttClient, "weather_station_sensors/dS-Weather/rain/state", "ON")
	if registry.callback == nil {
		t.Fatal("expected apartment status subscription")
	}
	updatedWindSpeed := 3.0
	updatedRain := false
	updatedStatus := *status
	updatedStatus.Attributes.Measurements.WindSpeed = &updatedWindSpeed
	updatedStatus.Attributes.Weather.Rain = &updatedRain
	registry.callback(status, &updatedStatus)
	assertPublishedWeatherValue(t, mqttClient, "weather_station_sensors/dS-Weather/wind_speed_10min_average/state", "3.00")
	assertPublishedWeatherValue(t, mqttClient, "weather_station_sensors/dS-Weather/rain/state", "OFF")

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
	if module.weatherSourceID != weatherSourceIDPrefix {
		t.Fatalf("expected weather source detection, got %q", module.weatherSourceID)
	}
}

func TestWeatherModuleRepresentsMultipleStationsAsOneApartmentSource(t *testing.T) {
	status := &digitalstrom.ApartmentStatus{ApartmentId: "apartment-1"}
	registry := &weatherRegistryStub{
		devices: []digitalstrom.Device{
			{DeviceId: "weather-1", Attributes: digitalstrom.DeviceAttributes{Name: "North Weather"}},
			{DeviceId: "weather-2", Attributes: digitalstrom.DeviceAttributes{Name: "South Weather"}},
		},
		functionBlocks: map[string]digitalstrom.FunctionBlock{
			"weather-1": {Attributes: digitalstrom.FunctionBlockAttributes{TechnicalName: "Weather"}},
			"weather-2": {Attributes: digitalstrom.FunctionBlockAttributes{TechnicalName: "weather"}},
		},
		status: status,
	}
	module := &WeatherModule{
		mqttClient: &weatherMQTTClientStub{prefix: "digitalstrom", published: map[string]interface{}{}},
		dsRegistry: registry,
	}

	if err := module.Start(); err != nil {
		t.Fatalf("expected weather module to start: %v", err)
	}
	if module.weatherSourceID != "apartment-weather-apartment-1" {
		t.Fatalf("expected apartment-scoped weather source, got %q", module.weatherSourceID)
	}
	configs, err := module.GetHomeAssistantEntities()
	if err != nil {
		t.Fatalf("expected Home Assistant configs: %v", err)
	}
	if len(configs) != 5 {
		t.Fatalf("expected one apartment weather source with five entities, got %d", len(configs))
	}
	for _, config := range configs {
		if config.DeviceId != "apartment-weather-apartment-1" {
			t.Fatalf("expected apartment-scoped device id, got %q", config.DeviceId)
		}
	}
}

func TestWeatherStatusEqualComparesOnlyWeatherValues(t *testing.T) {
	temperature := 21.5
	sameTemperature := 21.5
	differentTemperature := 22.0
	oldStatus := &digitalstrom.ApartmentStatus{
		Attributes: digitalstrom.ApartmentStatusAttributes{
			Measurements: digitalstrom.ApartmentMeasurements{Temperature: &temperature},
		},
		Included: digitalstrom.ApartmentStatusIncluded{
			Devices: []digitalstrom.DeviceStatus{{DeviceId: "device-1"}},
		},
	}
	sameWeather := &digitalstrom.ApartmentStatus{
		Attributes: digitalstrom.ApartmentStatusAttributes{
			Measurements: digitalstrom.ApartmentMeasurements{Temperature: &sameTemperature},
		},
		Included: digitalstrom.ApartmentStatusIncluded{
			Devices: []digitalstrom.DeviceStatus{{DeviceId: "device-2"}},
		},
	}
	changedWeather := &digitalstrom.ApartmentStatus{
		Attributes: digitalstrom.ApartmentStatusAttributes{
			Measurements: digitalstrom.ApartmentMeasurements{Temperature: &differentTemperature},
		},
	}

	if !weatherStatusEqual(oldStatus, sameWeather) {
		t.Fatal("expected equal weather values with different pointers and unrelated device status")
	}
	if weatherStatusEqual(oldStatus, changedWeather) {
		t.Fatal("expected changed weather values to differ")
	}
	if !weatherStatusEqual(nil, nil) || weatherStatusEqual(nil, oldStatus) {
		t.Fatal("expected nil apartment status to compare safely")
	}
}

func TestWeatherModuleDiscoversAllEntitiesWithPartialInitialStatus(t *testing.T) {
	temperature := 19.25
	status := &digitalstrom.ApartmentStatus{
		ApartmentId: "apartment-1",
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
			if config.DeviceId != "apartment-weather-apartment-1" {
				t.Fatalf("expected stable device id, got %s", config.DeviceId)
			}
			return
		}
	}
	t.Fatalf("missing Home Assistant config %s", objectID)
}
