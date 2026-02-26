package helper

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

func StringToStruct[I any](payload string) (result *I, err error) {
	err = json.Unmarshal([]byte(payload), &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func StringToJSON(payload string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(payload), &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func StringToInt(payload string) (int, error) {
	result, err := strconv.Atoi(payload)
	if err != nil {
		return 0, err
	}

	return result, nil
}

func StringToInt64(payload string) (int64, error) {
	result, err := strconv.ParseInt(payload, 10, 64)
	if err != nil {
		return 0, err
	}

	return result, nil
}

func StringToFloat64(payload string) (*float64, error) {
	// Remove "Rp" or "Rp." prefix if exists
	payload = strings.TrimPrefix(payload, "Rp.")
	payload = strings.TrimPrefix(payload, "Rp ")

	// Remove any dots and replace comma with dot for decimal point
	payload = strings.ReplaceAll(payload, ".", "")
	payload = strings.ReplaceAll(payload, ",", ".")

	// Parse the cleaned string to float64
	result, err := strconv.ParseFloat(strings.TrimSpace(payload), 64)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func StringToDate(data string) *time.Time {
	t, err := time.Parse("02 January 2006", data)
	if err != nil {
		return nil
	}
	return &t
}

func GetMapStringValue(header map[string]interface{}, key string) *string {
	str := ""
	value, exists := header[key]
	if !exists || value == nil {
		return &str
	}
	str = fmt.Sprintf("%v", value)
	return &str
}

func StringToUUID(payload string) (uuid.UUID, error) {
	return uuid.Parse(payload)
}

func ConvertToRawMessage(payload interface{}) *json.RawMessage {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil
	}
	rawMsg := json.RawMessage(raw)
	return &rawMsg
}

func ParseCommaSeperatedUUID(data string) []uuid.UUID {
	var uuids []uuid.UUID
	if data == "" {
		return uuids
	}

	parts := strings.Split(data, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		uuid, err := uuid.Parse(part)
		if err != nil {
			continue // Skip invalid UUIDs
		}
		uuids = append(uuids, uuid)
	}

	return uuids
}

func ParseCommaSeperatedString(data string) []string {
	var stringsList []string
	if data == "" {
		return stringsList
	}

	parts := strings.Split(data, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		stringsList = append(stringsList, part)
	}

	return stringsList
}

func UUIDPtrToStringPtr(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	s := id.String()
	return &s
}

func StringContains(parameter, value string) bool {
	return strings.Contains(parameter, value)
}
