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
	"sync"
	"time"

	"github.com/gaetancollaud/digitalstrom-mqtt/config"
	"github.com/gaetancollaud/digitalstrom-mqtt/digitalstrom/api"
	"github.com/gaetancollaud/digitalstrom-mqtt/utils"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
)

// void type used as value for maps so they behave as sets.
type void struct{}

// member is an instance of the void type.
var member void

// done is an instance of the void type.
var done void

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
	ApartmentGetCircuits() (*api.ApartmentGetCircuitsResponse, error)
	// Get the list of devices in the apartment.
	ApartmentGetDevices() (*api.ApartmentGetDevicesResponse, error)
	// Get the power consumption from a given circuit.
	CircuitGetConsumption(dsid string) (*api.CircuitGetConsumptionResponse, error)
	// Get the energy meter value from a given circuit.
	CircuitGetEnergyMeterValue(dsid string) (*api.CircuitGetEnergyMeterValueResponse, error)
	// Get the values for the channels in the given device.
	DeviceGetOutputChannelValue(dsid string, channels []string) (*api.DeviceGetOutputChannelValueResponse, error)
	// Sets the values for the channels in the given device.
	DeviceSetOutputChannelValue(dsid string, channelValues map[string]int) error
	// Get the latest event from the server. Note that you must be subscribed to
	// at least one event and the call is blocking until a new event is
	// available.
	EventGet() (*api.EventGetResponse, error)
	// Subscribe to an event type.
	EventSubscribe(event string) error
	// Unsubscribe to an event type.
	EventUnsubscribe(event string) error
	// Get the floating value for the given property path.
	PropertyGetFloating(path string) (*api.FloatValue, error)
	// Call action in a specified zone.
	ZoneCallAction(zoneId int, action api.Action) error
	// Get the zone name.
	ZoneGetName(zoneId int) (*api.ZoneGetNameResponse, error)
	// Get the list of scenes that are available at a given zone.
	ZoneGetReachableScenes(zoneId int, groupId int) (*api.ZoneGetReachableScenesResponse, error)
	// Get the scene name.
	ZoneSceneGetName(zoneId int, groupId int, sceneId int) (*api.ZoneSceneGetNameResponse, error)
}

// client implements the DigitalStrom interface.
// Clients are safe for concurrent use by multiple goroutines.
type client struct {
	httpClient *http.Client
	options    ClientOptions
	token      string

	eventsSubscribed map[string]void
	eventMutex       sync.Mutex
	eventLoopDone    chan void
	eventLoop        sync.WaitGroup
}

// NewClient will create a DigitalStrom client with all the options specified in
// the provided ClientOptions. The client must have the Connect() method called
// on it before it may be used.
func NewClient(options *ClientOptions) DigitalStromClient {
	return &client{
		httpClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
		options:          *options,
		token:            "",
		eventsSubscribed: map[string]void{},
	}
}

// Connect retrieves the token from the server by performing the login call.
func (c *client) Connect() error {
	if _, err := c.getToken(); err != nil {
		return err
	}

	return nil
}

// Disconnect stops all work on the client. It stops any running event loop,
// unsubscribe from any event in the server and closes any idle connection.
func (c *client) Disconnect() error {
	c.stopEventLoop()

	// Unsubscribe from events.
	for event := range c.eventsSubscribed {
		if err := c.EventUnsubscribe(event); err != nil {
			return fmt.Errorf("error unsubscribing from event '%s': %w", event, err)
		}
	}

	c.token = "" // Reset token.

	// Close all current connections.
	c.httpClient.CloseIdleConnections()
	return nil
}

func (c *client) ApartmentGetCircuits() (*api.ApartmentGetCircuitsResponse, error) {
	response, err := c.apiCall("json/apartment/getCircuits", url.Values{})
	return wrapApiResponse[api.ApartmentGetCircuitsResponse](response, err)
}

func (c *client) ApartmentGetDevices() (*api.ApartmentGetDevicesResponse, error) {
	response, err := c.apiCall("json/apartment/getDevices", url.Values{})
	return wrapApiResponse[api.ApartmentGetDevicesResponse](response, err)
}

func (c *client) CircuitGetConsumption(dsid string) (*api.CircuitGetConsumptionResponse, error) {
	params := url.Values{}
	params.Set("id", dsid)
	response, err := c.apiCall("json/circuit/getConsumption", params)
	return wrapApiResponse[api.CircuitGetConsumptionResponse](response, err)
}

