package utils

import (
	"encoding/json"
	"github.com/rs/zerolog/log"
)

func CheckNoErrorAndPrint(e error) bool {
	if e != nil {
		log.Info().Err(e).Msg("Error")
	}
	return e == nil
}

func PrettyPrintMap(value map[string]interface{}) string {
	b, err := json.Marshal(value)
	if err != nil {
		log.Info().Err(err).Msg("Cannot pretty print")
	}
	return string(b)
}

func PrettyPrintArray(value interface{}) string {
	b, err := json.Marshal(value)
	if err != nil {
		log.Info().Err(err).Msg("Cannot pretty print")
	}
	return string(b)
}
