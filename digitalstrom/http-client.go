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
	"github.com/rs/zerolog/log"
)

const MAX_RETRIES = 3

type HttpClient struct {
	client       *http.Client
	config       *config.ConfigDigitalstrom
	TokenManager *TokenManager
}

type DigitalStromResponse struct {
	isMap      bool
	mapValue   map[string]interface{}
	isArray    bool
	arrayValue []interface{}
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

func (httpClient *HttpClient) get(path string, params url.Values) (*DigitalStromResponse, error) {
	for i := 1; i <= MAX_RETRIES; i++ {
		token := httpClient.TokenManager.GetToken()
		params.Set("token", token)
		response, err := httpClient.getWithoutToken(path + "?" + params.Encode())
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

func (httpClient *HttpClient) getWithoutToken(path string) (*DigitalStromResponse, error) {
	url := "https://" + httpClient.config.Host + ":" + strconv.Itoa(httpClient.config.Port) + "/" + path

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
		res := new(DigitalStromResponse)
		mapValue, ok := val.(map[string]interface{})
		if ok {
			res.isMap = true
			res.isArray = false
			res.mapValue = mapValue
			return res, nil
		}
		arrayValue, ok := val.([]interface{})
		if ok {
			res.isMap = false
			res.isArray = true
			res.arrayValue = arrayValue
			return res, nil
		}
		return nil, errors.New("unknown return type")
	}
	// no value returned
	return nil, nil
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

func (httpClient *HttpClient) ApartmentGetCircuits() (*DigitalStromResponse, error) {
	return httpClient.get("json/apartment/getCircuits", url.Values{})
}

func (httpClient *HttpClient) ApartmentGetDevices() (*DigitalStromResponse, error) {
	return httpClient.get("json/apartment/getDevices", url.Values{})
}

func (httpClient *HttpClient) CircuitGetConsumption(dsid string) (*DigitalStromResponse, error) {
	params := url.Values{}
	params.Set("id", dsid)
	return httpClient.get("json/circuit/getConsumption", params)
}

func (httpClient *HttpClient) CircuitGetEnergyMeterValue(dsid string) (*DigitalStromResponse, error) {
	params := url.Values{}
	params.Set("id", dsid)
	return httpClient.get("json/circuit/getEnergyMeterValue", params)
}

func (httpClient *HttpClient) PropertyGetFloating(path string) (float64, error) {
	params := url.Values{}
	params.Set("path", path)
	response, err := httpClient.get("json/property/getFloating", params)
	if err != nil {
		return 0, fmt.Errorf("error calling GetTreeFloat: %w", err)
	}
	return response.mapValue["value"].(float64), nil
}

func (httpClient *HttpClient) ZoneGetName(zoneId int) (*DigitalStromResponse, error) {
	params := url.Values{}
	params.Set("id", strconv.Itoa(zoneId))
	return httpClient.get("json/zone/getName", params)
}

func (httpClient *HttpClient) ZoneCallAction(zoneId int, action Action) (*DigitalStromResponse, error) {
	params := url.Values{}
	params.Set("application", "2")
	params.Set("id", strconv.Itoa(zoneId))
	params.Set("action", string(action))
	return httpClient.get("json/zone/callAction", params)
}

// func (httpClient *HttpClient) ZoneGetReachableScenes(zoneId int) (*DigitalStromResponse, error) {
// }

func (httpClient *HttpClient) ZoneSceneGetName(zoneId int, groupId int, sceneId int) (*DigitalStromResponse, error) {
	params := url.Values{}
	params.Set("id", strconv.Itoa(zoneId))
	params.Set("groupID", strconv.Itoa(groupId))
	params.Set("sceneNumber", strconv.Itoa(sceneId))
	return httpClient.get("json/zone/sceneGetName", params)
}

func (httpClient *HttpClient) DeviceSetOutputChannelValue(dsid string, channelValues map[string]int) (*DigitalStromResponse, error) {
	params := url.Values{}
	params.Set("dsid", dsid)
	var channelValuesParam []string
	for channel, value := range channelValues {
		channelValuesParam = append(channelValuesParam, channel+"="+strconv.Itoa(value))
	}
	params.Set("channelvalues", strings.Join(channelValuesParam, ";"))
	params.Set("applyNow", "1")
	return httpClient.get("json/device/setOutputChannelValue", params)
}

func (httpClient *HttpClient) DeviceGetOutputChannelValue(dsid string, channels []string) (*DigitalStromResponse, error) {
	params := url.Values{}
	params.Set("dsid", dsid)
	params.Set("channels", strings.Join(channels, ";"))
	return httpClient.get("json/device/getOutputChannelValue", params)
}

func (httpClient *HttpClient) EventSubscribe(event string, subscriptionId int) (*DigitalStromResponse, error) {
	params := url.Values{}
	params.Set("name", event)
	params.Set("subscriptionID", strconv.Itoa(subscriptionId))
	return httpClient.get("json/event/subscribe", params)
}

func (httpClient *HttpClient) EventUnsubscribe(event string, subscriptionId int) (*DigitalStromResponse, error) {
	params := url.Values{}
	params.Set("name", event)
	params.Set("subscriptionID", strconv.Itoa(subscriptionId))
	return httpClient.get("json/event/unsubscribe", params)
}

func (httpClient *HttpClient) EventGet(subscriptionId int) (*DigitalStromResponse, error) {
	params := url.Values{}
	params.Set("subscriptionID", strconv.Itoa(subscriptionId))
	return httpClient.get("json/event/get", params)
}
