package digitalstrom

import (
	"errors"
	"fmt"
	"gaetancollaud/digitalstrom-mqtt/utils"
	"strconv"
	"strings"
)

type DeviceType string

const (
	Light   DeviceType = "GE"
	Blind              = "GR"
	Joker              = "SW"
	Unknown            = "Unknown"
)

type DeviceStatusChanged struct {
	Device   Device
	Channel  string
	NewValue float64
}

type DeviceCommand struct {
	DeviceName string
	Channel    string
	NewValue   float64
}

type Device struct {
	Name       string
	Dsid       string
	Dsuid      string
	DeviceType DeviceType
	MeterDsid  string
	MeterDsuid string
	MeterName  string
	ZoneId     int
	Channels   []string
	Values     map[string]float64
}

type DevicesManager struct {
	httpClient       *HttpClient
	devices          []Device
	deviceStatusChan chan DeviceStatusChanged
}

func NewDevicesManager(httpClient *HttpClient) *DevicesManager {
	dm := new(DevicesManager)
	dm.httpClient = httpClient
	dm.deviceStatusChan = make(chan DeviceStatusChanged)

	return dm
}

func (dm *DevicesManager) Start() {
	dm.reloadAllDevices()
}

func (dm *DevicesManager) reloadAllDevices() {
	response, err := dm.httpClient.get("json/apartment/getDevices")
	if utils.CheckNoError(err) {
		for _, s := range response.arrayValue {
			m := s.(map[string]interface{})
			dm.devices = append(dm.devices, Device{
				Dsid:       m["id"].(string),
				Dsuid:      m["dSUID"].(string),
				Name:       m["name"].(string),
				MeterDsid:  m["meterDSID"].(string),
				MeterDsuid: m["meterDSUID"].(string),
				MeterName:  m["meterName"].(string),
				ZoneId:     int(m["zoneID"].(float64)),
				DeviceType: extractDeviceType(m),
				Channels:   extractChannels(m),
				Values:     make(map[string]float64),
			})
		}

		//fmt.Println("Devices loaded", utils.PrettyPrintArray(dm.devices))
	}
}

func extractDeviceType(data map[string]interface{}) DeviceType {
	hwInfo := data["hwInfo"].(string)
	if strings.HasPrefix(hwInfo, "GE") {
		return Light
	}
	if strings.HasPrefix(hwInfo, "GR") {
		return Blind
	}
	if strings.HasPrefix(hwInfo, "SW") {
		return Joker
	}
	return Unknown
}

func extractChannels(data map[string]interface{}) []string {
	outputChannels := data["outputChannels"].([]interface{})

	var channels []string

	for _, outputChannel := range outputChannels {
		id := outputChannel.(map[string]interface{})["channelId"].(string)
		channels = append(channels, id)

	}
	return channels
}

func (dm *DevicesManager) getTreeFloat(path string) (float64, error) {
	response, err := dm.httpClient.get("json/property/getFloating?path=" + path)
	if utils.CheckNoError(err) {
		//fmt.Println("Properties:", prettyPrintMap(response.mapValue))
		return response.mapValue["value"].(float64), nil
	}
	return 0, err

}

func (dm *DevicesManager) updateZone(zoneId int) {
	for _, device := range dm.devices {
		if device.ZoneId == zoneId && len(device.Channels) > 0 {
			dm.updateDevice(device)
		}
	}
}

func (dm *DevicesManager) updateDevice(device Device) {
	// device need to be updated
	fmt.Println("Updating device ", device.Name)
	for _, channel := range device.Channels {
		newValue, err := dm.getTreeFloat("/apartment/zones/zone" + strconv.Itoa(device.ZoneId) + "/devices/" + device.Dsuid + "/status/outputs/" + channel + "/targetValue")
		if utils.CheckNoError(err) {
			dm.updateValue(device, channel, newValue)
		}
	}
}

func (dm *DevicesManager) updateValue(device Device, channel string, newValue float64) {
	publishValue := false
	if oldVal, ok := device.Values[channel]; ok {
		//do something here
		if oldVal != newValue {
			device.Values[channel] = newValue
			fmt.Println("Value changed", device.Name, channel, oldVal, newValue)
			publishValue = true
		}
	} else {
		// new value
		device.Values[channel] = newValue
		fmt.Println("New value", device.Name, channel, newValue)
		publishValue = true
	}
	if publishValue {
		dm.deviceStatusChan <- DeviceStatusChanged{
			Device:   device,
			Channel:  channel,
			NewValue: newValue,
		}
	}
}

func (dm *DevicesManager) SetValue(command DeviceCommand) error {
	deviceFound := false
	channelFound := false
	for _, device := range dm.devices {
		if device.Name == command.DeviceName && len(device.Channels) > 0 {
			deviceFound = true
			for _, c := range device.Channels {
				if c == command.Channel {
					channelFound = true

					fmt.Println("Setting value ", command)
					strValue := strconv.Itoa(int(command.NewValue))
					_, err := dm.httpClient.get("json/device/setOutputChannelValue?dsid=" + device.Dsid + "&channelvalues=" + c + "=" + strValue + "&applyNow=1")
					if utils.CheckNoError(err) {
						dm.updateValue(device, command.Channel, command.NewValue)
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
