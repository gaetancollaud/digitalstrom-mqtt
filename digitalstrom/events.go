package digitalstrom

import (
	"strconv"
	"time"

	"github.com/gaetancollaud/digitalstrom-mqtt/utils"
	"github.com/rs/zerolog/log"
)

// TODO: Make this to be randomly generated on each run so parallel instances
// do not reuse the same subscription ID.
const SUBSCRIPTION_ID = 42

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
	GroupId int

	IsApartment bool
	IsDevice    bool
	IsGroup     bool
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
	go em.listeningToEvents()
}

func (em *EventsManager) Stop() {
	log.Info().Msg("Stopping events")
	em.running = false
	em.httpClient.EventUnsubscribe(EVENT_CALL_SCENE, SUBSCRIPTION_ID)
	em.httpClient.EventUnsubscribe(EVENT_BUTTON_CLICK, SUBSCRIPTION_ID)
	em.httpClient.EventUnsubscribe(EVENT_MODEL_READY, SUBSCRIPTION_ID)
	log.Info().Str("SubscriptionId", strconv.Itoa(SUBSCRIPTION_ID)).Msg("Unregistering from events")
}

func (em *EventsManager) registerSubscription() {
	log.Info().Str("SubscriptionId", strconv.Itoa(SUBSCRIPTION_ID)).Msg("Registering to events")
	em.httpClient.EventSubscribe(EVENT_CALL_SCENE, SUBSCRIPTION_ID)
	em.httpClient.EventSubscribe(EVENT_BUTTON_CLICK, SUBSCRIPTION_ID)
	em.httpClient.EventSubscribe(EVENT_MODEL_READY, SUBSCRIPTION_ID)
}

func (em *EventsManager) listeningToEvents() {
	for {
		if !em.running {
			return
		}

		if em.lastTokenCounter < em.httpClient.TokenManager.tokenCounter {
			// new token ? so new subscription
			em.registerSubscription()
			em.lastTokenCounter = em.httpClient.TokenManager.tokenCounter
		}

		response, err := em.httpClient.EventGet(SUBSCRIPTION_ID)
		if utils.CheckNoErrorAndPrint(err) {
			if ret, ok := response.mapValue["events"]; ok {
				events := ret.([]interface{})

				log.Trace().Str("event", utils.PrettyPrintArray(events)).Msg("Events received :")

				for _, event := range events {
					m := event.(map[string]interface{})
					source := m["source"].(map[string]interface{})
					properties := m["properties"].(map[string]interface{})
					sceneId := -1
					groupId := -1
					if scene, ok := properties["sceneID"]; ok {
						sceneId, err = strconv.Atoi(scene.(string))
						utils.CheckNoErrorAndPrint(err)
					}
					if group, ok := properties["groupID"]; ok {
						groupId, err = strconv.Atoi(group.(string))
						utils.CheckNoErrorAndPrint(err)
					}
					eventObj := Event{
						ZoneId:      int(source["zoneID"].(float64)),
						GroupId:     groupId,
						SceneId:     sceneId,
						IsApartment: source["isApartment"].(bool),
						IsDevice:    source["isDevice"].(bool),
						IsGroup:     source["isGroup"].(bool),
					}
					em.events <- eventObj
				}
			} else {
				log.Warn().Msg("No event present")
			}
		} else {
			time.Sleep(1000 * time.Millisecond)
		}
	}
}
