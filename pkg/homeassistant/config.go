package homeassistant

// Interface to expose the endpoints to update any MQTT config needed by the
// Home Assistant discovery package.
type MqttConfig interface {
	// Returns a pointer to the device object for any modification required.
	GetDevice() *Device
	// Adds a new entry on the list of Availability topics.
	AddAvailability(Availability) MqttConfig
	// Get name of the entity.
	GetName() string
	// Set name for the entity.
	SetName(string) MqttConfig
	// Set retain value.
	SetRetain(bool) MqttConfig
	// Set availability mode.
	SetAvailabilityMode(string) MqttConfig
}

// Structure that encapsulates the information for the device exposed in
// Home Assistant.
type Device struct {
	ConfigurationUrl string   `json:"configuration_url"`
	Identifiers      []string `json:"identifiers"`
	Manufacturer     string   `json:"manufacturer"`
	Model            string   `json:"model,omitempty"`
	Name             string   `json:"name"`
}

// Structure that encapsulates the information to retrieve availability of
// devices and entities.
type Availability struct {
	Topic               string `json:"topic"`
	PayloadAvailable    string `json:"payload_available,omitempty"`
	PayloadNotAvailable string `json:"payload_not_available,omitempty"`
}

// Base config for all MQTT discovery configs.
type BaseConfig struct {
	Device           Device         `json:"device"`
	Name             string         `json:"name,omitempty"`
	UniqueId         string         `json:"unique_id,omitempty"`
	Retain           bool           `json:"retain"`
	Availability     []Availability `json:"availability,omitempty"`
	AvailabilityMode string         `json:"availability_mode,omitempty"`
	QoS              int            `json:"qos"`
}

// Returns a pointer to the device object.
func (c *BaseConfig) GetDevice() *Device {
	return &c.Device
}

// Adds a new entry on the list of Availability topics.
func (c *BaseConfig) AddAvailability(availability Availability) MqttConfig {
	c.Availability = append(c.Availability, availability)
	return c
}

// Get the name of the entity in the configuration.
func (c *BaseConfig) GetName() string {
	return c.Name
}

// Set the name for the entity in the configuration.
func (c *BaseConfig) SetName(name string) MqttConfig {
	c.Name = name
	return c
}

// Set retain value.
func (c *BaseConfig) SetRetain(retain bool) MqttConfig {
	c.Retain = retain
	return c
}

// Set availability mode.
func (c *BaseConfig) SetAvailabilityMode(mode string) MqttConfig {
	c.AvailabilityMode = mode
	return c
}

// Light configuration:
// https://www.home-assistant.io/integrations/light.mqtt/
type LightConfig struct {
	BaseConfig
	CommandTopic           string `json:"command_topic,omitempty"`
	StateTopic             string `json:"state_topic,omitempty"`
	StateValueTemplate     string `json:"state_value_template,omitempty"`
	PayloadOn              string `json:"payload_on,omitempty"`
	PayloadOff             string `json:"payload_off,omitempty"`
	OnCommandType          string `json:"on_command_type,omitempty"`
	BrightnessScale        int    `json:"brightness_scale,omitempty"`
	BrightnessStateTopic   string `json:"brightness_state_topic,omitempty"`
	BrightnessCommandTopic string `json:"brightness_command_topic,omitempty"`
}

// Cover configuration:
// https://www.home-assistant.io/integrations/cover.mqtt/
type CoverConfig struct {
	BaseConfig
	StateTopic         string `json:"state_topic,omitempty"`
	StateClosed        string `json:"state_closed,omitempty"`
	StateOpen          string `json:"state_open,omitempty"`
	CommandTopic       string `json:"command_topic,omitempty"`
	PayloadClose       string `json:"payload_close,omitempty"`
	PayloadOpen        string `json:"payload_open,omitempty"`
	PayloadStop        string `json:"payload_stop,omitempty"`
	PositionTopic      string `json:"position_topic,omitempty"`
	SetPositionTopic   string `json:"set_position_topic,omitempty"`
	PositionTemplate   string `json:"position_template,omitempty"`
	TiltStatusTopic    string `json:"tilt_status_topic,omitempty"`
	TiltCommandTopic   string `json:"tilt_command_topic,omitempty"`
	TiltStatusTemplate string `json:"tilt_status_template,omitempty"`
}

// Sensor configuration:
// https://www.home-assistant.io/integrations/sensor.mqtt/
type SensorConfig struct {
	BaseConfig
	StateTopic        string `json:"state_topic,omitempty"`
	UnitOfMeasurement string `json:"unit_of_measurement,omitempty"`
	DeviceClass       string `json:"device_class,omitempty"`
	StateClass        string `json:"state_class,omitempty"`
	Icon              string `json:"icon,omitempty"`
	ValueTemplate     string `json:"value_template,omitempty"`
}

// Scene configuration:
// https://www.home-assistant.io/integrations/scene.mqtt/
type SceneConfig struct {
	BaseConfig
	CommandTopic     string `json:"command_topic,omitempty"`
	PayloadOn        string `json:"payload_on,omitempty"`
	Icon             string `json:"icon,omitempty"`
	EnabledByDefault bool   `json:"enabled_by_default"`
}

// Device Trigger configuration:
// https://www.home-assistant.io/integrations/device_trigger.mqtt/
type DeviceTriggerConfig struct {
	BaseConfig
	AutomationType string `json:"automation_type"`
	Payload        string `json:"payload,omitempty"`
	Topic          string `json:"topic"`
	Type           string `json:"type"`
	Subtype        string `json:"subtype"`
}
