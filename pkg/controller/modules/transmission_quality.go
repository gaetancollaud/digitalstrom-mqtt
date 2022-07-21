package modules

import (
	"context"
	"fmt"
	"path"
	"time"

	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/config"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/digitalstrom"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/homeassistant"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/mqtt"
	"github.com/rs/zerolog/log"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Device Module encapsulates all the logic regarding the devices in your
// installation. It listens for events in DigitalStrom and messages in MQTT and
// send value updates to the other client.
type TransmissionQualityModule struct {
	mqttClient mqtt.Client
	dsClient   digitalstrom.Client

	normalizeDeviceName bool
	refreshAtStart      bool

	devices []digitalstrom.Device

	ticker     *time.Ticker
	tickerDone chan struct{}
}

func (c *TransmissionQualityModule) Start() error {
	// Prefetch the list of devices available in DigitalStrom.
	response, err := c.dsClient.ApartmentGetDevices(context.TODO())
	if err != nil {
		log.Panic().Err(err).Msg("Error fetching the devices in the apartment.")
	}
	c.devices = *response

	// Refresh devices values.
	if c.refreshAtStart {
		go c.updateDevicesTransmissionQuality()
	}

	// Create loop to test transmission quality with devices.
	c.ticker = time.NewTicker(3 * time.Minute)
	c.tickerDone = make(chan struct{})

	go func() {
		for {
			select {
			case <-c.tickerDone:
				return
			case <-c.ticker.C:
				c.updateDevicesTransmissionQuality()
			}
		}
	}()
	return nil
}

func (c *TransmissionQualityModule) Stop() error {
	c.ticker.Stop()
	c.tickerDone <- struct{}{}
	c.ticker = nil
	return nil
}

func (c *TransmissionQualityModule) updateDevicesTransmissionQuality() {
	// TODO(alberto): Add this to its own module and add also transmission
	// quality for the circuits dSM devices.
	for _, device := range c.devices {
		if device.Name == "" {
			continue
		}
		var upstream int64
		// var downstream int64
		transmissionQuality, err := c.dsClient.DeviceGetTransmissionQuality(context.TODO(), device.Dsid)
		if err != nil {
			log.Error().Err(err).Str("dsid", device.Dsid).Msg("Failed to get transmission quality for device")
			upstream = 0
			// downstream = 0
		} else {
			upstream = transmissionQuality.Upstream
			// downstream = transmissionQuality.Downstream
		}

		if err := c.mqttClient.Publish(c.deviceTransmissionQualityTopic(device.Name, "upstream"), fmt.Sprintf("%d", upstream)); err != nil {
			log.Error().Err(err).Msgf("Error updating upstream transmision quality for device '%s'", device.Name)
			continue
		}
		// if err := c.mqttClient.Publish(c.deviceTransmissionQualityTopic(device.Name, "downstream"), fmt.Sprintf("%d", downstream)); err != nil {
		// 	log.Error().Err(err).Msgf("Error updating downstream transmision quality for device '%s'", device.Name)
		// 	continue
		// }
		// TODO(alberto): publish also some attributes to the sensor like the
		// meter it is connected to, so you can aggregate this data on Home
		// Assistant.
	}
}

func (c *TransmissionQualityModule) deviceTransmissionQualityTopic(deviceName string, direction string) string {
	if c.normalizeDeviceName {
		deviceName = normalizeForTopicName(deviceName)
	}
	return path.Join(devices, deviceName, "transmission_quality", direction)
}

func (c *TransmissionQualityModule) GetHomeAssistantEntities() ([]homeassistant.DiscoveryConfig, error) {
	configs := []homeassistant.DiscoveryConfig{}

	for _, device := range c.devices {
		if device.Name == "" {
			continue
		}
		var config homeassistant.DiscoveryConfig
		for _, direction := range []string{"upstream"} {
			directionName := []byte(direction)
			cases.Title(language.AmericanEnglish).Transform(directionName, []byte(direction), true)
			entityConfig := &homeassistant.SensorConfig{
				BaseConfig: homeassistant.BaseConfig{
					Device: homeassistant.Device{
						Identifiers: []string{
							device.MeterDsid,
							device.MeterDsuid,
						},
						Name: device.MeterName,
					},
					Name:     device.Name + " Transmission Quality " + string(directionName),
					UniqueId: device.Dsid + "_transmission_quality_" + direction,
				},
				StateTopic:     c.mqttClient.GetFullTopic(c.deviceTransmissionQualityTopic(device.Name, direction)),
				EntityCategory: "diagnostic",
				Icon:           "mdi:signal",
				DeviceClass:    "signal_strength",
				StateClass:     "measurement",
			}
			config = homeassistant.DiscoveryConfig{
				Domain:   homeassistant.Sensor,
				DeviceId: device.Dsid,
				ObjectId: "transmission_quality_" + direction,
				Config:   entityConfig,
			}
			configs = append(configs, config)
		}
	}
	return configs, nil
}

func NewTransmissionQualityModule(mqttClient mqtt.Client, dsClient digitalstrom.Client, config *config.Config) Module {
	return &TransmissionQualityModule{
		mqttClient:          mqttClient,
		dsClient:            dsClient,
		normalizeDeviceName: config.Mqtt.NormalizeDeviceName,
		refreshAtStart:      config.RefreshAtStart,
		devices:             []digitalstrom.Device{},
	}
}

func init() {
	Register("transmission_quality", NewTransmissionQualityModule)
}
