package modules

import (
	"fmt"
	"path"
	"strings"

	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/config"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/digitalstrom"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/homeassistant"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/mqtt"
	"github.com/rs/zerolog/log"
)

const (
	weatherModuleID       = "weather"
	weatherStationSensors = "weather_station_sensors"
	weatherTemperature    = "temperature"
	weatherIlluminance    = "illuminance"
	weatherWindSpeed      = "wind_speed_10min_average"
	weatherWindGust       = "wind_gust"
	weatherRain           = "rain"
	weatherStateOn        = "ON"
	weatherStateOff       = "OFF"
)

type WeatherModule struct {
	mqttClient          mqtt.Client
	dsRegistry          digitalstrom.Registry
	normalizeDeviceName bool
	weatherStation      *digitalstrom.Device
	subscribed          bool
}

func (c *WeatherModule) Start() error {
	station, err := c.findWeatherStation()
	if err != nil {
		return err
	}
	if station == nil {
		log.Debug().Msg("No dS-Weather station found. Weather module disabled.")
		return nil
	}
	c.weatherStation = station
	log.Debug().
		Str("device", station.Attributes.Name).
		Msg("Weather module enabled.")

	status, err := c.dsRegistry.GetApartmentStatus()
	if err != nil {
		return fmt.Errorf("error getting apartment weather status: %w", err)
	}
	if err := c.publishWeatherStatus(status); err != nil {
		return err
	}
	if err := c.dsRegistry.ApartmentStatusChangeSubscribe(weatherModuleID, func(oldStatus *digitalstrom.ApartmentStatus, newStatus *digitalstrom.ApartmentStatus) {
		if weatherStatusEqual(oldStatus, newStatus) {
			return
		}
		if err := c.publishWeatherStatus(newStatus); err != nil {
			log.Error().Err(err).Msg("Error publishing dS-Weather status")
		}
	}); err != nil {
		return err
	}
	c.subscribed = true
	return nil
}

func (c *WeatherModule) Stop() error {
	if !c.subscribed {
		return nil
	}
	c.subscribed = false
	return c.dsRegistry.ApartmentStatusChangeUnsubscribe(weatherModuleID)
}

func (c *WeatherModule) findWeatherStation() (*digitalstrom.Device, error) {
	devices, err := c.dsRegistry.GetDevices()
	if err != nil {
		return nil, err
	}
	for _, device := range devices {
		functionBlock, err := c.dsRegistry.GetFunctionBlockForDevice(device.DeviceId)
		if err != nil {
			continue
		}
		if strings.EqualFold(functionBlock.Attributes.TechnicalName, "Weather") {
			station := device
			return &station, nil
		}
	}
	return nil, nil
}

func (c *WeatherModule) publishWeatherStatus(status *digitalstrom.ApartmentStatus) error {
	if status == nil || c.weatherStation == nil {
		return nil
	}
	measurements := status.Attributes.Measurements
	values := []struct {
		measurement string
		value       *float64
		format      string
	}{
		{weatherTemperature, measurements.Temperature, "%.2f"},
		{weatherIlluminance, measurements.Brightness, "%.0f"},
		{weatherWindSpeed, measurements.WindSpeed, "%.2f"},
		{weatherWindGust, measurements.WindGust, "%.2f"},
	}
	for _, item := range values {
		if item.value == nil {
			continue
		}
		if err := c.mqttClient.Publish(c.weatherStateTopic(item.measurement), fmt.Sprintf(item.format, *item.value)); err != nil {
			return fmt.Errorf("error publishing dS-Weather %s: %w", item.measurement, err)
		}
	}
	if rain := status.Attributes.Weather.Rain; rain != nil {
		value := weatherStateOff
		if *rain {
			value = weatherStateOn
		}
		if err := c.mqttClient.Publish(c.weatherStateTopic(weatherRain), value); err != nil {
			return fmt.Errorf("error publishing dS-Weather rain: %w", err)
		}
	}
	return nil
}

