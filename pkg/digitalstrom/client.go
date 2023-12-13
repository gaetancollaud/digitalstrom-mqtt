package digitalstrom

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/utils"
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

type ApiVersion uint32

const (
	apiClassic   ApiVersion = 1
	apiSmarthome ApiVersion = 2
)

type NotificationCallback func(notification WebsocketNotification)

// Deprecated: use new API instead
type EventCallback func(Client, Event) error

// Client is the interface definition as used by this library, the
// interface is primarily to allow mocking tests.
type Client interface {
	// Connect will perform login on the DigitalStrom server.
	Connect() error
	// Disconnect from the server by closing all idle connections, closing the
	// event loop if running and unsubscribing from the server.
	Disconnect() error

	// Start of the API calls to DigitalStrom.

	GetApartment() (*Apartment, error)
	GetApartmentStatus() (*ApartmentStatus, error)
	GetMeterings() (*Meterings, error)
	GetMeteringStatus() (*MeteringValues, error)

	// DeviceSetOutputValue Sets a list of outputs to a give values
	DeviceSetOutputValue(deviceId string, functionBlockId string, outputId string, value float64) error

	NotificationSubscribe(id string, callback NotificationCallback) error
	NotificationUnsubscribe(id string) error

	// Deprecated: use new API instead
	// DeviceSetOutputChannelValue Sets the values for the channels in the given device.
	DeviceSetOutputChannelValue(dsid string, channelValues map[string]int) error

	// Deprecated: use new API instead
	// Call a scene which will be immediately applied.
	ApartmentCallScene(sceneId int) error
	// Deprecated: use new API instead
	// Get the list of Zones and the groups on it.
	ApartmentGetReachableGroups() (*ApartmentGetReachableGroupsResponse, error)
	// Deprecated: use new API instead
	// Get the values for the channels in the given device.
	DeviceGetOutputChannelValue(dsid string, channels []string) (*DeviceGetOutputChannelValueResponse, error)
	// Deprecated: use new API instead
	// Gets the motion time for the device.
	DeviceGetMaxMotionTime(dsid string) (*DeviceGetMaxMotionTimeResponse, error)
	// Deprecated: use new API instead
	// Subscribe to an event and run the given callback when an event of the
	// given types is received.
	EventSubscribe(event EventType, eventCallback EventCallback) error
	// Deprecated: use new API instead
	// Unsubscribe to the given event type.
	EventUnsubscribe(event EventType) error
	// Deprecated: use new API instead
	// Get the latest event from the server. Note that you must be subscribed to
	// at least one event and the call is blocking until a new event is
	// available. This can be used when has been specified that the event loop
	// does not run and therefore is responsibility of the client to retrieve
	// the events manually using this call.
	EventGet() (*EventGetResponse, error)
	// Deprecated: use new API instead
	// Get the floating value for the given property path.
	PropertyGetFloating(path string) (*FloatValue, error)
	// Deprecated: use new API instead
	// Call scene in a specified zone.
	ZoneCallScene(zone string, groupId int, sceneId int) error
	// Deprecated: use new API instead
	// Call action in a specified zone.
	ZoneCallAction(zone string, action Action) error
	// Deprecated: use new API instead
	// Get the zone name.
	ZoneGetName(zoneId int) (*ZoneGetNameResponse, error)
	// Deprecated: use new API instead
	// Get the list of scenes that are available at a given zone.
	ZoneGetReachableScenes(zoneId int, groupId int) (*ZoneGetReachableScenesResponse, error)
	// Deprecated: use new API instead
	// Get the scene name.
	ZoneSceneGetName(zoneId int, groupId int, sceneId int) (*ZoneSceneGetNameResponse, error)
}

// client implements the DigitalStrom interface.
// Clients are safe for concurrent use by multiple goroutines.
type client struct {
	status uint32

	httpClient          *http.Client
	options             ClientOptions
	websocketConnection *websocket.Conn
	token               string

	notificationCallbacks map[string]NotificationCallback

	eventsSubscribedCallbacks map[EventType][]EventCallback
	eventLoopDone             chan void
	eventMutex                sync.Mutex

	decoder *mapstructure.Decoder

	// Protect the login process with a Mutex to avoid multiple goroutines
	// performing login in parallel and not have in sync the subscriptions for
	// each session.
	loginMutex sync.Mutex
}

