package helper

import (
	"encoding/json"
	"fmt"
)

func JSONToString(payload any) (string, error) {
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error:", err)
		return "", err
	}

	jsonString := string(jsonBytes)
	return jsonString, nil
}

func JSONToStruct[I any](payload any) (result *I, err error) {
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(jsonBytes, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func JSONToByte(payload any) ([]byte, error) {
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error:", err)
		return nil, err
	}
	return jsonBytes, nil
}