func weatherStatusEqual(oldStatus *digitalstrom.ApartmentStatus, newStatus *digitalstrom.ApartmentStatus) bool {
	if oldStatus == nil || newStatus == nil {
		return oldStatus == newStatus
	}
	oldMeasurements := oldStatus.Attributes.Measurements
	newMeasurements := newStatus.Attributes.Measurements
	return equalFloat(oldMeasurements.Temperature, newMeasurements.Temperature) &&
		equalFloat(oldMeasurements.Brightness, newMeasurements.Brightness) &&
		equalFloat(oldMeasurements.WindSpeed, newMeasurements.WindSpeed) &&
		equalFloat(oldMeasurements.WindGust, newMeasurements.WindGust) &&
		equalBool(oldStatus.Attributes.Weather.Rain, newStatus.Attributes.Weather.Rain)
}

func equalFloat(left *float64, right *float64) bool {
	if left == nil || right == nil {
		return left == right
	}
	return *left == *right
}

func equalBool(left *bool, right *bool) bool {
	if left == nil || right == nil {
		return left == right
	}
	return *left == *right
}

func (c *WeatherModule) weatherStateTopic(measurement string) string {
	deviceName := c.weatherStation.Attributes.Name
	if c.normalizeDeviceName {
		deviceName = normalizeForTopicName(deviceName)
	}
	return path.Join(weatherStationSensors, deviceName, measurement, mqtt.State)
}

func (c *WeatherModule) GetHomeAssistantEntities() ([]homeassistant.DiscoveryConfig, error) {
	if c.weatherStation == nil {
		return []homeassistant.DiscoveryConfig{}, nil
	}

	device := homeassistant.Device{
		Identifiers: []string{c.weatherStation.DeviceId},
		Model:       "dS-Weather",
		Name:        c.weatherStation.Attributes.Name,
	}
	configs := []homeassistant.DiscoveryConfig{
		c.sensorConfig(device, weatherTemperature, "°C", "temperature"),
		c.sensorConfig(device, weatherIlluminance, "lx", "illuminance"),
		c.sensorConfig(device, weatherWindSpeed, "m/s", "wind_speed"),
		c.sensorConfig(device, weatherWindGust, "m/s", "wind_speed"),
		{
			Domain:   homeassistant.BinarySensor,
			DeviceId: c.weatherStation.DeviceId,
			ObjectId: weatherRain,
			Config: &homeassistant.BinarySensorConfig{
				BaseConfig: homeassistant.BaseConfig{
					Device:   device,
					Name:     weatherRain,
					UniqueId: c.weatherStation.DeviceId + "_" + weatherRain,
				},
				StateTopic:  c.mqttClient.GetFullTopic(c.weatherStateTopic(weatherRain)),
				DeviceClass: "moisture",
				PayloadOn:   weatherStateOn,
				PayloadOff:  weatherStateOff,
				Icon:        "mdi:weather-pouring",
			},
		},
	}
	return configs, nil
}

func (c *WeatherModule) sensorConfig(device homeassistant.Device, measurement string, unit string, deviceClass string) homeassistant.DiscoveryConfig {
	return homeassistant.DiscoveryConfig{
		Domain:   homeassistant.Sensor,
		DeviceId: c.weatherStation.DeviceId,
		ObjectId: measurement,
		Config: &homeassistant.SensorConfig{
			BaseConfig: homeassistant.BaseConfig{
				Device:   device,
				Name:     measurement,
				UniqueId: c.weatherStation.DeviceId + "_" + measurement,
			},
			StateTopic:        c.mqttClient.GetFullTopic(c.weatherStateTopic(measurement)),
			UnitOfMeasurement: unit,
			DeviceClass:       deviceClass,
			StateClass:        "measurement",
		},
	}
}

func NewWeatherModule(mqttClient mqtt.Client, _ digitalstrom.Client, dsRegistry digitalstrom.Registry, config *config.Config) Module {
	return &WeatherModule{
		mqttClient:          mqttClient,
		dsRegistry:          dsRegistry,
		normalizeDeviceName: config.Mqtt.NormalizeDeviceName,
	}
}

func init() {
	Register(weatherModuleID, NewWeatherModule)
}