// NewClient will create a DigitalStrom client with all the options specified in
// the provided ClientOptions. The client must have the Connect() method called
// on it before it may be used.
func NewClient(options *ClientOptions) Client {
	return &client{
		status: disconnected,
		httpClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
		options:                   *options,
		token:                     "",
		eventsSubscribedCallbacks: map[EventType][]EventCallback{},
		notificationCallbacks:     map[string]NotificationCallback{},
	}
}

// Connect retrieves the token from the server by performing the login call.
func (c *client) Connect() error {
	if c.status == connected {
		// Already connected to the server.
		return nil
	}
	c.status = connecting

	websocketHost := "ws://" + c.options.Host + ":8090/api/v1/apartment/notifications"
	log.Trace().Str("host", websocketHost).Msg("Connecting to websocket")
	ws, _, err := websocket.DefaultDialer.Dial(websocketHost, nil)
	if err != nil {
		return fmt.Errorf("unable to connecting to notification websocket: %w", err)
	}
	c.websocketConnection = ws
	// initiate event stream
	err = c.websocketConnection.WriteJSON(WebsocketInitMessage{
		Protocol: "json",
		Version:  1,
	})
	if err != nil {
		return fmt.Errorf("error writing to websocket: %w", err)
	}

	go func() {
		firstMessage := true
		for {
			var notif WebsocketNotification
			err := c.websocketConnection.ReadJSON(&notif)
			if err != nil {
				log.Error().Err(err).Msg("Websocket reading error")
				break
			} else if notif.Arguments == nil || len(notif.Arguments) == 0 {
				if !firstMessage {
					log.Warn().Msg("No argument received in notification")
				}
			} else {
				for _, callback := range c.notificationCallbacks {
					callback(notif)
				}
				log.Trace().Str("target", notif.Target).Str("type", string(notif.Arguments[0].Type)).Msg("Websocket received")
			}
			firstMessage = false
		}
		log.Warn().Msg("Closing websocket reader")
	}()

	//c.startEventLoop()
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
	for event, _ := range c.eventsSubscribedCallbacks {
		if err := c.EventUnsubscribe(event); err != nil {
			return fmt.Errorf("error unsubscribing from event '%s': %w", event, err)
		}
	}

	c.token = "" // Reset token.

	// Close all current connections.
	c.httpClient.CloseIdleConnections()
	c.websocketConnection.Close()

	c.status = disconnected
	return nil
}

func (c *client) GetApartment() (*Apartment, error) {
	response, err := c.apiCall("api/v1/apartment", url.Values{}, apiSmarthome)
	return wrapApiResponse[Apartment](response, err)
}

func (c *client) GetApartmentStatus() (*ApartmentStatus, error) {
	response, err := c.apiCall("api/v1/apartment/status", url.Values{}, apiSmarthome)
	return wrapApiResponse[ApartmentStatus](response, err)
}

func (c *client) GetMeterings() (*Meterings, error) {
	response, err := c.apiCall("api/v1/apartment/meterings", url.Values{}, apiSmarthome)
	return wrapApiResponse[Meterings](response, err)
}

func (c *client) GetMeteringStatus() (*MeteringValues, error) {
	response, err := c.apiCall("api/v1/apartment/meterings/values", url.Values{}, apiSmarthome)
	return wrapApiResponse[MeteringValues](response, err)
}

func (c *client) DeviceSetOutputValue(deviceId string, functionBlockId string, outputId string, value float64) error {
	var contents []SetOutputValue
	contents = append(contents, SetOutputValue{
		Op:    SetOutputValueOperationReplace,
		Path:  fmt.Sprintf("/functionBlocks/%s/outputs/%s/value", functionBlockId, outputId),
		Value: fmt.Sprintf("%.0f", value),
	})

	path := fmt.Sprintf("api/v1/apartment/dsDevices/%s/status", deviceId)
	return c.patchRequest(path, contents)
}

func (c *client) NotificationSubscribe(id string, callback NotificationCallback) error {
	_, exists := c.notificationCallbacks[id]
	if exists {
		return errors.New("Notification callback with id " + id + " already exists")
	}
	c.notificationCallbacks[id] = callback
	return nil
}

