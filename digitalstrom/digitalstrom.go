package digitalstrom

import (
	"fmt"
	"github.com/gaetancollaud/digitalstrom-mqtt/config"
	"time"
)

type DigitalStrom struct {
	config         *config.Config
	cron           DigitalstromCron
	httpClient     *HttpClient
	eventsManager  *EventsManager
	devicesManager *DevicesManager
	circuitManager *CircuitsManager
}

type DigitalstromCron struct {
	ticker     *time.Ticker
	tickerDone chan bool
}

func New(config *config.Config) *DigitalStrom {
	ds := new(DigitalStrom)
	ds.config = config
	ds.httpClient = NewHttpClient(config)
	ds.eventsManager = NewDigitalstromEvents(ds.httpClient)
	ds.devicesManager = NewDevicesManager(ds.httpClient)
	ds.circuitManager = NewCircuitManager(ds.httpClient)
	return ds
}

func (ds *DigitalStrom) Start() {
	fmt.Println("Staring digitalstrom")
	ds.cron.ticker = time.NewTicker(30 * time.Second)
	ds.cron.tickerDone = make(chan bool)
	go ds.digitalstromCron()

	ds.eventsManager.Start()
	ds.circuitManager.Start()
	ds.devicesManager.Start()

	go ds.circuitManager.UpdateCircuitsValue()

	go ds.updateDevicesOnEvent(ds.eventsManager.events)
}

func (ds *DigitalStrom) Stop() {
	fmt.Println("Stopping digitalstrom")
	if ds.cron.ticker != nil {
		ds.cron.ticker.Stop()
		ds.cron.tickerDone <- true
		ds.cron.ticker = nil
	}
	ds.eventsManager.Stop()
}

func (ds *DigitalStrom) digitalstromCron() {
	for {
		select {
		case <-ds.cron.tickerDone:
			return
		case t := <-ds.cron.ticker.C:
			fmt.Println("Digitalstrom cron", t)
			ds.circuitManager.UpdateCircuitsValue()
		}
	}
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

func (ds *DigitalStrom) GetCircuitChangeChannel() chan CircuitValueChanged {
	return ds.circuitManager.circuitValuesChan
}

func (ds *DigitalStrom) SetDeviceValue(command DeviceCommand) error {
	return ds.devicesManager.SetValue(command)
}
