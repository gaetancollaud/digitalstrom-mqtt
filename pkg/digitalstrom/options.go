package digitalstrom

import (
	"math/rand"
	"time"
)

// ClientOptions contains configurable options for a Digitalstrom Client.
type ClientOptions struct {
	Host   string
	Port   int
	ApiKey string
}

// NewClientOptions will create a new ClientClientOptions type with some
// default values.
//
//	Host: dss.local
//	Port: 8080
func NewClientOptions() *ClientOptions {
	// Random generate subscriptionId in order to not have collisions of
	// multiple instances running at the same time.
	rand.Seed(time.Now().UnixNano())

	return &ClientOptions{
		Host:   "dss.local",
		Port:   8080,
		ApiKey: "",
	}
}

// SetHost will set the address for the DigitalStrom server to connect.
func (o *ClientOptions) SetHost(host string) *ClientOptions {
	o.Host = host
	return o
}

// SetPort will set the port for the DigitalStrom server to connect.
func (o *ClientOptions) SetPort(port int) *ClientOptions {
	o.Port = port
	return o
}

// SetUsername will set the username to be used by this client when connecting
// to the DigitalStrom server.
func (o *ClientOptions) SetApiKey(u string) *ClientOptions {
	o.ApiKey = u
	return o
}