func (c *client) NotificationUnsubscribe(id string) error {
	_, exists := c.notificationCallbacks[id]
	if !exists {
		return errors.New("Notification callback with id " + id + " does not exist")
	}
	delete(c.notificationCallbacks, id)
	return nil
}

func (c *client) ApartmentCallScene(sceneId int) error {
	params := url.Values{}
	params.Set("sceneNumber", strconv.Itoa(sceneId))
	_, err := c.apiCall("json/apartment/callScene", params, apiClassic)
	return err
}

func (c *client) ApartmentGetReachableGroups() (*ApartmentGetReachableGroupsResponse, error) {
	params := url.Values{}
	response, err := c.apiCall("json/apartment/getReachableGroups", params, apiClassic)
	return wrapApiResponse[ApartmentGetReachableGroupsResponse](response, err)
}

// Deprecated: use new API instead
func (c *client) PropertyGetFloating(path string) (*FloatValue, error) {
	params := url.Values{}
	params.Set("path", path)
	response, err := c.apiCall("json/property/getFloating", params, apiClassic)
	return wrapApiResponse[FloatValue](response, err)
}

func (c *client) ZoneCallScene(zone string, groupId int, sceneId int) error {
	params := url.Values{}
	params.Set("id", zone)
	params.Set("sceneNumber", strconv.Itoa(sceneId))
	params.Set("groupID", strconv.Itoa(groupId))
	params.Set("force", "true")
	_, err := c.apiCall("json/zone/callScene", params, apiClassic)
	return err
}

func (c *client) ZoneGetName(zoneId int) (*ZoneGetNameResponse, error) {
	params := url.Values{}
	params.Set("id", strconv.Itoa(zoneId))
	response, err := c.apiCall("json/zone/getName", params, apiClassic)
	return wrapApiResponse[ZoneGetNameResponse](response, err)
}

func (c *client) ZoneCallAction(zone string, action Action) error {
	params := url.Values{}
	params.Set("application", "2")
	params.Set("id", zone)
	params.Set("action", string(action))
	_, err := c.apiCall("json/zone/callAction", params, apiClassic)
	return err
}

func (c *client) ZoneSceneGetName(zoneId int, groupId int, sceneId int) (*ZoneSceneGetNameResponse, error) {
	params := url.Values{}
	params.Set("id", strconv.Itoa(zoneId))
	params.Set("groupID", strconv.Itoa(groupId))
	params.Set("sceneNumber", strconv.Itoa(sceneId))
	response, err := c.apiCall("json/zone/sceneGetName", params, apiClassic)
	return wrapApiResponse[ZoneSceneGetNameResponse](response, err)
}

func (c *client) ZoneGetReachableScenes(zoneId int, groupId int) (*ZoneGetReachableScenesResponse, error) {
	params := url.Values{}
	params.Set("id", strconv.Itoa(zoneId))
	params.Set("groupID", strconv.Itoa(groupId))
	response, err := c.apiCall("json/zone/getReachableScenes", params, apiClassic)
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
	_, err := c.apiCall("json/device/setOutputChannelValue", params, apiClassic)
	return err
}

func (c *client) DeviceGetOutputChannelValue(dsid string, channels []string) (*DeviceGetOutputChannelValueResponse, error) {
	params := url.Values{}
	params.Set("dsid", dsid)
	params.Set("channels", strings.Join(channels, ";"))
	response, err := c.apiCall("json/device/getOutputChannelValue", params, apiClassic)
	return wrapApiResponse[DeviceGetOutputChannelValueResponse](response, err)
}

func (c *client) DeviceGetMaxMotionTime(dsid string) (*DeviceGetMaxMotionTimeResponse, error) {
	params := url.Values{}
	params.Set("dsid", dsid)
	response, err := c.apiCall("json/device/getMaxMotionTime", params, apiClassic)
	return wrapApiResponse[DeviceGetMaxMotionTimeResponse](response, err)
}

