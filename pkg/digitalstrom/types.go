package digitalstrom

type DeviceType string

const (
	DeviceTypeLight   DeviceType = "GE"
	DeviceTypeBlind   DeviceType = "GR"
	DeviceTypeJoker   DeviceType = "SW"
	DeviceTypeUnknown DeviceType = "Unknown"
)

type Action string

const (
	ActionMoveUp        Action = "app.moveUp"
	ActionMoveDown      Action = "app.moveDown"
	ActionStepUp        Action = "app.stepUp"
	ActionStepDown      Action = "app.stepDown"
	ActionSunProtection Action = "app.sunProtection"
	ActionStop          Action = "app.stop"
)

type ChannelType string

const (
	ChannelTypeBrightness ChannelType = "brightness"
	ChannelTypeHue        ChannelType = "hue"
)

// Deprecated: use new API instead
type EventType string

const (
	EventTypeCallScene    EventType = "callScene"
	EventTypeUndoScene    EventType = "undoScene"
	EventTypeButtonClick  EventType = "buttonClick"
	EventTypeDeviceSensor EventType = "deviceSensorEvent"
	EventTypeRunning      EventType = "running"
	EventTypeModelReady   EventType = "model_ready"
	EventTypeDsMeterReady EventType = "dsMeter_ready"
)

type SubmoduleApplication string

const (
	SubmoduleTypeLights        SubmoduleApplication = "lights"
	SubmoduleTypeShades        SubmoduleApplication = "shades"
	SubmoduleTypeAwnings       SubmoduleApplication = "awnings"
	SubmoduleTypeAudio         SubmoduleApplication = "audio"
	SubmoduleTypeVideo         SubmoduleApplication = "video"
	SubmoduleTypeSecurity      SubmoduleApplication = "security"
	SubmoduleTypeAccess        SubmoduleApplication = "access"
	SubmoduleTypeHeating       SubmoduleApplication = "heating"
	SubmoduleTypeCooling       SubmoduleApplication = "cooling"
	SubmoduleTypeTemperature   SubmoduleApplication = "temperature"
	SubmoduleTypeVentilation   SubmoduleApplication = "ventilation"
	SubmoduleTypeRecirculation SubmoduleApplication = "recirculation"
	SubmoduleTypeWindow        SubmoduleApplication = "window"
	SubmoduleTypeJoker         SubmoduleApplication = "joker"
)

type OutputType string

const (
	OutputTypeLightBrightness           OutputType = "lightBrightness"
	OutputTypeLightHue                  OutputType = " lightHue"
	OutputTypeLightSaturation           OutputType = " lightSaturation"
	OutputTypeLightTemperature          OutputType = " lightTemperature"
	OutputTypeLightCieX                 OutputType = " lightCieX"
	OutputTypeLightCieY                 OutputType = " lightCieY"
	OutputTypeShadePositionOutside      OutputType = " shadePositionOutside"
	OutputTypeShadePositionIndoor       OutputType = " shadePositionIndoor"
	OutputTypeShadeOpeningAngleOutside  OutputType = " shadeOpeningAngleOutside"
	OutputTypeShadeOpeningAngleIndoor   OutputType = " shadeOpeningAngleIndoor"
	OutputTypeShadeTransparency         OutputType = " shadeTransparency"
	OutputTypeAirFlowIntensity          OutputType = " airFlowIntensity"
	OutputTypeAirFlowDirection          OutputType = " airFlowDirection"
	OutputTypeAirFlapOpeningAngle       OutputType = " airFlapOpeningAngle"
	OutputTypeVentilationLouverPosition OutputType = " ventilationLouverPosition"
	OutputTypeHeatingPower              OutputType = " heatingPower"
	OutputTypeCoolingCapacity           OutputType = " coolingCapacity"
	OutputTypeAudioVolume               OutputType = " audioVolume"
	OutputTypePowerState                OutputType = " powerState"
	OutputTypeVentilationSwingMode      OutputType = " ventilationSwingMode"
	OutputTypeVentilationAutoIntensity  OutputType = " ventilationAutoIntensity"
	OutputTypeWaterTemperature          OutputType = " waterTemperature"
	OutputTypeWaterFlowRate             OutputType = " waterFlowRate"
	OutputTypePowerLevel                OutputType = " powerLevel"
	OutputTypeVideoStation              OutputType = " videoStation"
	OutputTypeVideoInputSource          OutputType = " videoInputSource"
)

