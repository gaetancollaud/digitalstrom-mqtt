package digitalstrom

// type ApartmentGetCircuitsResponse struct {
// 	Result CircuitList `json:"result"`
// }

type ApartmentGetCircuitsResponse struct {
	Circuits []Circuit `json:"circuits"`
}

type Circuit struct {
	Name        string `json:"name"`
	Dsid        string `json:"dsid"`
	HwVersion   int    `json:"hwVersion"`
	HwName      string `json:"hwName"`
	HasMetering bool   `json:"hasMetering"`
	IsValid     bool   `json:"isValid"`
	IsPresent   bool   `json:"isPresent"`
}

// type ApartmentGetDevicesResponse struct {
// 	Devices []DsDevice `mapstructure:",squash"`
// }

type ApartmentGetDevicesResponse []DsDevice

type DsDevice struct {
	Dsid           string          `mapstructure:"id"`
	Name           string          `json:"name"`
	Dsuid          string          `json:"dsuid"`
	HwInfo         string          `json:"hwInfo"`
	MeterDsid      string          `json:"meterDSID"`
	MeterDsuid     string          `json:"meterDSUID"`
	MeterName      string          `json:"meterName"`
	ZoneId         int             `json:"zoneID"`
	OutputChannels []OutputChannel `json:"outputChannels"`
	Groups         []int           `json:"groups"`
}

type OutputChannel struct {
	ChannelName string `json:"channelName"`
}
