package utils

import (
	"encoding/json"
	"fmt"
)

func CheckNoError(e error) bool {
	if e != nil {
		panic(fmt.Errorf("Error with token: %v\n", e))
	}
	return e == nil
}

func PrettyPrintMap(value map[string]interface{}) string {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fmt.Println("error:", err)
	}
	return string(b)
}

func PrettyPrintArray(value interface{}) string {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fmt.Println("error:", err)
	}
	return string(b)
}
