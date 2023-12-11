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
