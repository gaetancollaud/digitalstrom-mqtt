package digitalstrom

import (
	"fmt"
	"github.com/gaetancollaud/digitalstrom-mqtt/config"
	"time"
)

type DigitalStrom struct {
	config         *config.Config
	KeepAlive      KeepAlive
	httpClient     *HttpClient
	eventsManager  *EventsManager
	devicesManager *DevicesManager
}

// TODO move keep alive in dedicated class
type KeepAlive struct {
	ticker     *time.Ticker
	tickerDone chan bool
}

func New(config *config.Config) *DigitalStrom {
	ds := new(DigitalStrom)
	ds.config = config
	ds.httpClient = NewHttpClient(config)
	ds.eventsManager = NewDigitalstromEvents(ds.httpClient)
	ds.devicesManager = NewDevicesManager(ds.httpClient)
	return ds
}

func (ds *DigitalStrom) Start() {
	fmt.Println("Staring digitalstrom")
	ds.KeepAlive.ticker = time.NewTicker(30 * time.Second)
	ds.KeepAlive.tickerDone = make(chan bool)
	go ds.digitalstromKeepAlive()
	user := ds.getLoggedInUser()
	fmt.Println("Checking user", user)
	ds.eventsManager.Start()
	ds.devicesManager.Start()

	go ds.updateDevicesOnEvent(ds.eventsManager.events)
}

func (ds *DigitalStrom) Stop() {
	fmt.Println("Stopping digitalstrom")
	if ds.KeepAlive.ticker != nil {
		ds.KeepAlive.ticker.Stop()
		ds.KeepAlive.tickerDone <- true
		ds.KeepAlive.ticker = nil
	}
	ds.eventsManager.Stop()
}

func (ds *DigitalStrom) digitalstromKeepAlive() {
	for {
		select {
		case <-ds.KeepAlive.tickerDone:
			return
		case t := <-ds.KeepAlive.ticker.C:
			user := ds.getLoggedInUser()
			fmt.Println("Keep alive, user", user, t)
		}
	}
}

func (ds *DigitalStrom) getLoggedInUser() string {
	response, err := ds.httpClient.get("json/system/loggedInUser")
	if checkNoError(err) {
		if !response.isMap || len(response.mapValue) == 0 {
			fmt.Errorf("No user logged in")
		} else {
			return response.mapValue["name"].(string)
		}
	}
	return ""
}

func (ds *DigitalStrom) updateDevicesOnEvent(events chan Event) {
	for event := range events {
		fmt.Println("Event received, updating devices")
		ds.devicesManager.updateZone(event.ZoneId)
	}
}

func (ds *DigitalStrom) GetDeviceChangeChannel() chan DeviceStatusChanged {
	return ds.devicesManager.deviceStatusChan
}
