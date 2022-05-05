package utils

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
)

func CheckNoErrorAndPrint(e error) bool {
	if e != nil {
		log.Error().Err(e).Msg("Error")
	}
	return e == nil
}

func PrettyPrint(value interface{}) string {
	b, err := json.Marshal(value)
	if err != nil {
		log.Info().Err(err).Msg("Cannot pretty print")
	}
	return string(b)
}

func RemoveRegexp(value string, expression string) string {
	if expression == "" {
		return value
	}
	regex := regexp.MustCompile("(?i)" + expression)
	return strings.TrimSpace(regex.ReplaceAllString(value, ""))
}
