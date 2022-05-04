package digitalstrom

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInvertValue(t *testing.T) {
	var deviceManager = DevicesManager{}
	assert.Equal(t, 40.0, deviceManager.invertValueIfNeeded("brightness", 40), "light should not be inverted")
	assert.Equal(t, 40.0, deviceManager.invertValueIfNeeded("shadePositionOutside", 40), "blinds should not be inverted")

	deviceManager.invertBlindsPosition = true

	assert.Equal(t, 40.0, deviceManager.invertValueIfNeeded("brightness", 40), "light should not be inverted")
	assert.Equal(t, 60.0, deviceManager.invertValueIfNeeded("shadePositionOutside", 40), "blinds should be inverted")
}
