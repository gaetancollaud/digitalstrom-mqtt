package controller

import (
	"errors"
	"testing"

	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/config"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/controller/modules"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/digitalstrom"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/homeassistant"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/mqtt"
	"github.com/stretchr/testify/require"
)

type testMQTT struct {
	mqtt.Client
	connect func() error
}

func (m testMQTT) Connect() error { return m.connect() }

type testDSS struct{ digitalstrom.Client }

func (testDSS) Connect() error { return nil }

type testRegistry struct{ digitalstrom.Registry }

func (testRegistry) Start() error { return nil }

type testModule struct{ start func() error }

func (m testModule) Start() error { return m.start() }
func (testModule) Stop() error    { return nil }

type testHealth struct{ started *bool }

func (h testHealth) Start() error { *h.started = true; return nil }
func (testHealth) Stop() error    { return nil }

func TestHealthIsUnavailableUntilEveryModuleStarts(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "module failure"}[fail], func(t *testing.T) {
			healthStarted := false
			mqttClient := testMQTT{connect: func() error { require.False(t, healthStarted); return nil }}
			c := &Controller{
				mqttClient: mqttClient, dsClient: testDSS{}, dsRegistry: testRegistry{},
				healthCheck:   testHealth{&healthStarted},
				hassDiscovery: homeassistant.NewHomeAssistantDiscovery(mqttClient, &config.ConfigHomeAssistant{}),
				modules: map[string]modules.Module{"test": testModule{start: func() error {
					require.False(t, healthStarted)
					if fail {
						return errors.New("initialization failed")
					}
					return nil
				}}},
			}
			err := c.Start()
			if fail {
				require.Error(t, err)
				require.False(t, healthStarted)
			} else {
				require.NoError(t, err)
				require.True(t, healthStarted)
			}
		})
	}
}
