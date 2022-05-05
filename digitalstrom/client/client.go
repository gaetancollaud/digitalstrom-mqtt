package client

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gaetancollaud/digitalstrom-mqtt/utils"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
)

// void type used as value for maps so they behave as sets.
type void struct{}

// done is an instance of the void type.
var done void

const (
	disconnected uint32 = 0
	connecting   uint32 = 1
	connected    uint32 = 2
)

// DigitalStromClient is the interface definition as used by this library, the
// interface is primarly to allow mocking tests.
type DigitalStromClient interface {
	// Connect will perform login on the DigitalStrom server.
	Connect() error
	// Disconnect from the server by closing all idle connections, closing the
	// event loop if running and unsubscribing from the server.
	Disconnect() error

	// Start of the API calls to DigitalStrom.

	// Get the list of circuits in the apartment.
	ApartmentGetCircuits() (*ApartmentGetCircuitsResponse, error)
	// Get the list of devices in the apartment.
	ApartmentGetDevices() (*ApartmentGetDevicesResponse, error)
	// Get the power consumption from a given circuit.
	CircuitGetConsumption(dsid string) (*CircuitGetConsumptionResponse, error)
	// Get the energy meter value from a given circuit.
	CircuitGetEnergyMeterValue(dsid string) (*CircuitGetEnergyMeterValueResponse, error)
	// Get the values for the channels in the given device.
	DeviceGetOutputChannelValue(dsid string, channels []string) (*DeviceGetOutputChannelValueResponse, error)
	// Sets the values for the channels in the given device.
	DeviceSetOutputChannelValue(dsid string, channelValues map[string]int) error
	// Get the latest event from the server. Note that you must be subscribed to
	// at least one event and the call is blocking until a new event is
	// available. This can be used when has been specified that the event loop
	// does not run and therefore is responsibility of the client to retrieve
	// the events manually using this call.
	EventGet() (*EventGetResponse, error)
	// Get the floating value for the given property path.
	PropertyGetFloating(path string) (*FloatValue, error)
	// Call action in a specified zone.
	ZoneCallAction(zoneId int, action Action) error
	// Get the zone name.
	ZoneGetName(zoneId int) (*ZoneGetNameResponse, error)
	// Get the list of scenes that are available at a given zone.
	ZoneGetReachableScenes(zoneId int, groupId int) (*ZoneGetReachableScenesResponse, error)
	// Get the scene name.
	ZoneSceneGetName(zoneId int, groupId int, sceneId int) (*ZoneSceneGetNameResponse, error)
}

// client implements the DigitalStrom interface.
// Clients are safe for concurrent use by multiple goroutines.
type client struct {
	status uint32

	httpClient *http.Client
	options    ClientOptions
	token      string

	eventLoopDone chan void
}

// NewClient will create a DigitalStrom client with all the options specified in
// the provided ClientOptions. The client must have the Connect() method called
// on it before it may be used.
func NewClient(options *ClientOptions) DigitalStromClient {
	return &client{
		status: disconnected,
		httpClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
		options: *options,
		token:   "",
	}
}

// Connect retrieves the token from the server by performing the login call.
func (c *client) Connect() error {
	if c.status == connected {
		// Already connected to the server.
		return nil
	}
	c.status = connecting
	if _, err := c.getToken(); err != nil {
		return err
	}

	// Subscribe to events.
	for _, event := range c.options.EventsToSubscribe {
		if err := c.eventSubscribe(event); err != nil {
			return fmt.Errorf("error unsubscribing from event '%s': %w", event, err)
		}
	}
	c.startEventLoop()
	c.status = connected
	return nil
}

// Disconnect stops all work on the  It stops any running event loop,
// unsubscribe from any event in the server and closes any idle connection.
func (c *client) Disconnect() error {
	if c.status == disconnected {
		// Already disconnected.
		return nil
	}
	c.stopEventLoop()

	// Unsubscribe from events.
	for _, event := range c.options.EventsToSubscribe {
		if err := c.eventUnsubscribe(event); err != nil {
			return fmt.Errorf("error unsubscribing from event '%s': %w", event, err)
		}
	}

	c.token = "" // Reset token.

	// Close all current connections.
	c.httpClient.CloseIdleConnections()

	c.status = disconnected
	return nil
}

func (c *client) ApartmentGetCircuits() (*ApartmentGetCircuitsResponse, error) {
	response, err := c.apiCall("json/apartment/getCircuits", url.Values{})
	return wrapApiResponse[ApartmentGetCircuitsResponse](response, err)
}

func (c *client) ApartmentGetDevices() (*ApartmentGetDevicesResponse, error) {
	response, err := c.apiCall("json/apartment/getDevices", url.Values{})
	return wrapApiResponse[ApartmentGetDevicesResponse](response, err)
}

func (c *client) CircuitGetConsumption(dsid string) (*CircuitGetConsumptionResponse, error) {
	params := url.Values{}
	params.Set("id", dsid)
	response, err := c.apiCall("json/circuit/getConsumption", params)
	return wrapApiResponse[CircuitGetConsumptionResponse](response, err)
}

