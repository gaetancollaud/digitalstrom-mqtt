package modules

import (
	"errors"
	"path"
	"testing"

	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/digitalstrom"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/homeassistant"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/mqtt"
)

type weatherMQTTClientStub struct {
	mqtt.Client
	prefix       string
	published    map[string]interface{}
	publishErr   error
	publishCalls int
}

func (c *weatherMQTTClientStub) Publish(topic string, value interface{}) error {
	c.publishCalls++
	if c.publishErr != nil {
		return c.publishErr
	}
	c.published[topic] = value
	return nil
}

func (c *weatherMQTTClientStub) GetFullTopic(topic string) string {
	return path.Join(c.prefix, topic)
}

type weatherRegistryStub struct {
	digitalstrom.Registry
	devices             []digitalstrom.Device
	devicesErr          error
	functionBlocks      map[string]digitalstrom.FunctionBlock
	functionBlockLists  map[string][]digitalstrom.FunctionBlock
	functionBlockErrors map[string]error
	status              *digitalstrom.ApartmentStatus
	statusErr           error
	callback            digitalstrom.ApartmentStatusChangeCallback
	subscribeErr        error
	unsubscribeErrors   []error
	unsubscribeCalls    int
}

func (r *weatherRegistryStub) GetDevices() ([]digitalstrom.Device, error) {
	return r.devices, r.devicesErr
}

func (r *weatherRegistryStub) GetFunctionBlockForDevice(deviceID string) (digitalstrom.FunctionBlock, error) {
	return r.functionBlocks[deviceID], nil
}

func (r *weatherRegistryStub) GetFunctionBlocksForDevice(deviceID string) ([]digitalstrom.FunctionBlock, error) {
	if err := r.functionBlockErrors[deviceID]; err != nil {
		return nil, err
	}
	if functionBlocks, ok := r.functionBlockLists[deviceID]; ok {
		return functionBlocks, nil
	}
	if functionBlock, ok := r.functionBlocks[deviceID]; ok {
		return []digitalstrom.FunctionBlock{functionBlock}, nil
	}
	return []digitalstrom.FunctionBlock{}, nil
}

func (r *weatherRegistryStub) GetApartmentStatus() (*digitalstrom.ApartmentStatus, error) {
	return r.status, r.statusErr
}

func (r *weatherRegistryStub) ApartmentStatusChangeSubscribe(_ string, callback digitalstrom.ApartmentStatusChangeCallback) error {
	if r.subscribeErr != nil {
		return r.subscribeErr
	}
	r.callback = callback
	return nil
}

func (r *weatherRegistryStub) ApartmentStatusChangeUnsubscribe(_ string) error {
	r.unsubscribeCalls++
	if len(r.unsubscribeErrors) > 0 {
		err := r.unsubscribeErrors[0]
		r.unsubscribeErrors = r.unsubscribeErrors[1:]
		if err != nil {
			return err
		}
	}
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

func TestWeatherModuleFindsWeatherFunctionBlockAmongMultipleBlocks(t *testing.T) {
	registry := &weatherRegistryStub{
		devices: []digitalstrom.Device{{DeviceId: "weather-1"}},
		functionBlockLists: map[string][]digitalstrom.FunctionBlock{
			"weather-1": {
				{Attributes: digitalstrom.FunctionBlockAttributes{TechnicalName: "Sensor"}},
				{Attributes: digitalstrom.FunctionBlockAttributes{TechnicalName: "Weather"}},
			},
		},
		status: &digitalstrom.ApartmentStatus{},
	}
	module := &WeatherModule{
		mqttClient: &weatherMQTTClientStub{published: map[string]interface{}{}},
		dsRegistry: registry,
	}

	if err := module.Start(); err != nil {
		t.Fatalf("expected weather module to start: %v", err)
	}
	if module.weatherSourceID != weatherSourceIDPrefix {
		t.Fatalf("expected weather source detection, got %q", module.weatherSourceID)
	}
}

func TestWeatherModuleStaysDisabledWithoutWeatherStation(t *testing.T) {
	mqttClient := &weatherMQTTClientStub{published: map[string]interface{}{}}
	registry := &weatherRegistryStub{
		devices: []digitalstrom.Device{{DeviceId: "light-1"}},
		functionBlocks: map[string]digitalstrom.FunctionBlock{
			"light-1": {Attributes: digitalstrom.FunctionBlockAttributes{TechnicalName: "Light"}},
		},
	}
	module := &WeatherModule{mqttClient: mqttClient, dsRegistry: registry}

	if err := module.Start(); err != nil {
		t.Fatalf("expected weather module to stay disabled: %v", err)
	}
	if module.weatherSourceID != "" || registry.callback != nil || mqttClient.publishCalls != 0 {
		t.Fatal("expected disabled weather module without discovery or publication")
	}
}

func TestWeatherModuleSkipsUnreadableDeviceAndFindsNextStation(t *testing.T) {
	registry := &weatherRegistryStub{
		devices: []digitalstrom.Device{{DeviceId: "broken-1"}, {DeviceId: "weather-1"}},
		functionBlocks: map[string]digitalstrom.FunctionBlock{
			"weather-1": {Attributes: digitalstrom.FunctionBlockAttributes{TechnicalName: "Weather"}},
		},
		functionBlockErrors: map[string]error{"broken-1": errors.New("function blocks unavailable")},
		status:              &digitalstrom.ApartmentStatus{},
	}
	module := &WeatherModule{
		mqttClient: &weatherMQTTClientStub{published: map[string]interface{}{}},
		dsRegistry: registry,
	}

	if err := module.Start(); err != nil {
		t.Fatalf("expected unreadable device to be isolated: %v", err)
	}
	if module.weatherSourceID != weatherSourceIDPrefix {
		t.Fatalf("expected readable weather station to enable module, got %q", module.weatherSourceID)
	}
}

func TestWeatherModuleStartReturnsDependencyErrors(t *testing.T) {
	tests := []struct {
		name       string
		registry   *weatherRegistryStub
		mqttClient *weatherMQTTClientStub
	}{
		{
			name:     "devices",
			registry: &weatherRegistryStub{devicesErr: errors.New("devices unavailable")},
		},
		{
			name:     "apartment status",
			registry: weatherRegistryWithError(errors.New("status unavailable"), nil),
		},
		{
			name:       "initial publish",
			registry:   weatherRegistryWithError(nil, nil),
			mqttClient: &weatherMQTTClientStub{published: map[string]interface{}{}, publishErr: errors.New("mqtt unavailable")},
		},
		{
			name:     "subscription",
			registry: weatherRegistryWithError(nil, errors.New("subscribe unavailable")),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mqttClient := test.mqttClient
			if mqttClient == nil {
				mqttClient = &weatherMQTTClientStub{published: map[string]interface{}{}}
			}
			module := &WeatherModule{mqttClient: mqttClient, dsRegistry: test.registry}
			if err := module.Start(); err == nil {
				t.Fatal("expected weather module startup error")
			}
		})
	}
}

