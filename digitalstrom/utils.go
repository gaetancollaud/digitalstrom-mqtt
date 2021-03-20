package digitalstrom

import (
	"encoding/json"
	"fmt"
)

func checkNoError(e error) bool {
	if e != nil {
		panic(fmt.Errorf("Error with token: %v\n", e))
	}
	return e == nil
}

func prettyPrintMap(value map[string]interface{}) string {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fmt.Println("error:", err)
	}
	return string(b)
}

func prettyPrintArray(value interface{}) string {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fmt.Println("error:", err)
	}
	return string(b)
}
