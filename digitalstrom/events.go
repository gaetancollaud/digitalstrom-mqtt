package digitalstrom

import (
	"math/rand"
	"strconv"
	"time"

	"github.com/gaetancollaud/digitalstrom-mqtt/utils"
	"github.com/rs/zerolog/log"
)

// TODO: Make this to be randomly generated on each run so parallel instances
// do not reuse the same subscription ID.
// const SUBSCRIPTION_ID = 42

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
	subscriptionId   int
}

func NewDigitalstromEvents(httpClient *HttpClient) *EventsManager {
	em := new(EventsManager)
	em.httpClient = httpClient
	em.events = make(chan Event)
	em.lastTokenCounter = -1
	// Random generate subscriptionId in order to not have collisions of
	// multiple instances running at the same time.
	rand.Seed(time.Now().UnixNano())
	em.subscriptionId = int(rand.Int31n(1 << 20))
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
	em.httpClient.EventUnsubscribe(EVENT_CALL_SCENE, em.subscriptionId)
	em.httpClient.EventUnsubscribe(EVENT_BUTTON_CLICK, em.subscriptionId)
	em.httpClient.EventUnsubscribe(EVENT_MODEL_READY, em.subscriptionId)
	log.Info().Str("SubscriptionId", strconv.Itoa(em.subscriptionId)).Msg("Unregistering from events")
}

func (em *EventsManager) registerSubscription() {
	log.Info().Str("SubscriptionId", strconv.Itoa(em.subscriptionId)).Msg("Registering to events")
	em.httpClient.EventSubscribe(EVENT_CALL_SCENE, em.subscriptionId)
	em.httpClient.EventSubscribe(EVENT_BUTTON_CLICK, em.subscriptionId)
	em.httpClient.EventSubscribe(EVENT_MODEL_READY, em.subscriptionId)
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

		response, err := em.httpClient.EventGet(em.subscriptionId)
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
