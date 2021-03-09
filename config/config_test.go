package config

import (
	"os"
	"testing"
)

func TestFromEnv(t *testing.T) {
	os.Setenv(envKeyDigitalstromUrl, "test_url")

	c := FromEnv()
	if c.Url != "test_url" {
		t.Errorf("wrong Endpoint")
	}
}
