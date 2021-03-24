package digitalstrom

import (
	"github.com/gaetancollaud/digitalstrom-mqtt/config"
	"github.com/gaetancollaud/digitalstrom-mqtt/utils"
)

type TokenManager struct {
	config     *config.ConfigDigitalstrom
	httpClient *HttpClient
	token      string
}

func NewTokenManager(config *config.ConfigDigitalstrom, httpClient *HttpClient) *TokenManager {
	tm := new(TokenManager)
	tm.config = config
	tm.httpClient = httpClient

	return tm
}

func (tm *TokenManager) refreshToken() string {
	response, err := tm.httpClient.getWithoutToken("json/system/login?user=" + tm.config.Username + "&password=" + tm.config.Password)

	utils.CheckNoError(err)

	if response.isMap {
		return response.mapValue["token"].(string)
	}
	return ""
}

func (tm *TokenManager) GetToken() string {
	if tm.token == "" {
		tm.token = tm.refreshToken()
	}
	// TODO refresh after 50sec
	return tm.token
}

//https://192.168.1.50:8080/json/system/login?user=dssadmin&password=m7Phf1Dl2EIvlHUABBeI
