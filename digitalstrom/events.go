package digitalstrom

import (
	"github.com/gaetancollaud/digitalstrom-mqtt/utils"
	"github.com/rs/zerolog/log"
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
	httpClient       *HttpClient
	events           chan Event
	running          bool
	lastTokenCounter int
}

func NewDigitalstromEvents(httpClient *HttpClient) *EventsManager {
	em := new(EventsManager)
	em.httpClient = httpClient
	em.events = make(chan Event)
	em.lastTokenCounter = -1
	return em
}

func (em *EventsManager) Start() {
	log.Info().Msg("Starting event manager")
	em.running = true
	go em.listeningToevents()
}

func (em *EventsManager) Stop() {
	log.Info().Msg("Stopping events")
	em.running = false
}

func (em *EventsManager) registerSubscription() {
	log.Info().Str("SubscriptionId", SUBSCRIPTION_ID).Msg("Registering to events")
	em.httpClient.get("json/event/subscribe?name=" + EVENT_CALL_SCENE + "&subscriptionID=" + SUBSCRIPTION_ID)
	em.httpClient.get("json/event/subscribe?name=" + EVENT_BUTTON_CLICK + "&subscriptionID=" + SUBSCRIPTION_ID)
	em.httpClient.get("json/event/subscribe?name=" + EVENT_MODEL_READY + "&subscriptionID=" + SUBSCRIPTION_ID)
}

func (em *EventsManager) listeningToevents() {
	for {
		if !em.running {
			return
		}

		if em.lastTokenCounter < em.httpClient.TokenManager.tokenCounter {
			// new token ? so new subscription
			em.registerSubscription()
			em.lastTokenCounter = em.httpClient.TokenManager.tokenCounter
		}

		response, err := em.httpClient.get("json/event/get?subscriptionID=" + SUBSCRIPTION_ID)
		if utils.CheckNoErrorAndPrint(err) {
			if ret, ok := response.mapValue["events"]; ok {
				events := ret.([]interface{})

				//log.Info().Msg("Events received :", events, utils.PrettyPrintArray(events))

				for _, event := range events {
					m := event.(map[string]interface{})
					source := m["source"].(map[string]interface{})
					properties := m["properties"].(map[string]interface{})
					sceneId := -1
					if scene, ok := properties["sceneID"]; ok {
						sceneId, err = strconv.Atoi(scene.(string))
						utils.CheckNoErrorAndPrint(err)
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
