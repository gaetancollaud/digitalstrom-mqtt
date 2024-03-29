package digitalstrom

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const WEBSOCKET_RECONNECT_DELAY = 5 * time.Second

type NotificationCallback func(notification WebsocketNotification)

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
}

// client implements the DigitalStrom interface.
// Clients are safe for concurrent use by multiple goroutines.
type client struct {
	httpClient              *http.Client
	options                 ClientOptions
	websocketConnection     *websocket.Conn
	websocketConnectionOpen bool

	notificationCallbacks map[string]NotificationCallback
}

// NewClient will create a DigitalStrom client with all the options specified in
// the provided ClientOptions. The client must have the Connect() method called
// on it before it may be used.
func NewClient(options *ClientOptions) Client {
	return &client{
		httpClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
		options:               *options,
		notificationCallbacks: map[string]NotificationCallback{},
	}
}

func (c *client) websocketConnect() error {
	websocketHost := "ws://" + c.options.Host + ":8090/api/v1/apartment/notifications"
	log.Trace().Str("host", websocketHost).Msg("Connecting to websocket")
	headers := http.Header{}
	headers.Add("Authorization", "Bearer "+c.options.ApiKey)
	ws, _, err := websocket.DefaultDialer.Dial(websocketHost, headers)
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
	log.Info().Msg("Connected to websocket for notifications")
	c.websocketConnectionOpen = true
	return nil
}

func (c *client) Connect() error {
	c.websocketConnectionOpen = false
	err := c.websocketConnect()
	if err != nil {
		return err
	}

	// websocket message reader
	go func() {
		firstMessage := true
		for {
			var notification WebsocketNotification
			err := c.websocketConnection.ReadJSON(&notification)
			if err != nil {
				if !c.websocketConnectionOpen {
					// we're closing, ignore read errors
					break
				} else {
					log.Error().Err(err).Msg("Websocket reading error, will try to reconnect")
					time.Sleep(WEBSOCKET_RECONNECT_DELAY)
					err = c.websocketConnect()
					if err != nil {
						log.Error().Err(err).Msg("Websocket reconnect error")
					}
				}
			} else if notification.Arguments == nil || len(notification.Arguments) == 0 {
				if !firstMessage {
					log.Warn().Msg("No argument received in notification")
				}
			} else {
				for _, callback := range c.notificationCallbacks {
					callback(notification)
				}
				log.Trace().Str("target", notification.Target).Str("type", string(notification.Arguments[0].Type)).Msg("Websocket received")
			}
			firstMessage = false
		}
		log.Info().Msg("Closing websocket reader")
	}()

	return nil
}

// Disconnect stops all the ongoing calls and unsubscribe from the notification websocket
func (c *client) Disconnect() error {
	c.websocketConnectionOpen = false
	c.httpClient.CloseIdleConnections()
	_ = c.websocketConnection.Close()
	return nil
}

func (c *client) GetApartment() (*Apartment, error) {
	params := url.Values{}
	params.Set("include", "installation,dsDevices,submodules,functionBlocks,zones,controllers,meterings")
	response, err := c.getRequest("api/v1/apartment", params)
	return wrapApiResponse[Apartment](response, err)
}

func (c *client) GetApartmentStatus() (*ApartmentStatus, error) {
	params := url.Values{}
	params.Set("include", "dsDevices,zones")
	response, err := c.getRequest("api/v1/apartment/status", params)
	return wrapApiResponse[ApartmentStatus](response, err)
}

func (c *client) GetMeterings() (*Meterings, error) {
	response, err := c.getRequest("api/v1/apartment/meterings", nil)
	return wrapApiResponse[Meterings](response, err)
}

func (c *client) GetMeteringStatus() (*MeteringValues, error) {
	response, err := c.getRequest("api/v1/apartment/meterings/values", nil)
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
	if err != nil {
		return nil, fmt.Errorf("error doing the request: %w", err)
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}

	responseBody, readErr := io.ReadAll(resp.Body)
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

func (c *client) getRequest(path string, params url.Values) (interface{}, error) {
	body, err := c.doRequest(http.MethodGet, path, params, nil)
	if err != nil {
		return nil, err
	}

	var jsonResponse map[string]interface{}
	err = json.Unmarshal(body, &jsonResponse)
	if err != nil {
		return nil, fmt.Errorf("error parsing response for path %s: %w", path, err)
	}

	if data, ok := jsonResponse["data"]; ok {
		return data, nil
	} else {
		// TODO maybe handle error
		log.Panic().Str("response", string(body)).Msg("no 'data' field present, cannot get data from request")
		return nil, errors.New("no 'data' field present, cannot get data from request")
	}
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
