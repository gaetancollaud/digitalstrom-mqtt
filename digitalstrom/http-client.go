package digitalstrom

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gaetancollaud/digitalstrom-mqtt/config"
	"io/ioutil"
	"net/http"
	"strconv"
)

type HttpClient struct {
	config *config.Config
}

type DigitalStromWrapper struct{

}

func NewUrlBuilder(config *config.Config) *HttpClient {
	u := new(HttpClient)
	u.config = config
	return u
}

func (ub *HttpClient) get(path string, a ...interface{}) (map[string]interface{}, error) {
	url := "https://" + ub.config.Ip + ":" + strconv.Itoa(ub.config.Port) + "/" + fmt.Sprintf(path, a...)
	fmt.Printf("URL: %s\n", url)

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

	fmt.Printf("Status: %s\n", resp.Status)

	var jsonValue map[string]interface{}
	json.Unmarshal(body, &jsonValue)

	if !jsonValue["ok"].(bool) {
		return nil, errors.New("Reponse ko")
	}

	return jsonValue["result"].(map[string]interface{}), nil
}
