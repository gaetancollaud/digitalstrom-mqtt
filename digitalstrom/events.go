package digitalstrom

import (
	"github.com/gaetancollaud/digitalstrom-mqtt/digitalstrom/api"
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

type EventsManager struct {
	events chan api.Event
}

func NewDigitalstromEvents() *EventsManager {
	em := new(EventsManager)
	em.events = make(chan api.Event)
	return em
}