func (c *client) CircuitGetEnergyMeterValue(dsid string) (*CircuitGetEnergyMeterValueResponse, error) {
	params := url.Values{}
	params.Set("id", dsid)
	response, err := c.apiCall("json/circuit/getEnergyMeterValue", params)
	return wrapApiResponse[CircuitGetEnergyMeterValueResponse](response, err)
}

func (c *client) PropertyGetFloating(path string) (*FloatValue, error) {
	params := url.Values{}
	params.Set("path", path)
	response, err := c.apiCall("json/property/getFloating", params)
	return wrapApiResponse[FloatValue](response, err)
}

func (c *client) ZoneGetName(zoneId int) (*ZoneGetNameResponse, error) {
	params := url.Values{}
	params.Set("id", strconv.Itoa(zoneId))
	response, err := c.apiCall("json/zone/getName", params)
	return wrapApiResponse[ZoneGetNameResponse](response, err)
}

func (c *client) ZoneCallAction(zoneId int, action Action) error {
	params := url.Values{}
	params.Set("application", "2")
	params.Set("id", strconv.Itoa(zoneId))
	params.Set("action", string(action))
	_, err := c.apiCall("json/zone/callAction", params)
	return err
}

func (c *client) ZoneSceneGetName(zoneId int, groupId int, sceneId int) (*ZoneSceneGetNameResponse, error) {
	params := url.Values{}
	params.Set("id", strconv.Itoa(zoneId))
	params.Set("groupID", strconv.Itoa(groupId))
	params.Set("sceneNumber", strconv.Itoa(sceneId))
	response, err := c.apiCall("json/zone/sceneGetName", params)
	return wrapApiResponse[ZoneSceneGetNameResponse](response, err)
}

func (c *client) ZoneGetReachableScenes(zoneId int, groupId int) (*ZoneGetReachableScenesResponse, error) {
	params := url.Values{}
	params.Set("id", strconv.Itoa(zoneId))
	params.Set("groupID", strconv.Itoa(groupId))
	response, err := c.apiCall("json/zone/getReachableScenes", params)
	return wrapApiResponse[ZoneGetReachableScenesResponse](response, err)
}

func (c *client) DeviceSetOutputChannelValue(dsid string, channelValues map[string]int) error {
	params := url.Values{}
	params.Set("dsid", dsid)
	var channelValuesParam []string
	for channel, value := range channelValues {
		channelValuesParam = append(channelValuesParam, channel+"="+strconv.Itoa(value))
	}
	params.Set("channelvalues", strings.Join(channelValuesParam, ";"))
	params.Set("applyNow", "1")
	_, err := c.apiCall("json/device/setOutputChannelValue", params)
	return err
}

func (c *client) DeviceGetOutputChannelValue(dsid string, channels []string) (*DeviceGetOutputChannelValueResponse, error) {
	params := url.Values{}
	params.Set("dsid", dsid)
	params.Set("channels", strings.Join(channels, ";"))
	response, err := c.apiCall("json/device/getOutputChannelValue", params)
	return wrapApiResponse[DeviceGetOutputChannelValueResponse](response, err)
}

func (c *client) eventSubscribe(event EventType) error {
	params := url.Values{}
	params.Set("name", string(event))
	params.Set("subscriptionID", strconv.Itoa(c.options.EventSubscriptionId))
	_, err := c.apiCall("json/event/subscribe", params)
	return err
}

func (c *client) eventUnsubscribe(event EventType) error {
	params := url.Values{}
	params.Set("name", string(event))
	params.Set("subscriptionID", strconv.Itoa(c.options.EventSubscriptionId))
	_, err := c.apiCall("json/event/unsubscribe", params)
	return err
}

func (c *client) EventGet() (*EventGetResponse, error) {
	params := url.Values{}
	params.Set("subscriptionID", strconv.Itoa(c.options.EventSubscriptionId))
	params.Set("timeout", strconv.Itoa(int(c.options.EventRequestTimeout.Milliseconds())))
	response, err := c.apiCall("json/event/get", params)
	return wrapApiResponse[EventGetResponse](response, err)
}

// getToken will retrieve the token of the current connection into the server.
// If already login, it will return the current connection token. Alternatively,
// if the token has been invalidated (e.g. expired), it will do login again and
// subscribe again to all the events the client was previously subscribed to.
func (c *client) getToken() (string, error) {
	if c.token != "" {
		return c.token, nil
	}
	// Get token by making login to the server.
	params := url.Values{}
	params.Set("user", c.options.Username)
	params.Set("password", c.options.Password)
	response, err := c.getRequest("json/system/login", params)
	res, err := wrapApiResponse[TokenResponse](response, err)
	if err != nil {
		return "", fmt.Errorf("error on login request: %w", err)
	}
	c.token = res.Token

	// Subscribe again to the events if there was an existing subscription
	// before. This should only happen when the token was revoked and we had to
	// reconnect to the server.
	for _, event := range c.options.EventsToSubscribe {
		if err := c.eventSubscribe(event); err != nil {
			return "", fmt.Errorf("error subscribing again to event '%s': %w", event, err)
		}
	}
	return c.token, nil
}

