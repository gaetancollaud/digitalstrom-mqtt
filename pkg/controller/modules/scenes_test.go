package modules

import (
	"testing"

	mqtt_base "github.com/eclipse/paho.mqtt.golang"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/config"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/digitalstrom"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/mqtt"
)

type scenePublish struct {
	topic   string
	message string
	retain  bool
}

type sceneRawClientStub struct {
	mqtt_base.Client
	published []scenePublish
}

func (c *sceneRawClientStub) Publish(topic string, _ byte, retained bool, payload interface{}) mqtt_base.Token {
	c.published = append(c.published, scenePublish{topic, string(payload.([]byte)), retained})
	return &mqtt_base.DummyToken{}
}

type sceneMQTTClientStub struct {
	mqtt.Client
	raw sceneRawClientStub
}

func (c *sceneMQTTClientStub) GetFullTopic(topic string) string { return "digitalstrom/" + topic }
func (c *sceneMQTTClientStub) RawClient() mqtt_base.Client      { return &c.raw }

func TestScenesV1PayloadAndTopic(t *testing.T) {
	mqttClient := &sceneMQTTClientStub{}
	module := &ScenesModule{mqttClient: mqttClient, normalizeDeviceName: true}

	module.onSceneCall(digitalstrom.SceneCall{ZoneId: 1234, ZoneName: "Wohn Zimmer", GroupId: 1, SceneId: 17, SceneName: "Movie"})
	module.onSceneCall(digitalstrom.SceneCall{ZoneId: 0, ZoneName: "unnamed-zone-0", GroupId: 0, SceneId: 72, SceneName: "unnamed-scene-72"})
	module.onSceneCall(digitalstrom.SceneCall{ZoneId: 1234, ZoneName: "Wohn Zimmer", GroupId: -1, SceneId: 5})

	expected := []scenePublish{
		{"digitalstrom/scenes/Wohn_Zimmer/Movie/event", `{"ZoneId":1234,"ZoneName":"Wohn Zimmer","GroupId":1,"GroupName":"light","SceneId":17,"SceneName":"Movie"}`, false},
		{"digitalstrom/scenes/unnamed-zone-0/unnamed-scene-72/event", `{"ZoneId":0,"ZoneName":"unnamed-zone-0","GroupId":0,"GroupName":"unknown","SceneId":72,"SceneName":"unnamed-scene-72"}`, false},
		{"digitalstrom/scenes/Wohn_Zimmer/5/event", `{"ZoneId":1234,"ZoneName":"Wohn Zimmer","GroupId":-1,"GroupName":"unknown","SceneId":5,"SceneName":""}`, false},
	}
	if len(mqttClient.raw.published) != len(expected) {
		t.Fatalf("unexpected publishes %+v", mqttClient.raw.published)
	}
	for i := range expected {
		if mqttClient.raw.published[i] != expected[i] {
			t.Errorf("publish %d: got %+v, expected %+v", i, mqttClient.raw.published[i], expected[i])
		}
	}
}

func TestScenesTopicWithoutWildcards(t *testing.T) {
	module := &ScenesModule{}

	topic := module.sceneEventTopic(SceneEvent{ZoneName: "Living Room", SceneName: "TV + Licht #2 Auf/Zu"})
	if topic != "scenes/Living Room/TV _ Licht _2 Auf_Zu/event" {
		t.Fatalf("unexpected topic %s", topic)
	}
}

func TestScenesDisabledDoesNotCreateLegacyClient(t *testing.T) {
	module := NewScenesModule(&sceneMQTTClientStub{}, nil, nil, &config.Config{ScenesEnabled: false}).(*ScenesModule)

	if err := module.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	if module.legacyClient != nil {
		t.Fatal("legacy client created while scenes are disabled")
	}
	if err := module.Stop(); err != nil {
		t.Fatalf("stop: %v", err)
	}
}
