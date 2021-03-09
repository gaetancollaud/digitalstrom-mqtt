package digitalstrom

import (
	"crypto/tls"
	"fmt"
	"github.com/gaetancollaud/digitalstrom-mqtt/config"
	"net/http"
)

type TokenManager struct {
	config     *config.Config
	httpClient *HttpClient
	token      string
}

type loginResponse struct {
}

func NewTokenManager(config *config.Config) *TokenManager {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	tm := new(TokenManager)

	tm.config = config
	tm.httpClient = NewUrlBuilder(config)

	return tm
}

func check(e error) {
	if e != nil {
		panic(fmt.Errorf("Error with token: %v\n", e))
	}
}

func (tm *TokenManager) RefreshToken() string {
	body, err := tm.httpClient.get("json/system/login?user=%s&password=%s", tm.config.Username, tm.config.Password)

	check(err)

	return body["token"].(string)
}

//https://192.168.1.50:8080/json/system/login?user=dssadmin&password=m7Phf1Dl2EIvlHUABBeI
