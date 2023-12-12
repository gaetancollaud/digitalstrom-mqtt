package digitalstrom

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

// Responses for the API calls into DigitalStrom JSON API.

// Token Response when logging in into the DigitalStrom server.
// It provides the token which can be used for future calls to the server.
type TokenResponse struct {
	Token string `mapstructure:"token"`
}

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
