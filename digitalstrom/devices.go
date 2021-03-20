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

type Device struct {
	Name       string
	Dsid       string
	Dsuid      string
	DeviceType DeviceType
	MeterDsid  string
	MeterDsuid string
	MeterName  string
	ZoneId     int
}

type DevicesManager struct {
	httpClient *HttpClient
	devices    []Device
}

func NewDevicesManager(httpClient *HttpClient) *DevicesManager {
	dm := new(DevicesManager)
	dm.httpClient = httpClient

	return dm
}

func (dm *DevicesManager) Start() {
	dm.reloadAllDevices()

	//store gaetan
	//dm.getDeviceOutputStatus("303505d7f8000f80001071bd", "position")
	//dm.getDeviceOutputStatus("303505d7f8000f80001071bd", "angle")
	//dm.getDeviceOutputStatusFromOffset("303505d7f8000f80001071bd", 2)
	//dm.getDeviceOutputStatusFromOffset("303505d7f8000f80001071bd", 0)

	// light gaetan
	//dm.getDeviceOutputStatusFromOffset("303505d7f80000400013befc", 0)
	//dm.getDeviceOutputChannel("303505d7f8000000000000400013befc00")
	//dm.getDeviceOutputChannel("303505d7f80000400013befc", "powerState;brightness;saturation;hue;colortemp")
	//dm.getDeviceOutputChannel("303505d7f80000400013befc", "brightness")
	//dm.getInfoStatic("303505d7f8000000000000400013befc00")

	// gaetan's room
	//dm.getSceneeOutputChannel("303505d7f80000400013befc", 50, "brightness;saturation")

	//dm.getTree("/system/version/version")
	//dm.getTree("/system/host/interfaces/lo")
	//dm.getTreeChildren("/apartment/zones/zone5/devices/303505d7f8000000000000400013befc00/status/outputs/brightness")
	dm.getTreeFloat("/apartment/zones/zone5/devices/303505d7f8000000000000400013befc00/status/outputs/brightness/targetValue")
	dm.getTreeFloat("/apartment/zones/zone5/devices/303505d7f800000000000f80001071bd00/status/outputs/shadePositionOutside/targetValue")
	dm.getTreeFloat("/apartment/zones/zone5/devices/303505d7f800000000000f80001071bd00/status/outputs/shadeOpeningAngleOutside/targetValue")
	//dm.getTree("/apartment/zones/{*}(ZoneID,scenes)/groups/{*}(group,name)/scenes/{*}(scene,name)")
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
				DeviceType: extractDeviceType(m["hwInfo"].(string)),
			})
		}

		fmt.Println("Devices loaded", prettyPrintArray(dm.devices))
	}
}

func extractDeviceType(hwInfo string) DeviceType {
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

func (dm *DevicesManager) reloadOneDevices(dsid string) {
	response, err := dm.httpClient.get("json/apartment/getDevices?dsid=" + dsid)
	if checkNoError(err) {
		fmt.Println("Devices loaded", prettyPrintArray(response.arrayValue))
	}
}

func (dm *DevicesManager) getDeviceOutputStatus(deviceId string, valueType string) {
	response, err := dm.httpClient.get("/json/device/getOutputValue?dsid=" + deviceId + "&type=" + valueType)
	if checkNoError(err) {
		fmt.Println("Device status:", prettyPrintMap(response.mapValue))
	}
}

func (dm *DevicesManager) getDeviceOutputStatusFromOffset(deviceId string, offset int) {
	response, err := dm.httpClient.get("/json/device/getOutputValue?dsid=" + deviceId + "&offset=" + strconv.Itoa(offset))
	if checkNoError(err) {
		fmt.Println("Device status:", prettyPrintMap(response.mapValue))
	}
}

func (dm *DevicesManager) getDeviceOutputChannel(deviceId string, channels string) {

	response, err := dm.httpClient.get("/json/device/getOutputChannelValue?dsid=" + deviceId + "&channels=" + channels)
	if checkNoError(err) {
		fmt.Println("Device channels:", prettyPrintMap(response.mapValue))
	}

}

func (dm *DevicesManager) getSceneeOutputChannel(dsid string, sceneId int, channels string) {
	response, err := dm.httpClient.get("/json/device/getOutputChannelSceneValue2?dsid=" + dsid + "&sceneNumber=" + strconv.Itoa(sceneId) + "&channels=" + channels)
	if checkNoError(err) {
		fmt.Println("Device channels:", prettyPrintMap(response.mapValue))
	}
}

func (dm *DevicesManager) getInfoStatic(dsuid string) {
	response, err := dm.httpClient.get("/json/device/getInfoStatic?dsuid=" + dsuid)
	if checkNoError(err) {
		fmt.Println("Device channels:", prettyPrintMap(response.mapValue))
	}
}
func (dm *DevicesManager) getTreeFloat(path string) {
	response, err := dm.httpClient.get("json/property/getFloating?path=" + path)
	if checkNoError(err) {
		fmt.Println("Properties:", prettyPrintMap(response.mapValue))
	}
}
func (dm *DevicesManager) getTreeChildren(path string) {
	response, err := dm.httpClient.get("json/property/getChildren?path=" + path)
	if checkNoError(err) {
		fmt.Println("Properties:", prettyPrintArray(response.arrayValue))
	}
}
