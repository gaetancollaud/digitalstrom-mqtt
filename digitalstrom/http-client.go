package digitalstrom

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gaetancollaud/digitalstrom-mqtt/config"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
)

type HttpClient struct {
	config       *config.Config
	TokenManager *TokenManager
}

func NewHttpClient(config *config.Config) *HttpClient {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	httpClient := new(HttpClient)
	httpClient.config = config
	httpClient.TokenManager = NewTokenManager(config, httpClient)
	return httpClient
}

func (httpClient *HttpClient) get(path string, a ...interface{}) (map[string]interface{}, error) {

	token := httpClient.TokenManager.GetToken()

	url, err := url.Parse(path)

	if checkNoError(err) {
		query := url.Query()
		query.Set("token", token)
		url.RawQuery = query.Encode()
		return httpClient.getWithoutToken(url.String(), a...)
	}
	return nil, err
}

func (httpClient *HttpClient) getWithoutToken(path string, a ...interface{}) (map[string]interface{}, error) {
	url := "https://" + httpClient.config.Ip + ":" + strconv.Itoa(httpClient.config.Port) + "/" + fmt.Sprintf(path, a...)
	fmt.Printf("Calling URL: %s\n", url)

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

	fmt.Printf("%s status: %s\n", url, resp.Status)

	var jsonValue map[string]interface{}
	json.Unmarshal(body, &jsonValue)

	if !jsonValue["ok"].(bool) {
		return nil, errors.New("Error with digitalstrom API: " + jsonValue["message"].(string))
	}

	return jsonValue["result"].(map[string]interface{}), nil
}