func (c *client) CircuitGetEnergyMeterValue(dsid string) (*api.CircuitGetEnergyMeterValueResponse, error) {
	params := url.Values{}
	params.Set("id", dsid)
	response, err := c.apiCall("json/circuit/getEnergyMeterValue", params)
	return wrapApiResponse[api.CircuitGetEnergyMeterValueResponse](response, err)
}

func (c *client) PropertyGetFloating(path string) (*api.FloatValue, error) {
	params := url.Values{}
	params.Set("path", path)
	response, err := c.apiCall("json/property/getFloating", params)
	return wrapApiResponse[api.FloatValue](response, err)
}

func (c *client) ZoneGetName(zoneId int) (*api.ZoneGetNameResponse, error) {
	params := url.Values{}
	params.Set("id", strconv.Itoa(zoneId))
	response, err := c.apiCall("json/zone/getName", params)
	return wrapApiResponse[api.ZoneGetNameResponse](response, err)
}

func (c *client) ZoneCallAction(zoneId int, action api.Action) error {
	params := url.Values{}
	params.Set("application", "2")
	params.Set("id", strconv.Itoa(zoneId))
	params.Set("action", string(action))
	_, err := c.apiCall("json/zone/callAction", params)
	return err
}

func (c *client) ZoneSceneGetName(zoneId int, groupId int, sceneId int) (*api.ZoneSceneGetNameResponse, error) {
	params := url.Values{}
	params.Set("id", strconv.Itoa(zoneId))
	params.Set("groupID", strconv.Itoa(groupId))
	params.Set("sceneNumber", strconv.Itoa(sceneId))
	response, err := c.apiCall("json/zone/sceneGetName", params)
	return wrapApiResponse[api.ZoneSceneGetNameResponse](response, err)
}

