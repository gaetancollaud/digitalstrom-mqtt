package digitalstrom

import (
	"fmt"
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
	if checkNoError(err) {
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

		//fmt.Println("Devices loaded", prettyPrintArray(dm.devices))
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

func (dm *DevicesManager) reloadOneDevices(dsid string) {
	response, err := dm.httpClient.get("json/apartment/getDevices?dsid=" + dsid)
	if checkNoError(err) {
		fmt.Println("Devices loaded", prettyPrintArray(response.arrayValue))
	}
}
func (dm *DevicesManager) getTreeFloat(path string) (float64, error) {
	response, err := dm.httpClient.get("json/property/getFloating?path=" + path)
	if checkNoError(err) {
		//fmt.Println("Properties:", prettyPrintMap(response.mapValue))
		return response.mapValue["value"].(float64), nil
	}
	return 0, err

}

func (dm *DevicesManager) updateZone(zoneId int) {
	for _, device := range dm.devices {
		if device.ZoneId == zoneId && len(device.Channels) > 0 {
			// device need to be updated
			fmt.Println("Updating device ", device.Name)
			for _, channel := range device.Channels {
				newValue, err := dm.getTreeFloat("/apartment/zones/zone" + strconv.Itoa(device.ZoneId) + "/devices/" + device.Dsuid + "/status/outputs/" + channel + "/targetValue")
				if checkNoError(err) {
					dm.updateValue(&device, channel, newValue)
				}
			}
		}
	}
}

func (dm *DevicesManager) updateValue(device *Device, channel string, newValue float64) {
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
			DeviceName: device.Name,
			Channel:    channel,
			NewValue:   newValue,
		}
	}
}

//
//func (dm *DevicesManager) getDeviceOutputStatus(deviceId string, valueType string) {
//	response, err := dm.httpClient.get("/json/device/getOutputValue?dsid=" + deviceId + "&type=" + valueType)
//	if checkNoError(err) {
//		fmt.Println("Device status:", prettyPrintMap(response.mapValue))
//	}
//}
//
//func (dm *DevicesManager) getDeviceOutputStatusFromOffset(deviceId string, offset int) {
//	response, err := dm.httpClient.get("/json/device/getOutputValue?dsid=" + deviceId + "&offset=" + strconv.Itoa(offset))
//	if checkNoError(err) {
//		fmt.Println("Device status:", prettyPrintMap(response.mapValue))
//	}
//}
//
//func (dm *DevicesManager) getDeviceOutputChannel(deviceId string, channels string) {
//
//	response, err := dm.httpClient.get("/json/device/getOutputChannelValue?dsid=" + deviceId + "&channels=" + channels)
//	if checkNoError(err) {
//		fmt.Println("Device channels:", prettyPrintMap(response.mapValue))
//	}
//
//}
//
//func (dm *DevicesManager) getSceneeOutputChannel(dsid string, sceneId int, channels string) {
//	response, err := dm.httpClient.get("/json/device/getOutputChannelSceneValue2?dsid=" + dsid + "&sceneNumber=" + strconv.Itoa(sceneId) + "&channels=" + channels)
//	if checkNoError(err) {
//		fmt.Println("Device channels:", prettyPrintMap(response.mapValue))
//	}
//}
//
//func (dm *DevicesManager) getInfoStatic(dsuid string) {
//	response, err := dm.httpClient.get("/json/device/getInfoStatic?dsuid=" + dsuid)
//	if checkNoError(err) {
//		fmt.Println("Device channels:", prettyPrintMap(response.mapValue))
//	}
//}
//func (dm *DevicesManager) getTreeChildren(path string) {
//	response, err := dm.httpClient.get("json/property/getChildren?path=" + path)
//	if checkNoError(err) {
//		fmt.Println("Properties:", prettyPrintArray(response.arrayValue))
//	}
//}
