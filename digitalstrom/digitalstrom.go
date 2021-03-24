package digitalstrom

import (
	"github.com/gaetancollaud/digitalstrom-mqtt/config"
	"github.com/rs/zerolog/log"
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
	ds.httpClient = NewHttpClient(&config.DigitalStrom)
	ds.eventsManager = NewDigitalstromEvents(ds.httpClient)
	ds.devicesManager = NewDevicesManager(ds.httpClient)
	ds.circuitManager = NewCircuitManager(ds.httpClient)
	return ds
}

func (ds *DigitalStrom) Start() {
	log.Info().Msg("Staring digitalstrom")
	ds.cron.ticker = time.NewTicker(30 * time.Second)
	ds.cron.tickerDone = make(chan bool)
	go ds.digitalstromCron()

	ds.eventsManager.Start()
	ds.circuitManager.Start()
	ds.devicesManager.Start()

	go ds.circuitManager.UpdateCircuitsValue()

	go ds.updateDevicesOnEvent(ds.eventsManager.events)

	if ds.config.RefreshAtStart {
		go ds.refreshAllDevices()
	}
}

func (ds *DigitalStrom) Stop() {
	log.Info().Msg("Stopping digitalstrom")
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
		case <-ds.cron.ticker.C:
			log.Info().Msg("Updating circuits values")
			ds.circuitManager.UpdateCircuitsValue()
		}
	}
}

func (ds *DigitalStrom) updateDevicesOnEvent(events chan Event) {
	for event := range events {
		log.Info().Msg("Event received, updating devices")
		ds.devicesManager.updateZone(event.ZoneId)

		time.AfterFunc(2*time.Second, func() {
			// update again because maybe the three was not up to date yet
			ds.devicesManager.updateZone(event.ZoneId)
		})
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

func (ds *DigitalStrom) refreshAllDevices() {
	log.Info().
		Int("size", len(ds.devicesManager.devices)).
		Msg("Refreshing all devices")
	for _, device := range ds.devicesManager.devices {
		ds.devicesManager.updateDevice(device)
	}
}