type OutputMode string

const (
	OutputModeDisabled   OutputMode = "disabled"
	OutputModeSwitched   OutputMode = "switched"
	OutputModeGradual    OutputMode = "gradual"
	OutputModePositional OutputMode = "positional"
	OutputModeInternal   OutputMode = "internal"
)

type ButtonInputType string

const (
	ButtonInputTypeDevice      ButtonInputType = "device"
	ButtonInputTypeArea1       ButtonInputType = "area1"
	ButtonInputTypeArea2       ButtonInputType = "area2"
	ButtonInputTypeArea3       ButtonInputType = "area3"
	ButtonInputTypeArea4       ButtonInputType = "area4"
	ButtonInputTypeZone        ButtonInputType = "zone"
	ButtonInputTypeZone1       ButtonInputType = "zone1"
	ButtonInputTypeZone2       ButtonInputType = "zone2"
	ButtonInputTypeZone3       ButtonInputType = "zone3"
	ButtonInputTypeZone4       ButtonInputType = "zone4"
	ButtonInputTypeZonex1      ButtonInputType = "zonex1"
	ButtonInputTypeZonex2      ButtonInputType = "zonex2"
	ButtonInputTypeZonex3      ButtonInputType = "zonex3"
	ButtonInputTypeZonex4      ButtonInputType = "zonex4"
	ButtonInputTypeApplication ButtonInputType = "application"
	ButtonInputTypeGroup       ButtonInputType = "group"
	ButtonInputTypeAppmode     ButtonInputType = "appmode"
)

type ButtonInputMode string

const (
	ButtonInputModeDisabled   ButtonInputMode = "disabled"
	ButtonInputModeButton1way ButtonInputMode = "button1way"
	ButtonInputModeButton2way ButtonInputMode = "button2way"
)

type SensorInputType string

const (
	SensorInputTypeTemperature             SensorInputType = "temperature"
	SensorInputTypeBrightness              SensorInputType = "brightness"
	SensorInputTypeHumidity                SensorInputType = "humidity"
	SensorInputTypeCarbonDioxide           SensorInputType = "carbonDioxide"
	SensorInputTypeTemperatureSetpoint     SensorInputType = "temperatureSetpoint"
	SensorInputTypeTemperatureControlValue SensorInputType = "temperatureControlValue"
	SensorInputTypeEnergy                  SensorInputType = "energy"
	SensorInputTypeEnergyCounter           SensorInputType = "energyCounter"
)

type SensorInputUsage string

const (
	SensorInputUsageZone          SensorInputUsage = "zone"
	SensorInputUsageOutdoor       SensorInputUsage = "outdoor"
	SensorInputUsageSettings      SensorInputUsage = "settings"
	SensorInputUsageDevice        SensorInputUsage = "device"
	SensorInputUsageDeviceLastRun SensorInputUsage = "deviceLastRun"
	SensorInputUsageDeviceAverage SensorInputUsage = "deviceAverage"
)

type ScenarioType string

const (
	ScenarioTypeApplicationZoneScenario ScenarioType = "applicationZoneScenario"
	ScenarioTypeDeviceScenario          ScenarioType = "deviceScenario"
	ScenarioTypeUserDefinedAction       ScenarioType = "userDefinedAction"
)

type ScenarioApplication string

const (
	ScenarioApplicationLights        ScenarioApplication = "lights"
	ScenarioApplicationShades        ScenarioApplication = "shades"
	ScenarioApplicationAwnings       ScenarioApplication = "awnings"
	ScenarioApplicationAudio         ScenarioApplication = "audio"
	ScenarioApplicationVideo         ScenarioApplication = "video"
	ScenarioApplicationSecurity      ScenarioApplication = "security"
	ScenarioApplicationAccess        ScenarioApplication = "access"
	ScenarioApplicationHeating       ScenarioApplication = "heating"
	ScenarioApplicationCooling       ScenarioApplication = "cooling"
	ScenarioApplicationTemperature   ScenarioApplication = "temperature"
	ScenarioApplicationVentilation   ScenarioApplication = "ventilation"
	ScenarioApplicationRecirculation ScenarioApplication = "recirculation"
	ScenarioApplicationWindow        ScenarioApplication = "window"
	ScenarioApplicationJoker         ScenarioApplication = "joker"
)
