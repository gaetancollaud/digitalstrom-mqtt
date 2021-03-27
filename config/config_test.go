package config

import (
	"os"
	"testing"
)

func TestFromEnv(t *testing.T) {
	os.Setenv(envKeyDigitalstromIp, "test_ip")

	c := FromEnv()
	if c.Digitalstrom.Ip != "test_ip" {
		t.Errorf("wrong Endpoint")
	}
}
