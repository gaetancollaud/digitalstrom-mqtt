package digitalstrom

import (
	"strconv"

	"github.com/gaetancollaud/digitalstrom-mqtt/utils"
	"github.com/rs/zerolog/log"
)

type SceneEvent struct {
	ZoneId    int
	ZoneName  string
	GroupId   int
	GroupName string
	SceneId   int
	SceneName string
}

type SceneIdentifier struct {
	ZoneId  int
	GroupId int
	SceneId int
}

type SceneManager struct {
	httpClient *HttpClient
	zonesById  map[int]string
	sceneById  map[SceneIdentifier]string
	sceneEvent chan SceneEvent
}

func NewSceneManager(httpClient *HttpClient) *SceneManager {
	sm := new(SceneManager)
	sm.httpClient = httpClient
	sm.sceneEvent = make(chan SceneEvent)
	sm.zonesById = make(map[int]string)
	sm.sceneById = make(map[SceneIdentifier]string)
	return sm
}

func (sm *SceneManager) Start() {
}

func (sm *SceneManager) getZoneName(zoneId int) (string, error) {
	name, ok := sm.zonesById[zoneId]
	if ok {
		return name, nil
	} else {
		response, err := sm.httpClient.ZoneGetName(zoneId)
		if utils.CheckNoErrorAndPrint(err) {
			name = response.mapValue["name"].(string)
			if len(name) == 0 {
				name = "unnamed-zone-" + strconv.Itoa(zoneId)
			}
			sm.zonesById[zoneId] = name
			return name, nil
		}
		return "", err
	}
}

func (sm *SceneManager) getSceneName(zoneId int, groupId int, sceneId int) (string, error) {
	if groupId == -1 {
		return "", nil
	}
	id := SceneIdentifier{
		ZoneId:  zoneId,
		GroupId: groupId,
		SceneId: sceneId,
	}
	name, ok := sm.sceneById[id]
	if ok {
		return name, nil
	} else {
		response, err := sm.httpClient.ZoneSceneGetName(zoneId, groupId, sceneId)
		if utils.CheckNoErrorAndPrint(err) {
			name = response.mapValue["name"].(string)
			if len(name) == 0 {
				name = "unnamed-scene-" + strconv.Itoa(sceneId)
			}
			sm.sceneById[id] = name
			return name, nil
		}
		return "", err
	}
}

func (sm *SceneManager) EventReceived(event Event) {
	log.Debug().Int("zoneId", event.ZoneId).Int("sceneId", event.SceneId).Msg("New scene event")
	zoneName, errZone := sm.getZoneName(event.ZoneId)
	sceneName, errScene := sm.getSceneName(event.ZoneId, event.GroupId, event.SceneId)
	if errZone == nil && errScene == nil {
		sceneEvent := SceneEvent{
			ZoneId:    event.ZoneId,
			ZoneName:  zoneName,
			GroupId:   event.GroupId,
			GroupName: sm.getGroupName(event.GroupId),
			SceneId:   event.SceneId,
			SceneName: sceneName,
		}
		sm.sceneEvent <- sceneEvent
	}
}

func (sm *SceneManager) getGroupName(id int) string {
	switch id {
	case 1:
		return "light"
	case 2:
		return "shade"
	case 3:
		return "climate"
	case 4:
		return "audio"
	case 5:
		return "video"
	case 6:
		return "safety"
	case 7:
		return "access"
	case 8:
		return "joker"
	default:
		return "unknown"
	}

}
