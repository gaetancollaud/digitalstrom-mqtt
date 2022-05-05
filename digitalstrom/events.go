package digitalstrom

import "github.com/gaetancollaud/digitalstrom-mqtt/pkg/digitalstrom"

type EventsManager struct {
	events chan digitalstrom.Event
}

func NewDigitalstromEvents() *EventsManager {
	em := new(EventsManager)
	em.events = make(chan digitalstrom.Event)
	return em
}
