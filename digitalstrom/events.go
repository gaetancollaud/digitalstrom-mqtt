package digitalstrom

import (
	"fmt"
)

const SUBSCRIPTION_ID = "42"

// https://developer.digitalstrom.org/Architecture/system-interfaces.pdf#1e

const EVENT_CALL_SCENE = "callScene"
const EVENT_UNDO_SCENE = "undoScene"
const EVENT_BUTTON_CLICK = "buttonClick"
const EVENT_DEVICE_SENSOR_EVENT = "deviceSensorEvent"
const EVENT_RUNNING = "running"
const EVENT_MODEL_READY = "model_ready"
const EVENT_DSMETER_READY = "dsMeter_ready"

type EventsManager struct {
	httpClient *HttpClient
	events     chan string
	running    bool
}

func NewDigitalstromEvents(httpClient *HttpClient) *EventsManager {
	em := new(EventsManager)
	em.httpClient = httpClient
	return em
}

func (em *EventsManager) Start() {
	fmt.Println("Register subscription and listen to events")
	em.running = true
	em.registerSubscription()
	go em.listeningToevents()
}

func (em *EventsManager) Stop() {
	fmt.Println("Stopping events")
	em.running = false
}

func (em *EventsManager) registerSubscription() {
	em.httpClient.get("json/event/subscribe?name=" + EVENT_CALL_SCENE + "&subscriptionID=" + SUBSCRIPTION_ID)
	em.httpClient.get("json/event/subscribe?name=" + EVENT_BUTTON_CLICK + "&subscriptionID=" + SUBSCRIPTION_ID)
	em.httpClient.get("json/event/subscribe?name=" + EVENT_MODEL_READY + "&subscriptionID=" + SUBSCRIPTION_ID)
}

func (em *EventsManager) listeningToevents() {

	for {
		if !em.running {
			return
		}

		get, err := em.httpClient.get("json/event/get?subscriptionID=" + SUBSCRIPTION_ID)
		if checkNoError(err) {
			if ret, ok := get["events"]; ok {
				events := ret.([]interface{})
				fmt.Println("Event received ! ", events, len(events))
			}
		}
	}
}
