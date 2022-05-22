package digitalstrom

import "strings"

type DeviceType string

const (
	Light   DeviceType = "GE"
	Blind   DeviceType = "GR"
	Joker   DeviceType = "SW"
	Unknown DeviceType = "Unknown"
)

type Action string

const (
	MoveUp        Action = "app.moveUp"
	MoveDown      Action = "app.moveDown"
	StepUp        Action = "app.stepUp"
	StepDown      Action = "app.stepDown"
	SunProtection Action = "app.sunProtection"
	Stop          Action = "app.stop"
)

type ChannelType string

const (
	Brightness ChannelType = "brightness"
	Hue        ChannelType = "hue"
)

type EventType string

const (
	EventCallScene    EventType = "callScene"
	EventUndoScene    EventType = "undoScene"
	EventButtonClick  EventType = "buttonClick"
	EventDeviceSensor EventType = "deviceSensorEvent"
	EventRunning      EventType = "running"
	EventModelReady   EventType = "model_ready"
	EventDsMeterReady EventType = "dsMeter_ready"
)

// A Device is the smallest entity represented in the DigitalStrom system and
// represents a input/output physical device that can receive human input
// (button push) or can actuate (open light).
type Device struct {
	Dsid           string          `mapstructure:"id"`
	Dsuid          string          `mapstructure:"dSUID"`
	Name           string          `mapstructure:"name"`
	HwInfo         string          `mapstructure:"hwInfo"`
	MeterDsid      string          `mapstructure:"meterDSID"`
	MeterDsuid     string          `mapstructure:"meterDSUID"`
	MeterName      string          `mapstructure:"meterName"`
	ZoneId         int             `mapstructure:"zoneID"`
	OutputChannels []OutputChannel `mapstructure:"outputChannels"`
	Groups         []int           `mapstructure:"groups"`
	OutputMode     int             `mapstructure:"outputMode"`
}

// Returns the device type given its hardware version.
func (device *Device) DeviceType() DeviceType {
	switch {
	case strings.HasPrefix(device.HwInfo, "GE"):
		return Light
	case strings.HasPrefix(device.HwInfo, "GR"):
		return Blind
	case strings.HasPrefix(device.HwInfo, "SW"):
		return Joker
	default:
		return Unknown
	}
}

// Returns some inferred properties from the device.
func (device *Device) Properties() DeviceProperties {
	positionChannel := ""
	tiltChannel := ""
	for _, channelName := range device.OutputChannelsNames() {
		if strings.Contains(channelName, "Angle") {
			tiltChannel = channelName
		}
		if strings.Contains(channelName, "Position") {
			positionChannel = channelName
		}
	}

	return DeviceProperties{
		// outputMode is set to 22 for GE devices where the
		// output is configure to be "dimmed".
		Dimmable:        device.OutputMode == 22,
		PositionChannel: positionChannel,
		TiltChannel:     tiltChannel,
	}
}

// Flattens the output channels into a slice of channel names.
func (device *Device) OutputChannelsNames() []string {
	names := make([]string, len(device.OutputChannels))
	for i, channel := range device.OutputChannels {
		names[i] = channel.Name
	}
	return names
}

// OutputChannel gives the information about an output that a Device can actuate
// on.
type OutputChannel struct {
	Name string `mapstructure:"channelName"`
}

// Properties a device can have and helps us better understand how it works.
// Note that all these properties are inferred from the attributes in the Device
// structure.
type DeviceProperties struct {
	Dimmable        bool
	PositionChannel string
	TiltChannel     string
}

// Entity that represents an electrical circuit managed by DigitaqlStrom. This
// circuit is the controller for a set of devices and can perform extra
// functions as power metering.
type Circuit struct {
	Name        string `mapstructure:"name"`
	DsId        string `mapstructure:"dsid"`
	DsUid       string `mapstructure:"dSUID"`
	HwVersion   int    `mapstructure:"hwVersion"`
	HwName      string `mapstructure:"hwName"`
	HasMetering bool   `mapstructure:"hasMetering"`
	IsValid     bool   `mapstructure:"isValid"`
	IsPresent   bool   `mapstructure:"isPresent"`
}

// Channel value information obtained from the server.
type ChannelValue struct {
	Name  string  `mapstructure:"channel"`
	Id    string  `mapstructure:"channelId"`
	Type  string  `mapstructure:"channelType"`
	Value float64 `mapstructure:"value"`
}

// Event that registers a change in the DigitalStrom system.
type Event struct {
	Name       EventType       `mapstructure:"name"`
	Properties EventProperties `mapstructure:"properties"`
	Source     EventSource     `mapstructure:"source"`
}

// Set of properties for an event.
type EventProperties struct {
	OriginToken string `mapstructure:"originToken"`
	OriginDsUid string `mapstructure:"originDSUID"`
	ZoneId      int    `mapstructure:"zoneID"`
	SceneId     int    `mapstructure:"sceneID"`
	GroupId     int    `mapstructure:"groupId"`
	CallOrigin  string `mapstructure:"callOrigin"`
	ButtonIndex int    `mapstructure:"buttonIndex"`
	ClickType   int    `mapstructure:"clickType"`
}

// Information about the source of the event responsible to fire it.
type EventSource struct {
	Dsid        string `mapstructure:"dsid"`
	ZoneId      int    `mapstructure:"zoneID"`
	GroupId     int    `mapstructure:"groupId"`
	IsApartment bool   `mapstructure:"isApartment"`
	IsGroup     bool   `mapstructure:"isGroup"`
	IsDevice    bool   `mapstructure:"isDevice"`
}

// Structure to hold a scene number and name pair.
type SceneName struct {
	Number int    `mapstructure:"sceneNr"`
	Name   string `mapstructure:"name"`
}

// Holds a float64 value.
type FloatValue struct {
	Value float64 `mapstructure:"value"`
}

// Zone representation.
type Zone struct {
	Id     int    `mapstructure:"zoneId"`
	Name   string `mapstructure:"name"`
	Groups []int  `mapstructure:"groups"`
}

// Responses for the API calls into DigitalStrom JSON API.

// Token Response when logging in into the DigitalStrom server.
// It provides the token which can be used for future calls to the server.
type TokenResponse struct {
	Token string `mapstructure:"token"`
}

type ApartmentGetCircuitsResponse struct {
	Circuits []Circuit `mapstructure:"circuits"`
}

type ApartmentGetDevicesResponse []Device

type ApartmentGetReachableGroupsResponse struct {
	Zones []Zone `mapstructure:"zones"`
}

type DeviceGetOutputChannelValueResponse struct {
	Channels []ChannelValue `mapstructure:"channels"`
}

type DeviceGetMaxMotionTimeResponse struct {
	Supported bool  `mapstructure:"supported"`
	Value     int64 `mapstructure:"value"`
}

type CircuitGetConsumptionResponse struct {
	Consumption float64 `mapstructure:"consumption"`
}

type CircuitGetEnergyMeterValueResponse struct {
	MeterValue float64 `mapstructure:"meterValue"`
}

type EventGetResponse struct {
	Events []Event `mapstructure:"events"`
}

type ZoneGetReachableScenesResponse struct {
	ReachableScenes []int       `mapstructure:"reachableScenes"`
	UserSceneNames  []SceneName `mapstructure:"userSceneNames"`
}

type ZoneGetNameResponse struct {
	Name string `mapstructure:"name"`
}

type ZoneSceneGetNameResponse struct {
	Name string `mapstructure:"name"`
}