// Deprecated: use new API instead
func (c *client) EventSubscribe(event EventType, eventCallback EventCallback) error {
	c.eventMutex.Lock()
	defer c.eventMutex.Unlock()

	params := url.Values{}
	params.Set("name", string(event))
	params.Set("subscriptionID", strconv.Itoa(c.options.EventSubscriptionId))
	_, err := c.apiCall("json/event/subscribe", params, apiClassic)
	if err != nil {
		return err
	}

	if _, ok := c.eventsSubscribedCallbacks[event]; !ok {
		c.eventsSubscribedCallbacks[event] = []EventCallback{}
	}
	c.eventsSubscribedCallbacks[event] = append(c.eventsSubscribedCallbacks[event], eventCallback)
	return nil
}

// Deprecated: use new API instead
func (c *client) EventUnsubscribe(event EventType) error {
	c.eventMutex.Lock()
	defer c.eventMutex.Unlock()

	params := url.Values{}
	params.Set("name", string(event))
	params.Set("subscriptionID", strconv.Itoa(c.options.EventSubscriptionId))
	_, err := c.apiCall("json/event/unsubscribe", params, apiClassic)
	if err != nil {
		return err
	}
	delete(c.eventsSubscribedCallbacks, event)
	return nil
}

// Deprecated: use new API instead
func (c *client) EventGet() (*EventGetResponse, error) {
	params := url.Values{}
	params.Set("subscriptionID", strconv.Itoa(c.options.EventSubscriptionId))
	params.Set("timeout", strconv.Itoa(int(c.options.EventRequestTimeout.Milliseconds())))
	response, err := c.apiCall("json/event/get", params, apiClassic)
	return wrapApiResponse[EventGetResponse](response, err)
}

// Deprecated: use new API instead
// getToken will retrieve the token of the current connection into the server.
// If already login, it will return the current connection token. Alternatively,
// if the token has been invalidated (e.g. expired), it will do login again and
// subscribe again to all the events the client was previously subscribed to.
func (c *client) getToken() (string, error) {
	c.loginMutex.Lock()
	defer c.loginMutex.Unlock()

	if c.token != "" {
		return c.token, nil
	}
	// Get token by making login to the server.
	params := url.Values{}
	//params.Set("user", c.options.Username)
	//params.Set("password", c.options.Password)
	response, err := c.getRequest("json/system/login", params, apiClassic)
	res, err := wrapApiResponse[TokenResponse](response, err)
	if err != nil {
		return "", fmt.Errorf("error on login request: %w", err)
	}
	c.token = res.Token

	// Subscribe again to the events if there was an existing subscription
	// before. This should only happen when the token was revoked and we had to
	// reconnect to the server.
	eventsSubscribed := c.eventsSubscribedCallbacks
	c.eventsSubscribedCallbacks = map[EventType][]EventCallback{}
	for event, callbacks := range eventsSubscribed {
		for _, callback := range callbacks {
			if err := c.EventSubscribe(event, callback); err != nil {
				return "", fmt.Errorf("error subscribing again to event '%s': %w", event, err)
			}
		}
	}
	return c.token, nil
}

// Deprecated: use getRequest instead
// apiCall performs a request to the DigitalStrom server by using retry and
// automatically populating the token on the request.
func (c *client) apiCall(path string, params url.Values, version ApiVersion) (interface{}, error) {
	return c.getRequest(path, params, version)

	//
	////var token string
	//var err error
	//var response interface{}
	//
	//for i := 0; i < c.options.MaxRetries; i++ {
	//	//token, err = c.getToken()
	//	if err != nil {
	//		// In case of error retrieving token, wait some time and continue to
	//		// next retry.
	//		log.Warn().Err(err).Msg("Failed to retrieve token. Will wait for next retry.")
	//		time.Sleep(c.options.RetryDuration)
	//		continue
	//	}
	//	//params.Set("token", token)
	//	response, err = c.getRequest(path, params, version)
	//	if err == nil {
	//		break
	//	}
	//	if strings.Contains(err.Error(), "not logged in") {
	//		// Issue with token, invalidate the old one before retrying.
	//		c.token = "" // Invalidate current token.
	//		log.Warn().Err(err).Msg("Not logged error. Retrying...")
	//	} else {
	//		// Don't retry in case its not an authetication error.
	//		break
	//	}
	//	// This is a retry, wait a bit before we retry to avoid loops.
	//	time.Sleep(c.options.RetryDuration)
	//}
	//if err != nil {
	//	log.Error().Err(err).Msg("Failed API GET request")
	//	return nil, fmt.Errorf("unable to refresh token after "+strconv.Itoa(c.options.MaxRetries)+" retries: %w", err)
	//}
	//return response, nil
}

