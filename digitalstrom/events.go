package digitalstrom

import (
	"fmt"
	"github.com/gaetancollaud/digitalstrom-mqtt/utils"
	"strconv"
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

type Event struct {
	ZoneId  int
	SceneId int
}

type EventsManager struct {
	httpClient *HttpClient
	events     chan Event
	running    bool
}

func NewDigitalstromEvents(httpClient *HttpClient) *EventsManager {
	em := new(EventsManager)
	em.httpClient = httpClient
	em.events = make(chan Event)
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

		response, err := em.httpClient.get("json/event/get?subscriptionID=" + SUBSCRIPTION_ID)
		if utils.CheckNoError(err) {
			if ret, ok := response.mapValue["events"]; ok {
				events := ret.([]interface{})

				fmt.Println("Events received :", events, utils.PrettyPrintArray(events))

				for _, event := range events {
					m := event.(map[string]interface{})
					source := m["source"].(map[string]interface{})
					properties := m["properties"].(map[string]interface{})
					sceneId := -1
					if scene, ok := properties["sceneID"]; ok {
						sceneId, err = strconv.Atoi(scene.(string))
						utils.CheckNoError(err)
					}
					eventObj := Event{
						ZoneId:  int(source["zoneID"].(float64)),
						SceneId: sceneId,
					}
					em.events <- eventObj
				}
			}
		}
	}
}
