package digitalstrom

// Appartment structure
type Apartment struct {
	ApartmentId string              `mapstructure:"id"`
	Attributes  ApartmentAttributes `mapstructure:"attributes"`
	Included    ApartmentIncluded   `mapstructure:"included"`
}

type ApartmentAttributes struct {
	Name     string   `mapstructure:"name"`
	Zones    []string `mapstructure:"zones"`
	Devices  []string `mapstructure:"dsDevices"`
	Clusters []string `mapstructure:"clusters"`
}

type ApartmentIncluded struct {
	Installation   Installation    `mapstructure:"installation"`
	Devices        []Device        `mapstructure:"dsDevices"`
	Submodules     []Submodule     `mapstructure:"submodules"`
	FunctionBlocks []FunctionBlock `mapstructure:"functionBlocks"`
	Zones          []Zone          `mapstructure:"zones"`
	//Scenarios      []Scenarios     `mapstructure:"scenarios"`
	// floors
	// clusters
	// dsServer
	// controllers
	// apiRevision
	Meterings []Metering `mapstructure:"meterings"`
	// userDefinedStates
	// applications
}

type Installation struct {
	InstallationId string                `mapstructure:"id"`
	Type           string                `mapstructure:"type"`
	Attributes     InstallationAttribute `mapstructure:"attributes"`
}

type InstallationAttribute struct {
	CountryCode string `mapstructure:"countryCode"`
	City        string `mapstructure:"city"`
	Timezone    string `mapstructure:"timezone"`
}

type Device struct {
	DeviceId   string           `mapstructure:"id"`
	Attributes DeviceAttributes `mapstructure:"attributes"`
}

type DeviceAttributes struct {
	Name       string   `mapstructure:"name"`
	Dsid       string   `mapstructure:"dsid"`
	DisplayId  string   `mapstructure:"displayId"`
	Present    bool     `mapstructure:"present"`
	Submodules []string `mapstructure:"submodules"`
	Zone       string   `mapstructure:"zone"`
	Scenarios  []string `mapstructure:"scenarios"`
	Controller string   `mapstructure:"controller"`
}

type Submodule struct {
	SubmoduleId string              `mapstructure:"id"`
	Attributes  SubmoduleAttributes `mapstructure:"attributes"`
}

type SubmoduleAttributes struct {
	Name           string               `mapstructure:"name"`
	TechnicalName  string               `mapstructure:"technicalName"`
	DeviceId       string               `mapstructure:"dsDevice"`
	FunctionBlocks []string             `mapstructure:"functionBlocks"`
	Zone           string               `mapstructure:"zone"`
	Application    SubmoduleApplication `mapstructure:"application"`
	Scenarios      []string             `mapstructure:"scenarios"`
	Controller     string               `mapstructure:"controller"`
}

type FunctionBlock struct {
	FunctionBlockId string                  `mapstructure:"id"`
	Attributes      FunctionBlockAttributes `mapstructure:"attributes"`
}

type FunctionBlockAttributes struct {
	Name          string         `mapstructure:"name"`
	TechnicalName string         `mapstructure:"technicalName"`
	Active        bool           `mapstructure:"active"`
	Outputs       []Output       `mapstructure:"outputs"`
	ButtonInputs  []ButtonInputs `mapstructure:"buttonInputs"`
	SensorInputs  []SensorInputs `mapstructure:"sensorInputs"`
	Submodule     string         `mapstructure:"submodule"`
	DeviceAdapter string         `mapstructure:"deviceAdapter"`
}

type Output struct {
	OutputId   string           `mapstructure:"id"`
	Attributes OutputAttributes `mapstructure:"attributes"`
}

type OutputAttributes struct {
	TechnicalName string     `mapstructure:"technicalName"`
	Type          OutputType `mapstructure:"type"`
	Function      string     `mapstructure:"function"`
	Mode          OutputMode `mapstructure:"mode"`
	Min           float64    `mapstructure:"min"`
	Max           float64    `mapstructure:"max"`
	Resolution    float64    `mapstructure:"resolution"`
}

type ButtonInputs struct {
	ButtonInputId string                 `mapstructure:"id"`
	Attributes    ButtonInputsAttributes `mapstructure:"attributes"`
}

type ButtonInputsAttributes struct {
	TechnicalName string          `mapstructure:"technicalName"`
	Type          ButtonInputType `mapstructure:"type"`
	Mode          ButtonInputMode `mapstructure:"mode"`
}

type SensorInputs struct {
	SensorInputId string                 `mapstructure:"id"`
	Attributes    SensorInputsAttributes `mapstructure:"attributes"`
}

