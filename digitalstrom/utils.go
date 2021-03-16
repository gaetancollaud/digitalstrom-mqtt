package digitalstrom

import "fmt"

func checkNoError(e error) bool {
	if e != nil {
		panic(fmt.Errorf("Error with token: %v\n", e))
	}
	return e == nil
}
