package digitalstrom

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
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

func (httpClient *HttpClient) get(path string) (*DigitalStromResponse, error) {
	for i := 1; i <= MAX_RETRIES; i++ {
		token := httpClient.TokenManager.GetToken()
		if !strings.Contains(path, "?") {
			path = path + "?token=" + token
		} else {
			path = path + "&token=" + token
		}
		response, err := httpClient.getWithoutToken(path)
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