type SensorInputsAttributes struct {
	TechnicalName string           `mapstructure:"technicalName"`
	Type          SensorInputType  `mapstructure:"type"`
	Mode          SensorInputUsage `mapstructure:"usage"`
	Min           float64          `mapstructure:"min"`
	Max           float64          `mapstructure:"max"`
	Resolution    float64          `mapstructure:"resolution"`
}

// Zone representation.
type Zone struct {
	ZoneId     string         `mapstructure:"id"`
	Attributes ZoneAttributes `mapstructure:"attributes"`
}

type ZoneAttributes struct {
	Name               string              `mapstructure:"name"`
	Floor              string              `mapstructure:"floor"`
	OrderId            float64             `mapstructure:"orderId"`
	Submodules         []string            `mapstructure:"submodules"`
	Applications       []string            `mapstructure:"applications"`
	ApplicationTypes   []string            `mapstructure:"applicationTypes"`
	ApplicationDetails []ApplicationDetail `mapstructure:"applicationDetails"`
}

type ApplicationDetail struct {
	ApplicationDetailId string `mapstructure:"id"`
	Areas               []Area `mapstructure:"areas"`
}

type Area struct {
	AreaId string `mapstructure:"id"`
	Name   string `mapstructure:"name"`
}

type Scenarios struct {
	ScenarioId string             `mapstructure:"id"`
	Type       ScenarioType       `mapstructure:"type"`
	Attributes ScenarioAttributes `mapstructure:"attributes"`
}

type ScenarioAttributes struct {
	Name        string              `mapstructure:"name"`
	ActionId    string              `mapstructure:"actionId"`
	Context     string              `mapstructure:"context"`
	Submodules  []string            `mapstructure:"submodules"`
	Devices     []string            `mapstructure:"dsDevices"`
	Zone        string              `mapstructure:"zone"`
	Application ScenarioApplication `mapstructure:"application"`
}

type Metering struct {
	MeteringId string             `mapstructure:"id"`
	Attributes MeteringAttributes `mapstructure:"attributes"`
}

type MeteringAttributes struct {
	Unit          string         `mapstructure:"unit"`
	TechnicalName string         `mapstructure:"technicalName"`
	Origin        MeteringOrigin `mapstructure:"origin"`
}

type MeteringOrigin struct {
	MeteringOriginId string `mapstructure:"id"`
	Type             string `mapstructure:"type"`
}

// Meterings
type Meterings struct {
	Meterings []Metering `mapstructure:"meterings"`
}

type MeteringValues struct {
	Values []MeteringValue `mapstructure:"values"`
}
type MeteringValue struct {
	Id         string                  `mapstructure:"id"`
	Attributes MeteringValueAttributes `mapstructure:"attributes"`
}

type MeteringValueAttributes struct {
	Value float64 `json:"value"`
}

// Status

type ApartmentStatus struct {
	ApartmentId string                  `mapstructure:"id"`
	Included    ApartmentStatusIncluded `mapstructure:"included"`
}

type ApartmentStatusIncluded struct {
	Devices []DeviceStatus `mapstructure:"dsDevices"`
}

type DeviceStatus struct {
	DeviceId   string                 `mapstructure:"id"`
	Type       string                 `mapstructure:"type"`
	Attributes DeviceStatusAttributes `mapstructure:"attributes"`
}

type DeviceStatusAttributes struct {
	FunctionBlocks []struct {
		FunctionBlockId string        `mapstructure:"id"`
		Outputs         []OutputValue `mapstructure:"outputs,omitempty"`
	} `mapstructure:"functionBlocks"`
	Submodules []struct {
		SubmoduleId      string `mapstructure:"id"`
		OperationsLocked bool   `mapstructure:"operationsLocked"`
	} `mapstructure:"submodules"`
	States []struct {
		StateId string `mapstructure:"id"`
		Value   string `mapstructure:"value"`
	} `mapstructure:"states,omitempty"`
}

type OutputValue struct {
	OutputId    string  `mapstructure:"id"`
	Value       float64 `mapstructure:"value"`
	Status      string  `mapstructure:"status"`
	TargetValue float64 `mapstructure:"targetValue"`
	Level       int     `mapstructure:"level,omitempty"`
}

// Websocket

type NotificationType string

const (
	NotificationTypeApartmentStructureChanged NotificationType = "apartmentStructureChanged"
	NotificationTypeApartmentStatusChanged    NotificationType = "apartmentStatusChanged"
)

type WebsocketInitMessage struct {
	Protocol string `json:"protocol"`
	Version  uint32 `json:"version"`
}

type WebsocketNotification struct {
	Type      int                             `json:"type"`
	Target    string                          `json:"target"`
	Arguments []WebsocketNotificationArgument `json:"arguments"`
}

type WebsocketNotificationArgument struct {
	Type NotificationType `json:"type"`
}
