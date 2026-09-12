package mqtt

import (
	"errors"
	"fmt"
	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/monitor"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestConnectionErrorClassification(t *testing.T) {
	for _, err := range []error{packets.ErrorRefusedBadUsernameOrPassword, packets.ErrorRefusedNotAuthorised} {
		require.True(t, monitor.IsPermanent(connectionError(fmt.Errorf("connect: %w", err))))
	}
	for _, err := range []error{packets.ErrorRefusedServerUnavailable, errors.New("connection refused")} {
		require.False(t, monitor.IsPermanent(connectionError(err)))
	}
}
