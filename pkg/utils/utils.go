package utils

import (
	"regexp"
	"strings"
)

func RemoveRegexp(value string, expression string) string {
	if expression == "" {
		return value
	}
	regex := regexp.MustCompile("(?i)" + expression)
	return strings.TrimSpace(regex.ReplaceAllString(value, ""))
}
