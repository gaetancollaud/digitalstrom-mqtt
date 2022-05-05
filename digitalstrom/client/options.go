package client

import (
	"math/rand"
	"time"

	"github.com/gaetancollaud/digitalstrom-mqtt/digitalstrom/api"
)

type EventHandler func(DigitalStromClient, api.Event)

// ClientOptions contains configurable options for a DigitalStrom Client.
type ClientOptions struct {
	Host                string
	Port                int
	Username            string
	Password            string
	MaxRetries          int
	RetryDuration       time.Duration
	EventSubscriptionId int
	RunEventLoop        bool
	EventRequestTimeout time.Duration
	OnEventHandler      EventHandler
}

// NewClientOptions will create a new ClientClientOptions type with some
// default values.
//   Host: dss.local
//   Port: 8080
//   MaxRetries: 3
// 	 RetryDuration: 2 seconds
//	 EventSubscriptionId: (randomly generated)
//	 RunEventLoop: true
//	 EventRequestTimeout: 10 seconds
func NewClientOptions() *ClientOptions {
	// Random generate subscriptionId in order to not have collisions of
	// multiple instances running at the same time.
	rand.Seed(time.Now().UnixNano())

	return &ClientOptions{
		Host:                "dss.local",
		Port:                8080,
		Username:            "",
		Password:            "",
		MaxRetries:          3,
		RetryDuration:       2 * time.Second,
		EventSubscriptionId: int(rand.Int31n(1 << 20)),
		RunEventLoop:        true,
		EventRequestTimeout: 10 * time.Second,
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
func (o *ClientOptions) SetUsername(u string) *ClientOptions {
	o.Username = u
	return o
}

// SetPassword will set the password to be used by this client when connecting
// to the DigitalStrom server.
func (o *ClientOptions) SetPassword(p string) *ClientOptions {
	o.Password = p
	return o
}

// SetMaxRetries will set the number of retries the client will perform when
// login errors are found. This applies to all API calls.
func (o *ClientOptions) SetMaxRetries(maxRetries int) *ClientOptions {
	o.MaxRetries = maxRetries
	return o
}

// SetRetryDuration will set the time the client will wait between retries.
func (o *ClientOptions) SetRetryDuration(duration time.Duration) *ClientOptions {
	o.RetryDuration = duration
	return o
}

// SetEventSubscriptionId will define the ID used to identify the subscription
// to the server when receiving events. Make sure the ID is unique between
// instances that are subscribing to events on the same DigitalStrom server.
func (o *ClientOptions) SetEventSubscriptionId(id int) *ClientOptions {
	o.EventSubscriptionId = id
	return o
}

// SetRunEventLoop will define whether a event loop is triggered to listen to
// all new events coming from the DigitalStrom server.
func (o *ClientOptions) SetRunEventLoop(b bool) *ClientOptions {
	o.RunEventLoop = b
	return o
}

// SetEventRequestTimeout will set the timeout for the get event requests.
func (o *ClientOptions) SetEventRequestTimeout(timeout time.Duration) *ClientOptions {
	o.EventRequestTimeout = timeout
	return o
}

// SetOnEventHandler will set the function that will be called when a new event
// is received in the event loop. On new events, each event handler will be
// called on a new goroutine and therefore must be safe for concurrency.
func (o *ClientOptions) SetOnEventHandler(handler EventHandler) *ClientOptions {
	o.OnEventHandler = handler
	return o
}
