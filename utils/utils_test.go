package utils

import (
	"testing"
)

func TestRemoveRegexp(t *testing.T) {
	expect(t, RemoveRegexp("Light Location", "light"), "Location")
	expect(t, RemoveRegexp("light location", "light"), "location")
	expect(t, RemoveRegexp("location light", "light"), "location")
	expect(t, RemoveRegexp("Location Light", "light"), "Location")
}

func expect(t *testing.T, result string, expect string) {
	if expect != result {
		t.Errorf("Expected='%s' but got '%s'", expect, result)
	}
}