func (c *client) doRequest(method string, path string, params url.Values, body interface{}) ([]byte, error) {
	var bodyReader io.Reader = nil
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = strings.NewReader(string(jsonBody))
	}
	callUrl := "https://" + c.options.Host + ":" + strconv.Itoa(c.options.Port) + "/" + path
	if params != nil && len(params) > 0 {
		callUrl = callUrl + "?" + params.Encode()
	}

	request, err := http.NewRequest(method, callUrl, bodyReader)
	request.Header.Set("Authorization", "Bearer "+c.options.ApiKey)
	if err != nil {
		return nil, fmt.Errorf("error building the request: %w", err)
	}
	resp, err := c.httpClient.Do(request)
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("error doing the request: %w", err)
	}

	responseBody, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("error reading the request: %w", err)
	}

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("error response from server, httpStatus=%d: %s", resp.StatusCode, responseBody)
	}

	log.Debug().
		Str("url", callUrl).
		Str("status", resp.Status).
		Msg("Response received")
	log.Trace().
		Str("body", string(responseBody)).
		Msg("Response body")

	return responseBody, nil
}

func (c *client) patchRequest(path string, body interface{}) error {
	_, err := c.doRequest(http.MethodPatch, path, nil, body)
	return err
}

// getRequest performs a GET request to the DigitalStrom server given the path
// and parameters. It will parse the returned message to identify errors in the
// request and return a generic interface that corresponds to the `result` item
// in the response.
func (c *client) getRequest(path string, params url.Values, version ApiVersion) (interface{}, error) {
	body, err := c.doRequest(http.MethodGet, path, params, nil)
	if err != nil {
		return nil, err
	}

	var jsonResponse map[string]interface{}
	err = json.Unmarshal(body, &jsonResponse)
	if err != nil {
		return nil, fmt.Errorf("error parsing response for path %s: %w", path, err)
	}

	if version == apiClassic {
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
	} else if version == apiSmarthome {
		if data, ok := jsonResponse["data"]; ok {
			return data, nil
		} else {
			// TODO maybe handle error
			log.Panic().Str("response", string(body)).Msg("no 'data' field present, cannot get data from request")
			return nil, errors.New("no 'data' field present, cannot get data from request")
		}
	} else {
		log.Panic().Uint32("Version", uint32(version)).Msg("Unknown API version")
	}
	return nil, nil
}

// Deprecated: use new API instead
// Starts the event loop that will watch for new events in the DigitalStrom
// server and call the user provided callback when new events are received.
func (c *client) startEventLoop() {
	if !c.options.RunEventLoop {
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
				// In case there is no subscription to any event, in order to
				// avoid an error in the GET request, let's put to sleep the
				// loop.
				if len(c.eventsSubscribedCallbacks) == 0 {
					time.Sleep(1 * time.Second)
					continue
				}

				response, err := c.EventGet()
				if err != nil {
					log.Error().Err(err).Msg("Error getting the event.")
					time.Sleep(1 * time.Second)
					continue
				}
				// For each event received, spawn a goroutine executing its
				// callback.
				for _, event := range response.Events {
					log.Debug().
						Str("event", utils.PrettyPrint(event)).
						Msg("Event received.")

					callbacks, ok := c.eventsSubscribedCallbacks[event.Name]
					if !ok {
						log.Warn().
							Str("event type", string(event.Name)).
							Str("even", utils.PrettyPrint(event)).
							Msg("Received an event that does not have any callback registered.")
						continue
					}
					for _, callback := range callbacks {
						go callback(c, event)
					}
				}
			}
		}
	}()
}

// Deprecated: use new API instead
// stopEventLoop signals the event loop to stop and waits until any work on the
// event loop is done. The waiting time can be control using the
// EventRequestTimeout in the ClientOptions as the get requests to get the next
// event are blocking and will not return until the timeout is hit.
func (c *client) stopEventLoop() {
	if !c.options.RunEventLoop {
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
		ErrorUnset:       false,
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
