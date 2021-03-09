package digitalstrom

import (
	"crypto/tls"
	"fmt"
	"github.com/gaetancollaud/digitalstrom-mqtt/config"
	"net/http"
	"strconv"
)

type TokenManager struct {
	config *config.Config
	token  string
}

func NewTokenManager(config *config.Config) *TokenManager {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	tm := new(TokenManager)
	tm.config = config
	return tm
}

func check(e error) {
	if e != nil {
		panic(fmt.Errorf("Error with token: %v\n", e))
	}
}

func (tm *TokenManager) RefreshToken() string {
	url := "https://" + tm.config.Ip + ":" + strconv.Itoa(tm.config.Port) + "/json/system/login?user=" + tm.config.Username + "&password=" + tm.config.Password

	fmt.Printf("URL: %s\n", url)
	resp, err := http.Get(url)
	check(err)

	return resp.Status
}

//https://192.168.1.50:8080/json/system/login?user=dssadmin&password=m7Phf1Dl2EIvlHUABBeI
