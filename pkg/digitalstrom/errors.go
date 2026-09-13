package digitalstrom

import (
	"fmt"
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/monitor"
)

func responseError(status int) error {
	err := fmt.Errorf("digitalSTROM returned HTTP %d", status)
	if status >= 400 && status < 500 && status != 408 && status != 429 {
		return monitor.Permanent(err)
	}
	return err
}
