package helper

import (
	"github.com/samber/lo"
	"strconv"
	"strings"
)

func GetMapBoolValue(header map[string]interface{}, key string) *bool {
	val := false
	parsed, err := strconv.ParseBool(strings.ToLower(*GetMapStringValue(header, key)))
	return lo.Ternary(err == nil, &parsed, &val)
}
