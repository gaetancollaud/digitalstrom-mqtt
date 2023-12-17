package digitalstrom

import "strings"

// Returns the device type given its hardware technicalName
func (functionBlock *FunctionBlock) DeviceType() DeviceType {
	switch {
	case strings.HasPrefix(functionBlock.Attributes.TechnicalName, "GE"):
		return DeviceTypeLight
	case strings.HasPrefix(functionBlock.Attributes.TechnicalName, "GR"):
		return DeviceTypeBlind
	case strings.HasPrefix(functionBlock.Attributes.TechnicalName, "SW"):
		return DeviceTypeJoker
	default:
		return DeviceTypeUnknown
	}
}

// Properties a device can have and helps us better understand how it works.
// Note that all these properties are inferred from the attributes in the Device
// structure.
type DeviceProperties struct {
	Dimmable        bool
	PositionChannel string
	TiltChannel     string
}

// Returns some inferred properties from the device.
func (functionBlock *FunctionBlock) Properties() DeviceProperties {
	positionChannel := ""
	tiltChannel := ""
	dimmable := false
	for _, outputs := range functionBlock.Attributes.Outputs {
		if strings.Contains(outputs.OutputId, "Angle") {
			tiltChannel = outputs.OutputId
		}
		if strings.Contains(outputs.OutputId, "Position") {
			positionChannel = outputs.OutputId
		}
		if outputs.Attributes.Mode == OutputModeGradual {
			dimmable = true
		}
	}

	return DeviceProperties{
		Dimmable:        dimmable,
		PositionChannel: positionChannel,
		TiltChannel:     tiltChannel,
	}
}
