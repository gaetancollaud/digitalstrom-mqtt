package modules

import (
	"encoding/json"
	"fmt"
	"path"
	"strconv"

	mqtt_base "github.com/eclipse/paho.mqtt.golang"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/config"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/digitalstrom"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/mqtt"
	"github.com/rs/zerolog/log"
)

const (
	scenes string = "scenes"
)

// Circuit Module encapsulates all the logic regarding the circuits. The logic
// is the following: every 30 seconds the circuit values are being checked and
// pushed to the corresponding topic in the MQTT server.
type SceneModule struct {
	mqttClient mqtt.Client
	dsClient   digitalstrom.Client

	normalizeTopicName bool

	scenes      []Scene
	sceneLookup map[int]map[int]Scene
}

// Structure to hold the information about a Scene.
type Scene struct {
	ZoneId    int    `json:"ZoneId"`
	ZoneName  string `json:"ZoneName"`
	GroupId   int    `json:"GroupId"`
	GroupName string `json:"GroupName"`
	SceneId   int    `json:"SceneId"`
	SceneName string `json:"SceneName"`
}

func (c *SceneModule) Start() error {
	// First retrieve all available groups in the apartment.
	response, err := c.dsClient.ApartmentGetReachableGroups()
	if err != nil {
		return fmt.Errorf("error retrieving reachable groups from apartment: %w", err)
	}

	// Retrieve all the scenes available in the apartment.
	for _, zone := range response.Zones {
		for _, groupId := range zone.Groups {
			response, err := c.dsClient.ZoneGetReachableScenes(zone.Id, groupId)
			if err != nil {
				return fmt.Errorf("error retrieving scenes for zone %d and group %d: %w", zone.Id, groupId, err)
			}
			// Create a lookup map for the scenes that have names.
			sceneNameMapping := map[int]string{}
			for _, sceneName := range response.UserSceneNames {
				sceneNameMapping[sceneName.Number] = sceneName.Name
			}
			// Take all the scenes in the response and add them into the scenes
			// list.
			for _, sceneId := range response.ReachableScenes {
				sceneName, ok := sceneNameMapping[sceneId]
				if !ok {
					// Set a default name for scenes without user provided name.
					sceneName = "scene-" + strconv.Itoa(sceneId)
				}
				scene := Scene{
					ZoneId:    zone.Id,
					ZoneName:  zone.Name,
					GroupId:   groupId,
					SceneId:   sceneId,
					SceneName: sceneName,
				}
				c.scenes = append(c.scenes, scene)
			}
		}
	}

	// Create maps regarding Scenes for fast lookup when a new Event is
	// received.
	for _, scene := range c.scenes {
		if _, ok := c.sceneLookup[scene.ZoneId]; !ok {
			c.sceneLookup[scene.ZoneId] = map[int]Scene{}
		}
		c.sceneLookup[scene.ZoneId][scene.GroupId] = scene
	}

	// Subscribe to DigitalStrom events.
	if err := c.dsClient.EventSubscribe(digitalstrom.EventCallScene, func(client digitalstrom.Client, event digitalstrom.Event) error {
		return c.onDsEvent(event)
	}); err != nil {
		return err
	}

	// Subscribe to MQTT events.
	for _, scene := range c.scenes {
		topic := c.sceneCommandTopic(scene.ZoneName, scene.SceneName)
		log.Trace().
			Str("topic", topic).
			Str("zoneName", scene.ZoneName).
			Str("sceneName", scene.SceneName).
			Msg("Subscribing for topic.")
		c.mqttClient.Subscribe(topic, func(mqtt_base.Client, mqtt_base.Message) {
			// Payload is ignored. As long as we receive the message to the
			// command topic, the scene will be called.
			if err := c.onMqttMessage(&scene); err != nil {
				log.Error().Str("topic", topic).Err(err).Msg("Error handling MQTT Message.")
			}
		})
	}
	return nil
}

func (c *SceneModule) Stop() error {
	if err := c.dsClient.EventUnsubscribe(digitalstrom.EventCallScene); err != nil {
		return err
	}
	return nil
}

func (c *SceneModule) onMqttMessage(scene *Scene) error {
	log.Info().
		Int("zoneId", scene.ZoneId).
		Int("groupId", scene.GroupId).
		Int("sceneId", scene.SceneId).
		Msg("Received MQTT command to set scene")
	return c.dsClient.ZoneCallScene(scene.ZoneId, scene.GroupId, scene.SceneId)
}

func (c *SceneModule) onDsEvent(event digitalstrom.Event) error {
	// Only events that come from groups correspond to a scene.
	if !event.Source.IsGroup {
		log.Debug().Msg("Received event which does not come from a group and therefore does not match a scene.")
		return nil
	}

	log.Info().Msg("onDsEvent from scene.")
	scene, ok := c.sceneLookup[event.Source.ZoneId][event.Source.GroupId]
	if !ok {
		log.Warn().
			Int("zoneId", event.Source.ZoneId).
			Int("groupID", event.Source.GroupId).
			Msg("No scene found for group when event received.")
		return fmt.Errorf("error when retrieving scene given a zone and group ID")
	}
	if err := c.publishScene(&scene); err != nil {
		return fmt.Errorf("error publishing scene to MQTT: %w", err)
	}
	return nil
}

func (c *SceneModule) publishScene(scene *Scene) error {
	message, err := json.Marshal(scene)
	if err != nil {
		return fmt.Errorf("error encoding scene into json: %w", err)
	}
	return c.mqttClient.Publish(c.sceneEventTopic(scene.ZoneName, scene.SceneName), message)
}

func (c *SceneModule) sceneEventTopic(zoneName string, sceneName string) string {
	if c.normalizeTopicName {
		zoneName = normalizeForTopicName(zoneName)
		sceneName = normalizeForTopicName(sceneName)
	}
	return path.Join(scenes, zoneName, sceneName, mqtt.Event)
}

func (c *SceneModule) sceneCommandTopic(zoneName string, sceneName string) string {
	if c.normalizeTopicName {
		zoneName = normalizeForTopicName(zoneName)
		sceneName = normalizeForTopicName(sceneName)
	}
	return path.Join(scenes, zoneName, sceneName, mqtt.Command)
}

func NewSceneModule(mqttClient mqtt.Client, dsClient digitalstrom.Client, config *config.Config) Module {
	return &SceneModule{
		mqttClient:         mqttClient,
		dsClient:           dsClient,
		normalizeTopicName: config.Mqtt.NormalizeDeviceName,
		scenes:             []Scene{},
		sceneLookup:        map[int]map[int]Scene{},
	}
}

func init() {
	Register("scenes", NewSceneModule)
}