func (c *client) ZoneGetReachableScenes(zoneId int, groupId int) (*api.ZoneGetReachableScenesResponse, error) {
	params := url.Values{}
	params.Set("id", strconv.Itoa(zoneId))
	params.Set("groupID", strconv.Itoa(groupId))
	response, err := c.apiCall("json/zone/getReachableScenes", params)
	return wrapApiResponse[api.ZoneGetReachableScenesResponse](response, err)
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

func (c *client) DeviceGetOutputChannelValue(dsid string, channels []string) (*api.DeviceGetOutputChannelValueResponse, error) {
	params := url.Values{}
	params.Set("dsid", dsid)
	params.Set("channels", strings.Join(channels, ";"))
	response, err := c.apiCall("json/device/getOutputChannelValue", params)
	return wrapApiResponse[api.DeviceGetOutputChannelValueResponse](response, err)
}

func (c *client) EventSubscribe(event string) error {
	c.eventMutex.Lock()
	defer c.eventMutex.Unlock()

	if _, ok := c.eventsSubscribed[event]; ok {
		// Event already subscribed.
		return nil
	}
	params := url.Values{}
	params.Set("name", event)
	params.Set("subscriptionID", strconv.Itoa(c.options.EventSubscriptionId))
	_, err := c.apiCall("json/event/subscribe", params)
	if err != nil {
		return err
	}

	// Handle the registration of the event to the event loop.
	c.eventsSubscribed[event] = member
	if len(c.eventsSubscribed) == 1 {
		// Just added the first event and therefore let's start the event loop.
		c.startEventLoop()
	}

	return nil
}

func (c *client) EventUnsubscribe(event string) error {
	c.eventMutex.Lock()
	defer c.eventMutex.Unlock()

	if _, ok := c.eventsSubscribed[event]; !ok {
		return fmt.Errorf("error when unsubscribing from event '%s': not subscribed", event)
	}

	params := url.Values{}
	params.Set("name", event)
	params.Set("subscriptionID", strconv.Itoa(c.options.EventSubscriptionId))
	_, err := c.apiCall("json/event/unsubscribe", params)
	if err != nil {
		return err
	}

	// Handle the unregistration of the event to the client and the event loop.
	delete(c.eventsSubscribed, event)
	if len(c.eventsSubscribed) == 0 {
		// Just removed the last event, let's stop the event loop.
		c.stopEventLoop()
	}
	return nil
}

func (c *client) EventGet() (*api.EventGetResponse, error) {
	params := url.Values{}
	params.Set("subscriptionID", strconv.Itoa(c.options.EventSubscriptionId))
	params.Set("timeout", strconv.Itoa(int(c.options.EventRequestTimeout.Seconds())))
	response, err := c.apiCall("json/event/get", params)
	return wrapApiResponse[api.EventGetResponse](response, err)
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
	res, err := wrapApiResponse[api.TokenResponse](response, err)
	if err != nil {
		return "", fmt.Errorf("error on login request: %w", err)
	}
	c.token = res.Token

	// Subscribe again to the events if there was an existing subscription
	// before. This should only happen when the token was revoked and we had to
	// reconnect to the server.
	c.eventMutex.Lock()
	for event := range c.eventsSubscribed {
		if err := c.EventSubscribe(event); err != nil {
			return "", fmt.Errorf("error subscribing again to event '%s': %w", event, err)
		}
	}
	c.eventMutex.Unlock()
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
	if !c.options.RunEventLoop {
		return
	}

	c.eventLoop.Add(1)
	c.eventLoopDone = make(chan void)

	go func() {
		log.Info().Msg("Starting event loop.")
		defer c.eventLoop.Done()
		for {
			select {
			case <-c.eventLoopDone:
				log.Info().Msg("Stopping event loop.")
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
	if !c.options.RunEventLoop {
		return
	}
	// Send signal to terminate the event loop.
	c.eventLoopDone <- done
	// Wait until the event loop is actually stopped which comes determined by
	// the timeout of the event get request.
	c.eventLoop.Wait()

	// Closing all channels.
	close(c.eventLoopDone)
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

////////////////////////////////////////////////////////////////////////////////
const MAX_RETRIES = 3

type HttpClient struct {
	httpClient   *http.Client
	config       *config.ConfigDigitalstrom
	TokenManager *TokenManager
}

func NewHttpClient(config *config.ConfigDigitalstrom) *HttpClient {
	httpClient := new(HttpClient)
	httpClient.httpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	httpClient.config = config
	httpClient.TokenManager = NewTokenManager(config, httpClient)
	return httpClient
}

func (httpClient *HttpClient) getRequestWithToken(path string, params url.Values) (interface{}, error) {
	for i := 1; i <= MAX_RETRIES; i++ {
		token := httpClient.TokenManager.GetToken()
		params.Set("token", token)
		response, err := httpClient.getRequest(path, params)
		if err == nil {
			return response, err
		} else {
			log.Warn().Err(err).Msg("Failed GET request")
		}
		if strings.Contains(err.Error(), "not logged in") {
			// Issue with token, invalidate the old one before retrying.
			httpClient.TokenManager.InvalidateToken()
		} else {
			// Don't retry in case its not an authetication error.
			return nil, err
		}
		// This is a retry, wait a bit before we retry to avoid loops.
		time.Sleep(2 * time.Second)
	}
	return nil, errors.New("unable to refresh token after " + strconv.Itoa(MAX_RETRIES) + " retries")
}

func (httpClient *HttpClient) getRequest(path string, params url.Values) (interface{}, error) {
	url := "https://" + httpClient.config.Host +
		":" + strconv.Itoa(httpClient.config.Port) +
		"/" + path +
		"?" + params.Encode()

	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.httpClient.Do(request)
	if err != nil {
		return nil, err
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		return nil, err
	}

	log.Trace().
		Str("url", url).
		Str("status", resp.Status).
		Str("body", string(body)).
		Msg("Response received")

	var jsonValue map[string]interface{}
	json.Unmarshal(body, &jsonValue)

	if val, ok := jsonValue["ok"]; ok {
		if !val.(bool) {
			return nil, errors.New("error with DigitalStrom API: " + jsonValue["message"].(string))
		}
	} else {
		return nil, errors.New("no 'ok' field present, cannot check request")
	}

	if val, ok := jsonValue["result"]; ok {
		return val, nil
	}
	return nil, nil
}

// Methods for the specific endpoints called in DigitalStrom API.

func (httpClient *HttpClient) SystemLogin(user string, password string) (*api.TokenResponse, error) {
	params := url.Values{}
	params.Set("user", user)
	params.Set("password", password)
	response, err := httpClient.getRequest("json/system/login", params)
	return wrapApiResponse[api.TokenResponse](response, err)
}

func (httpClient *HttpClient) ApartmentGetCircuits() (*api.ApartmentGetCircuitsResponse, error) {
	response, err := httpClient.getRequestWithToken("json/apartment/getCircuits", url.Values{})
	return wrapApiResponse[api.ApartmentGetCircuitsResponse](response, err)
}

func (httpClient *HttpClient) ApartmentGetDevices() (*api.ApartmentGetDevicesResponse, error) {
	response, err := httpClient.getRequestWithToken("json/apartment/getDevices", url.Values{})
	return wrapApiResponse[api.ApartmentGetDevicesResponse](response, err)
}

func (httpClient *HttpClient) CircuitGetConsumption(dsid string) (*api.CircuitGetConsumptionResponse, error) {
	params := url.Values{}
	params.Set("id", dsid)
	response, err := httpClient.getRequestWithToken("json/circuit/getConsumption", params)
	return wrapApiResponse[api.CircuitGetConsumptionResponse](response, err)
}

func (httpClient *HttpClient) CircuitGetEnergyMeterValue(dsid string) (*api.CircuitGetEnergyMeterValueResponse, error) {
	params := url.Values{}
	params.Set("id", dsid)
	response, err := httpClient.getRequestWithToken("json/circuit/getEnergyMeterValue", params)
	return wrapApiResponse[api.CircuitGetEnergyMeterValueResponse](response, err)
}

func (httpClient *HttpClient) PropertyGetFloating(path string) (*api.FloatValue, error) {
	params := url.Values{}
	params.Set("path", path)
	response, err := httpClient.getRequestWithToken("json/property/getFloating", params)
	return wrapApiResponse[api.FloatValue](response, err)
}

func (httpClient *HttpClient) ZoneGetName(zoneId int) (*api.ZoneGetNameResponse, error) {
	params := url.Values{}
	params.Set("id", strconv.Itoa(zoneId))
	response, err := httpClient.getRequestWithToken("json/zone/getName", params)
	return wrapApiResponse[api.ZoneGetNameResponse](response, err)
}

func (httpClient *HttpClient) ZoneCallAction(zoneId int, action api.Action) error {
	params := url.Values{}
	params.Set("application", "2")
	params.Set("id", strconv.Itoa(zoneId))
	params.Set("action", string(action))
	_, err := httpClient.getRequestWithToken("json/zone/callAction", params)
	return err
}

func (httpClient *HttpClient) ZoneSceneGetName(zoneId int, groupId int, sceneId int) (*api.ZoneSceneGetNameResponse, error) {
	params := url.Values{}
	params.Set("id", strconv.Itoa(zoneId))
	params.Set("groupID", strconv.Itoa(groupId))
	params.Set("sceneNumber", strconv.Itoa(sceneId))
	response, err := httpClient.getRequestWithToken("json/zone/sceneGetName", params)
	return wrapApiResponse[api.ZoneSceneGetNameResponse](response, err)
}

func (httpClient *HttpClient) ZoneGetReachableScenes(zoneId int, groupId int) (*api.ZoneGetReachableScenesResponse, error) {
	params := url.Values{}
	params.Set("id", strconv.Itoa(zoneId))
	params.Set("groupID", strconv.Itoa(groupId))
	response, err := httpClient.getRequestWithToken("json/zone/getReachableScenes", params)
	return wrapApiResponse[api.ZoneGetReachableScenesResponse](response, err)
}

func (httpClient *HttpClient) DeviceSetOutputChannelValue(dsid string, channelValues map[string]int) error {
	params := url.Values{}
	params.Set("dsid", dsid)
	var channelValuesParam []string
	for channel, value := range channelValues {
		channelValuesParam = append(channelValuesParam, channel+"="+strconv.Itoa(value))
	}
	params.Set("channelvalues", strings.Join(channelValuesParam, ";"))
	params.Set("applyNow", "1")
	_, err := httpClient.getRequestWithToken("json/device/setOutputChannelValue", params)
	return err
}

func (httpClient *HttpClient) DeviceGetOutputChannelValue(dsid string, channels []string) (*api.DeviceGetOutputChannelValueResponse, error) {
	params := url.Values{}
	params.Set("dsid", dsid)
	params.Set("channels", strings.Join(channels, ";"))
	response, err := httpClient.getRequestWithToken("json/device/getOutputChannelValue", params)
	return wrapApiResponse[api.DeviceGetOutputChannelValueResponse](response, err)
}

func (httpClient *HttpClient) EventSubscribe(event string, subscriptionId int) error {
	params := url.Values{}
	params.Set("name", event)
	params.Set("subscriptionID", strconv.Itoa(subscriptionId))
	_, err := httpClient.getRequestWithToken("json/event/subscribe", params)
	return err
}

func (httpClient *HttpClient) EventUnsubscribe(event string, subscriptionId int) error {
	params := url.Values{}
	params.Set("name", event)
	params.Set("subscriptionID", strconv.Itoa(subscriptionId))
	_, err := httpClient.getRequestWithToken("json/event/unsubscribe", params)
	return err
}

func (httpClient *HttpClient) EventGet(subscriptionId int) (*api.EventGetResponse, error) {
	params := url.Values{}
	params.Set("subscriptionID", strconv.Itoa(subscriptionId))
	params.Set("timeout", "10000")
	response, err := httpClient.getRequestWithToken("json/event/get", params)
	return wrapApiResponse[api.EventGetResponse](response, err)
}
