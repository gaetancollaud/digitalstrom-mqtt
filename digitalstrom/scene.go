package digitalstrom

import (
	"github.com/gaetancollaud/digitalstrom-mqtt/utils"
	"github.com/rs/zerolog/log"
	"strconv"
)

type SceneEvent struct {
	ZoneId   int
	ZoneName string
	SceneId  int
}

type SceneManager struct {
	httpClient *HttpClient
	zonesById  map[int]string
	sceneEvent chan SceneEvent
}

func NewSceneManager(httpClient *HttpClient) *SceneManager {
	sm := new(SceneManager)
	sm.httpClient = httpClient
	sm.sceneEvent = make(chan SceneEvent)
	sm.zonesById = make(map[int]string)
	return sm
}

func (sm *SceneManager) Start() {
}

func (sm *SceneManager) getZoneName(zoneId int) (string, error) {
	name, ok := sm.zonesById[zoneId]
	if ok {
		return name, nil
	} else {
		response, err := sm.httpClient.get("/json/zone/getName?id=" + strconv.Itoa(zoneId))
		if utils.CheckNoErrorAndPrint(err) {
			name = response.mapValue["name"].(string)
			sm.zonesById[zoneId] = name
			return name, nil
		}
		return "", err
	}
}

func (sm *SceneManager) EventReceived(event Event) {
	log.Debug().Int("zoneId", event.ZoneId).Int("sceneId", event.SceneId).Msg("New scene event")
	name, err := sm.getZoneName(event.ZoneId)
	if err == nil {
		sceneEvent := SceneEvent{
			ZoneId:   event.ZoneId,
			ZoneName: name,
			SceneId:  event.SceneId,
		}
		sm.sceneEvent <- sceneEvent
	}
}
