package modules

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/config"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/digitalstrom"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/mqtt"
	"github.com/rs/zerolog/log"
)

const (
	scenes         string = "scenes"
	publishTimeout        = 10 * time.Second
)

// Group names used by the scene events of version 1.x.
var sceneGroupNames = map[int]string{
	1: "light",
	2: "shade",
	3: "climate",
	4: "audio",
	5: "video",
	6: "safety",
	7: "access",
	8: "joker",
}

// MQTT forbids the wildcards in published topics, and a '/' would add a topic
// level.
var sceneTopicReplacer = strings.NewReplacer("+", "_", "#", "_", "/", "_")

// SceneEvent is the scene call message of version 1.x, published on
// scenes/{zoneName}/{sceneName or sceneId}/event. Field names are kept as is
// (no JSON tags) to stay compatible with existing consumers.
type SceneEvent struct {
	ZoneId    int
	ZoneName  string
	GroupId   int
	GroupName string
	SceneId   int
	SceneName string
}

// Scenes Module forwards the scene calls of digitalSTROM to MQTT, with the
// same topic and payload as version 1.x. Scene calls are not available in the
// Smarthome API, so they are read from the legacy JSON API.
type ScenesModule struct {
	mqttClient mqtt.Client

	enabled             bool
	normalizeDeviceName bool
	dsOptions           *digitalstrom.ClientOptions
	legacyClient        digitalstrom.LegacyClient
}

func (c *ScenesModule) Start() error {
	if !c.enabled {
		log.Info().Msg("Scenes module disabled.")
		return nil
	}
	c.legacyClient = digitalstrom.NewLegacyClient(c.dsOptions)
	return c.legacyClient.SceneCallsStart(c.onSceneCall)
}

func (c *ScenesModule) Stop() error {
	if c.legacyClient != nil {
		c.legacyClient.SceneCallsStop()
	}
	return nil
}

func (c *ScenesModule) onSceneCall(call digitalstrom.SceneCall) {
	groupName, ok := sceneGroupNames[call.GroupId]
	if !ok {
		groupName = "unknown"
	}
	event := SceneEvent{
		ZoneId:    call.ZoneId,
		ZoneName:  call.ZoneName,
		GroupId:   call.GroupId,
		GroupName: groupName,
		SceneId:   call.SceneId,
		SceneName: call.SceneName,
	}
	log.Debug().
		Int("zoneId", event.ZoneId).
		Int("groupId", event.GroupId).
		Int("sceneId", event.SceneId).
		Str("scene", event.SceneName).
		Msg("Scene event")

	message, err := json.Marshal(event)
	if err != nil {
		log.Error().Err(err).Msg("Error serializing scene event")
		return
	}
	// Events are never retained: a retained event would be replayed, and the
	// scene applied again, by every client subscribing later.
	topic := c.mqttClient.GetFullTopic(c.sceneEventTopic(event))
	t := c.mqttClient.RawClient().Publish(topic, mqtt.QOS, false, message)
	if t.WaitTimeout(publishTimeout) && t.Error() != nil {
		log.Error().Err(t.Error()).Str("topic", topic).Msg("Error publishing scene event")
	}
}

func (c *ScenesModule) sceneEventTopic(event SceneEvent) string {
	zoneName := event.ZoneName
	if c.normalizeDeviceName {
		zoneName = normalizeForTopicName(zoneName)
	}
	sceneNameOrId := event.SceneName
	if sceneNameOrId == "" {
		// No name for the scene, take the id instead.
		sceneNameOrId = strconv.Itoa(event.SceneId)
	}
	return scenes + "/" + sceneTopicReplacer.Replace(zoneName) + "/" + sceneTopicReplacer.Replace(sceneNameOrId) + "/" + mqtt.Event
}

func NewScenesModule(mqttClient mqtt.Client, _ digitalstrom.Client, _ digitalstrom.Registry, config *config.Config) Module {
	return &ScenesModule{
		mqttClient:          mqttClient,
		enabled:             config.ScenesEnabled,
		normalizeDeviceName: config.Mqtt.NormalizeDeviceName,
		dsOptions: digitalstrom.NewClientOptions().
			SetHost(config.Digitalstrom.Host).
			SetPort(config.Digitalstrom.Port).
			SetApiKey(config.Digitalstrom.ApiKey),
	}
}

func init() {
	Register(scenes, NewScenesModule)
}
