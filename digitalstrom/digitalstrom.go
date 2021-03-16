package digitalstrom

import (
	"fmt"
	"github.com/gaetancollaud/digitalstrom-mqtt/config"
	"time"
)

type DigitalStrom struct {
	config     *config.Config
	KeepAlive  KeepAlive
	httpClient *HttpClient
}

type KeepAlive struct {
	ticker     *time.Ticker
	tickerDone chan bool
}

func New(config *config.Config) *DigitalStrom {
	ds := new(DigitalStrom)
	ds.config = config
	ds.httpClient = NewHttpClient(config)
	return ds
}

func (ds *DigitalStrom) Start() {
	fmt.Println("Staring keep alive")
	ds.KeepAlive.ticker = time.NewTicker(30 * time.Second)
	ds.KeepAlive.tickerDone = make(chan bool)
	go ds.digitalstromKeepAlive()
	user := ds.getLoggedInUser()
	fmt.Println("Checking user", user)
}

func (ds *DigitalStrom) Stop() {
	fmt.Println("Stopping keep alive")
	if ds.KeepAlive.ticker != nil {
		ds.KeepAlive.ticker.Stop()
		ds.KeepAlive.ticker = nil
	}
}

func (ds *DigitalStrom) digitalstromKeepAlive() {
	for {
		select {
		case <-ds.KeepAlive.tickerDone:
			return
		case t := <-ds.KeepAlive.ticker.C:
			user := ds.getLoggedInUser()
			fmt.Println("Checking user", user, t)
		}
	}
}

func (ds *DigitalStrom) getLoggedInUser() string {
	get, err := ds.httpClient.get("json/system/loggedInUser")
	if checkNoError(err) {
		if len(get) == 0 {
			fmt.Errorf("No user logged in")
		} else {
			return get["name"].(string)
		}
	}
	return ""

}
