package mqtt

import (
	"time"
)

// ClientOptions contains configurable options for the MQTT client responsible
// to communicate with DigitalStrom data.
type ClientOptions struct {
	MqttUrl             string
	Username            string
	Password            string
	TopicPrefix         string
	NormalizeDeviceName bool
	Retain              bool
	QoS                 byte
	DisconnectTimeout   time.Duration
}

// NewClientOptions will create a new ClientOptions type with some default
// values.
//   TopicPrefix: "digitalstrom"
//	 NormalizeDeviceName: true
// 	 Retain: false
//	 QoS: 0
//	 DisconnectTimeout: 1 second
func NewClientOptions() *ClientOptions {
	return &ClientOptions{
		MqttUrl:           "",
		Username:          "",
		Password:          "",
		TopicPrefix:       "digitalstrom",
		Retain:            false,
		QoS:               0,
		DisconnectTimeout: 1 * time.Second,
	}
}

// SetMqttUrl will set the address for the DigitalStrom server to connect.
func (o *ClientOptions) SetMqttUrl(server string) *ClientOptions {
	o.MqttUrl = server
	return o
}

// SetUsername will set the username to be used by this client when connecting
// to the MQTT server.
func (o *ClientOptions) SetUsername(u string) *ClientOptions {
	o.Username = u
	return o
}

// SetPassword will set the password to be used by this client when connecting
// to the MQTT server.
func (o *ClientOptions) SetPassword(p string) *ClientOptions {
	o.Password = p
	return o
}

// SetTopicPrefix will set the prefix that will be prepended to all the
// published messages.
func (o *ClientOptions) SetTopicPrefix(prefix string) *ClientOptions {
	o.TopicPrefix = prefix
	return o
}

// SetRetain will define the value for the retain flag for all published
// messages.
func (o *ClientOptions) SetRetain(retain bool) *ClientOptions {
	o.Retain = retain
	return o
}
