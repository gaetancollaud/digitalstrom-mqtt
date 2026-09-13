package mqtt

import (
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/monitor"
	"github.com/stretchr/testify/require"
)

func TestReconnectRecoversButRejectedCredentialsStopAttempts(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })
	connections := make(chan net.Conn, 4)
	var attempts atomic.Int32
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			n := attempts.Add(1)
			go func() {
				_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
				if _, err := packets.ReadPacket(conn); err != nil {
					_ = conn.Close()
					return
				}
				ack := packets.NewControlPacket(packets.Connack).(*packets.ConnackPacket)
				if n >= 3 {
					ack.ReturnCode = packets.ErrRefusedNotAuthorised
				}
				if err := ack.Write(conn); err != nil {
					_ = conn.Close()
					return
				}
				if n >= 3 {
					_ = conn.Close()
					return
				}
				connections <- conn
			}()
		}
	}()
	options := NewClientOptions().SetMqttUrl("tcp://" + listener.Addr().String())
	options.Monitor = monitor.New()
	c := NewClient(options)
	t.Cleanup(func() { c.RawClient().Disconnect(0) })
	require.NoError(t, c.Connect())
	first := <-connections
	require.NoError(t, first.Close())
	var second net.Conn
	select {
	case second = <-connections:
	case <-time.After(8 * time.Second):
		t.Fatal("MQTT did not recover after connection loss")
	}
	require.Eventually(t, c.RawClient().IsConnectionOpen, time.Second, 10*time.Millisecond)
	require.NoError(t, options.Monitor.Check(), "a recoverable outage must not trigger liveness failure")
	require.NoError(t, second.Close())
	select {
	case err := <-options.Monitor.Failures():
		require.True(t, monitor.IsPermanent(err))
	case <-time.After(8 * time.Second):
		t.Fatal("MQTT rejection did not reach the App runtime")
	}
	// Paho's first reconnect delay is one second. Observe beyond that window.
	time.Sleep(1500 * time.Millisecond)
	require.Equal(t, int32(3), attempts.Load(), "authentication rejection must stop reconnects")
}
