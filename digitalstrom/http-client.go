package digitalstrom

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

	"github.com/gaetancollaud/digitalstrom-mqtt/config"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
)

const MAX_RETRIES = 3

type HttpClient struct {
	client       *http.Client
	config       *config.ConfigDigitalstrom
	TokenManager *TokenManager
}

func NewHttpClient(config *config.ConfigDigitalstrom) *HttpClient {
	httpClient := new(HttpClient)
	httpClient.client = &http.Client{
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
	resp, err := httpClient.client.Do(request)
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

// Methods for the specific endpoints called in DigitalStrom API.

type Action string

const (
	MoveUp        Action = "app.moveUp"
	MoveDown      Action = "app.moveDown"
	StepUp        Action = "app.stepUp"
	StepDown      Action = "app.stepDown"
	SunProtection Action = "app.sunProtection"
	Stop          Action = "app.stop"
)

type ChannelType string

const (
	Brightness ChannelType = "brightness"
	Hue        ChannelType = "hue"
)

func (httpClient *HttpClient) SystemLogin(user string, password string) (*TokenResponse, error) {
	params := url.Values{}
	params.Set("user", user)
	params.Set("password", password)
	response, err := httpClient.getRequest("json/system/login", params)
	return wrapApiResponse[TokenResponse](response, err)
}

func (httpClient *HttpClient) ApartmentGetCircuits() (*ApartmentGetCircuitsResponse, error) {
	response, err := httpClient.getRequestWithToken("json/apartment/getCircuits", url.Values{})
	return wrapApiResponse[ApartmentGetCircuitsResponse](response, err)
}

func (httpClient *HttpClient) ApartmentGetDevices() (*ApartmentGetDevicesResponse, error) {
	response, err := httpClient.getRequestWithToken("json/apartment/getDevices", url.Values{})
	return wrapApiResponse[ApartmentGetDevicesResponse](response, err)
}

func (httpClient *HttpClient) CircuitGetConsumption(dsid string) (*CircuitGetConsumptionResponse, error) {
	params := url.Values{}
	params.Set("id", dsid)
	response, err := httpClient.getRequestWithToken("json/circuit/getConsumption", params)
	return wrapApiResponse[CircuitGetConsumptionResponse](response, err)
}

func (httpClient *HttpClient) CircuitGetEnergyMeterValue(dsid string) (*CircuitGetEnergyMeterValueResponse, error) {
	params := url.Values{}
	params.Set("id", dsid)
	response, err := httpClient.getRequestWithToken("json/circuit/getEnergyMeterValue", params)
	return wrapApiResponse[CircuitGetEnergyMeterValueResponse](response, err)
}

func (httpClient *HttpClient) PropertyGetFloating(path string) (*FloatValue, error) {
	params := url.Values{}
	params.Set("path", path)
	response, err := httpClient.getRequestWithToken("json/property/getFloating", params)
	return wrapApiResponse[FloatValue](response, err)
}

func (httpClient *HttpClient) ZoneGetName(zoneId int) (*ZoneGetNameResponse, error) {
	params := url.Values{}
	params.Set("id", strconv.Itoa(zoneId))
	response, err := httpClient.getRequestWithToken("json/zone/getName", params)
	return wrapApiResponse[ZoneGetNameResponse](response, err)
}

func (httpClient *HttpClient) ZoneCallAction(zoneId int, action Action) error {
	params := url.Values{}
	params.Set("application", "2")
	params.Set("id", strconv.Itoa(zoneId))
	params.Set("action", string(action))
	_, err := httpClient.getRequestWithToken("json/zone/callAction", params)
	return err
}

func (httpClient *HttpClient) ZoneSceneGetName(zoneId int, groupId int, sceneId int) (*ZoneSceneGetNameResponse, error) {
	params := url.Values{}
	params.Set("id", strconv.Itoa(zoneId))
	params.Set("groupID", strconv.Itoa(groupId))
	params.Set("sceneNumber", strconv.Itoa(sceneId))
	response, err := httpClient.getRequestWithToken("json/zone/sceneGetName", params)
	return wrapApiResponse[ZoneSceneGetNameResponse](response, err)
}

func (httpClient *HttpClient) ZoneGetReachableScenes(zoneId int, groupId int) (*ZoneGetReachableScenesResponse, error) {
	params := url.Values{}
	params.Set("id", strconv.Itoa(zoneId))
	params.Set("groupID", strconv.Itoa(groupId))
	response, err := httpClient.getRequestWithToken("json/zone/getReachableScenes", params)
	return wrapApiResponse[ZoneGetReachableScenesResponse](response, err)
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

func (httpClient *HttpClient) DeviceGetOutputChannelValue(dsid string, channels []string) (*DeviceGetOutputChannelValueResponse, error) {
	params := url.Values{}
	params.Set("dsid", dsid)
	params.Set("channels", strings.Join(channels, ";"))
	response, err := httpClient.getRequestWithToken("json/device/getOutputChannelValue", params)
	return wrapApiResponse[DeviceGetOutputChannelValueResponse](response, err)
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

func (httpClient *HttpClient) EventGet(subscriptionId int) (*EventGetResponse, error) {
	params := url.Values{}
	params.Set("subscriptionID", strconv.Itoa(subscriptionId))
	response, err := httpClient.getRequestWithToken("json/event/get", params)
	return wrapApiResponse[EventGetResponse](response, err)
}
