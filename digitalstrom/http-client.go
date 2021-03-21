package digitalstrom

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"github.com/gaetancollaud/digitalstrom-mqtt/config"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

type HttpClient struct {
	config       *config.Config
	TokenManager *TokenManager
}

type DigitalStromResponse struct {
	isMap      bool
	mapValue   map[string]interface{}
	isArray    bool
	arrayValue []interface{}
}

func NewHttpClient(config *config.Config) *HttpClient {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	httpClient := new(HttpClient)
	httpClient.config = config
	httpClient.TokenManager = NewTokenManager(config, httpClient)
	return httpClient
}

func (httpClient *HttpClient) get(path string) (*DigitalStromResponse, error) {

	token := httpClient.TokenManager.GetToken()

	if strings.Index(path, "?") == -1 {
		path = path + "?token=" + token
	} else {
		path = path + "&token=" + token
	}
	return httpClient.getWithoutToken(path)
}

func (httpClient *HttpClient) getWithoutToken(path string) (*DigitalStromResponse, error) {
	url := "https://" + httpClient.config.Ip + ":" + strconv.Itoa(httpClient.config.Port) + "/" + path

	resp, err := http.Get(url)
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

	//fmt.Printf("%s status: %s\n", url, resp.Status)

	var jsonValue map[string]interface{}
	json.Unmarshal(body, &jsonValue)

	if val, ok := jsonValue["ok"]; ok {
		if !val.(bool) {
			return nil, errors.New("Error with digitalstrom API: " + jsonValue["message"].(string))
		}
	} else {
		return nil, errors.New("No 'ok' field present, cannot check request")
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
		return nil, errors.New("Unknown return type")
	}
	// no value returned
	return nil, nil
}
