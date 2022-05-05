package digitalstrom

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/digitalstrom"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/utils"
	"github.com/rs/zerolog/log"
)

type DeviceAction string

const (
	CommandSet  DeviceAction = "set"
	CommandStop DeviceAction = "stop"
)

type DeviceStateChanged struct {
	Device   digitalstrom.Device
	Channel  string
	NewValue float64
}

type DeviceCommand struct {
	DeviceName string
	Channel    string
	Action     DeviceAction
	NewValue   float64
}

type DevicesManager struct {
	httpClient           digitalstrom.Client
	invertBlindsPosition bool
	devices              []digitalstrom.Device
	deviceStateChan      chan DeviceStateChanged
	lastDeviceCommand    time.Time
}

func NewDevicesManager(httpClient digitalstrom.Client, invertBlindsPosition bool) *DevicesManager {
	dm := new(DevicesManager)
	dm.httpClient = httpClient
	dm.invertBlindsPosition = invertBlindsPosition
	dm.deviceStateChan = make(chan DeviceStateChanged)
	dm.lastDeviceCommand = time.Now()

	return dm
}

func (dm *DevicesManager) Start() {
	dm.reloadAllDevices()
}

func (dm *DevicesManager) reloadAllDevices() {
	response, err := dm.httpClient.ApartmentGetDevices()
	if err != nil {
		log.Panic().Err(err).Msg("Unable to reload all devices")
	}
	fmt.Printf("First device: %+v\n", (*response)[0])
	for _, device := range *response {
		if len(device.Dsuid) == 0 {
			log.Info().
				Str("name", device.Name).
				Msg("Device not supported because it has no dSUID. Enable debug to see the complete devices")
			log.Debug().
				Str("device", utils.PrettyPrint(device)).
				Msg("Device not supported because it has no dSUID")
			continue
		}
		dm.devices = append(dm.devices, device)
	}
}

func (dm *DevicesManager) updateZone(zoneId int) {
	for _, device := range dm.devices {
		if device.ZoneId == zoneId {
			dm.updateDevice(device)
		}
	}
}

func (dm *DevicesManager) updateGroup(groupId int) {
	for _, device := range dm.devices {
		for _, gId := range device.Groups {
			if gId == groupId {
				log.Info().Int("Group", groupId).Str("device", device.Name).Msg("Updating device from group")
				dm.updateDevice(device)
			}
		}
	}
}

func (dm *DevicesManager) updateDevice(device digitalstrom.Device) {
	// device need to be updated
	if len(device.OutputChannels) == 0 {
		log.Debug().Str("device", device.Name).Msg("Skipping update. No output channels.")
		return
	}
	outputChannels := device.OutputChannelsNames()
	log.Debug().
		Str("device", device.Name).
		Str("outputChannels", strings.Join(outputChannels, ";")).
		Msg("Updating device")
	response, err := dm.httpClient.DeviceGetOutputChannelValue(device.Dsid, outputChannels)
	if err != nil {
		log.Error().
			Err(err).
			Str("device", device.Name).
			Msg("Unable to update device")
	} else {
		for _, channelValue := range response.Channels {
			dm.updateValue(device, channelValue.Name, channelValue.Value)
		}
	}
}

func (dm *DevicesManager) updateValue(device digitalstrom.Device, channel string, newValue float64) {
	newValue = dm.invertValueIfNeeded(channel, newValue)

	// Always send new updated value to the channel. If the current value is not
	// change this will not overload the MQTT server as is expected to run when
	// events are received from DigitalStrom.
	dm.deviceStateChan <- DeviceStateChanged{
		Device:   device,
		Channel:  channel,
		NewValue: newValue,
	}
}

func (dm *DevicesManager) SetValue(command DeviceCommand) error {
	now := time.Now()
	duration := now.Sub(dm.lastDeviceCommand)
	if duration < time.Second {
		log.Debug().
			Dur("lastCommand", duration).
			Msg("Waiting before setting value. DigitalSTROM cannot handle more than 1 change/seconds")
		time.Sleep(time.Second - duration)
	}
	dm.lastDeviceCommand = now

	deviceFound := false
	channelFound := false
	for _, device := range dm.devices {
		if device.Name == command.DeviceName && len(device.OutputChannels) > 0 {
			deviceFound = true
			for _, c := range device.OutputChannels {
				if c.Name == command.Channel {
					channelFound = true

					newValue := dm.invertValueIfNeeded(c.Name, command.NewValue)

					log.Info().
						Str("device", command.DeviceName).
						Str("channel", command.Channel).
						Str("action", string(command.Action)).
						Float64("value", newValue).
						Msg("Setting value ")

					var err error
					if command.Action == CommandStop {
						err = dm.httpClient.ZoneCallAction(device.ZoneId, digitalstrom.Stop)
					} else {
						err = dm.httpClient.DeviceSetOutputChannelValue(device.Dsid, map[string]int{c.Name: int(newValue)})
					}
					if utils.CheckNoErrorAndPrint(err) {
						dm.updateValue(device, command.Channel, newValue)
					}
				}
			}
		}
	}
	if !deviceFound {
		return errors.New("No device '" + command.DeviceName + "' found")
	}
	if !channelFound {
		return errors.New("No channel '" + command.Channel + "' found on device '" + command.DeviceName + "'")
	}
	return nil
}

func (dm *DevicesManager) invertValueIfNeeded(channel string, value float64) float64 {
	if dm.invertBlindsPosition {
		if strings.HasPrefix(strings.ToLower(channel), "shadeposition") {
			return 100 - value
		}
	}

	// nothing to invert
	return value
}