// apiCall performs a request to the DigitalStrom server by using retry and
// automatically populating the token on the request.
func (c *client) apiCall(path string, params url.Values) (interface{}, error) {
	var token string
	var err error
	var response interface{}

	for i := 0; i < c.options.MaxRetries; i++ {
		token, err = c.getToken()
		if err != nil {
			// In case of error retrieving token, wait some time and continue to
			// next retry.
			log.Warn().Err(err).Msg("Failed to retrieve tokenn. Will wait for next retry.")
			time.Sleep(c.options.RetryDuration)
			continue
		}
		params.Set("token", token)
		response, err = c.getRequest(path, params)
		if err == nil {
			break
		}
		if strings.Contains(err.Error(), "not logged in") {
			// Issue with token, invalidate the old one before retrying.
			c.token = "" // Invalidate current token.
			log.Warn().Err(err).Msg("Not logged error. Retrying...")
		} else {
			// Don't retry in case its not an authetication error.
			break
		}
		// This is a retry, wait a bit before we retry to avoid loops.
		time.Sleep(c.options.RetryDuration)
	}
	if err != nil {
		log.Error().Err(err).Msg("Failed API GET request")
		return nil, fmt.Errorf("unable to refresh token after "+strconv.Itoa(c.options.MaxRetries)+" retries: %w", err)
	}
	return response, nil
}

// getRequest performs a GET request to the DigitalStrom server given the path
// and parameters. It will parse the returned message to identify errors in the
// request and return a generic interface that corresponds to the `result` item
// in the response.
func (c *client) getRequest(path string, params url.Values) (interface{}, error) {
	// If Client is not connected refuse to make the request.
	if c.status == disconnected {
		return nil, fmt.Errorf("error performing request: client disconnected")
	}
	url := "https://" + c.options.Host +
		":" + strconv.Itoa(c.options.Port) +
		"/" + path +
		"?" + params.Encode()

	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error building the request: %w", err)
	}
	resp, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error doing the request: %w", err)
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("error reading the request: %w", err)
	}

	log.Debug().
		Str("url", url).
		Str("status", resp.Status).
		Msg("Response received")
	log.Trace().
		Str("body", string(body)).
		Msg("Response body")

	var jsonResponse map[string]interface{}
	json.Unmarshal(body, &jsonResponse)

	if val, ok := jsonResponse["ok"]; ok {
		if !val.(bool) {
			return nil, errors.New("error with DigitalStrom API: " + jsonResponse["message"].(string))
		}
	} else {
		log.Panic().Str("response", string(body)).Msg("No 'ok' field present in API response.")
		return nil, errors.New("no 'ok' field present, cannot check request")
	}

	if val, ok := jsonResponse["result"]; ok {
		return val, nil
	}
	return nil, nil
}

// Starts the event loop that will watch for new events in the DigitalStrom
// server and call the user provided callback when new events are received.
func (c *client) startEventLoop() {
	if !c.options.RunEventLoop || len(c.options.EventsToSubscribe) == 0 {
		return
	}

	c.eventLoopDone = make(chan void)

	go func() {
		log.Info().Msg("Starting event loop.")
		for {
			select {
			case <-c.eventLoopDone:
				return
			default:
				response, err := c.EventGet()
				if err != nil {
					log.Error().Err(err).Msg("Error getting the event.")
					time.Sleep(1 * time.Second)
					continue
				}
				for _, event := range response.Events {
					log.Debug().
						Str("event", utils.PrettyPrint(event)).
						Msg("Event received.")
					// Spawn a new goroutine handling the received event.
					go c.options.OnEventHandler(c, event)
				}
			}
		}
	}()
}

// stopEventLoop signals the event loop to stop and waits until any work on the
// event loop is done. The waiting time can be control using the
// EventRequestTimeout in the ClientOptions as the get requests to get the next
// event are blocking and will not return until the timeout is hit.
func (c *client) stopEventLoop() {
	if !c.options.RunEventLoop || len(c.options.EventsToSubscribe) == 0 {
		return
	}
	log.Info().Msg("Stopping event loop. Waiting for remaining event requests...")
	// Send signal to terminate the event loop.
	// Waits until the event loop is actually stopped which comes determined by
	// the timeout of the event get request.
	c.eventLoopDone <- done

	// Closing all channels.
	close(c.eventLoopDone)
	log.Info().Msg("Event loop stopped.")
}

// wrapApiResponse takes a generic response interface and maps it to the given
// structure. This is used to decode the responses from the apiCall responses
// into explicit structs.
func wrapApiResponse[T any](response interface{}, err error) (*T, error) {
	// Handle original error coming from the response.
	if err != nil {
		return nil, err
	}

	// Decode the response into the given struct type.
	res := new(T)
	config := &mapstructure.DecoderConfig{
		Metadata:         nil,
		Result:           res,
		WeaklyTypedInput: true,
		ErrorUnset:       true,
	}
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return nil, fmt.Errorf("error building decored: %w", err)
	}
	if err = decoder.Decode(response); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}
	return res, nil
}