func TestWeatherModuleRetriesFailedRuntimePublish(t *testing.T) {
	temperature := 20.0
	status := &digitalstrom.ApartmentStatus{
		Attributes: digitalstrom.ApartmentStatusAttributes{
			Measurements: digitalstrom.ApartmentMeasurements{Temperature: &temperature},
		},
	}
	mqttClient := &weatherMQTTClientStub{published: map[string]interface{}{}}
	registry := weatherRegistryWithError(nil, nil)
	registry.status = status
	module := &WeatherModule{mqttClient: mqttClient, dsRegistry: registry}
	if err := module.Start(); err != nil {
		t.Fatalf("expected weather module to start: %v", err)
	}

	updatedTemperature := 21.0
	updatedStatus := &digitalstrom.ApartmentStatus{
		Attributes: digitalstrom.ApartmentStatusAttributes{
			Measurements: digitalstrom.ApartmentMeasurements{Temperature: &updatedTemperature},
		},
	}
	mqttClient.publishErr = errors.New("temporary mqtt failure")
	registry.callback(status, updatedStatus)
	failedCallCount := mqttClient.publishCalls
	mqttClient.publishErr = nil
	registry.callback(updatedStatus, updatedStatus)

	if mqttClient.publishCalls <= failedCallCount {
		t.Fatal("expected unchanged apartment notification to retry failed weather publication")
	}
	assertPublishedWeatherValue(t, mqttClient, "weather_station_sensors/dS-Weather/temperature/state", "21.00")
}

func TestWeatherModuleRetriesFailedUnsubscribe(t *testing.T) {
	registry := weatherRegistryWithError(nil, nil)
	registry.unsubscribeErrors = []error{errors.New("temporary unsubscribe failure"), nil}
	module := &WeatherModule{
		mqttClient: &weatherMQTTClientStub{published: map[string]interface{}{}},
		dsRegistry: registry,
	}
	if err := module.Start(); err != nil {
		t.Fatalf("expected weather module to start: %v", err)
	}

	if err := module.Stop(); err == nil {
		t.Fatal("expected first unsubscribe attempt to fail")
	}
	if !module.subscribed || registry.callback == nil {
		t.Fatal("expected failed unsubscribe to preserve subscription state")
	}
	if err := module.Stop(); err != nil {
		t.Fatalf("expected unsubscribe retry to succeed: %v", err)
	}
	if module.subscribed || registry.callback != nil || registry.unsubscribeCalls != 2 {
		t.Fatal("expected successful retry to clear subscription state")
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

func weatherRegistryWithError(statusErr error, subscribeErr error) *weatherRegistryStub {
	temperature := 20.0
	return &weatherRegistryStub{
		devices: []digitalstrom.Device{{DeviceId: "weather-1"}},
		functionBlocks: map[string]digitalstrom.FunctionBlock{
			"weather-1": {Attributes: digitalstrom.FunctionBlockAttributes{TechnicalName: "Weather"}},
		},
		status: &digitalstrom.ApartmentStatus{
			Attributes: digitalstrom.ApartmentStatusAttributes{
				Measurements: digitalstrom.ApartmentMeasurements{Temperature: &temperature},
			},
		},
		statusErr:    statusErr,
		subscribeErr: subscribeErr,
	}
}
