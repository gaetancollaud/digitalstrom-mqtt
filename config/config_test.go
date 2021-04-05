package config

import (
	"os"
	"testing"
)

func TestFromEnv(t *testing.T) {
	os.Setenv(envKeyDigitalstromHost, "test_ip")

	c := FromEnv()
	if c.Digitalstrom.Host != "test_ip" {
		t.Errorf("wrong Endpoint")
	}
}
