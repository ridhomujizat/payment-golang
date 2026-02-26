package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"

	"github.com/joho/godotenv"
)

// GetEnv TODO: Add validation for the config
func GetEnv() (config *Config, er error) {
	err := godotenv.Load()
	if err != nil {
		_ = godotenv.Load("../../.env")
	}

	config = &Config{}
	v := reflect.ValueOf(config).Elem()
	t := v.Type()

	for i := range make([]struct{}, v.NumField()) {
		field := t.Field(i)
		envTag := field.Tag.Get("env")

		if envTag != "" {
			value, exists := os.LookupEnv(envTag)
			if !exists {
				er = fmt.Errorf("environment variable %s not set", envTag)
				return nil, er
			}

			switch field.Type.Kind() {
			case reflect.String:
				v.Field(i).SetString(value)
			case reflect.Int:
				intValue, err := strconv.Atoi(value)
				if err != nil {
					er = fmt.Errorf("invalid value for %s: %v", envTag, err)
					return nil, er
				}
				v.Field(i).SetInt(int64(intValue))
			case reflect.Bool:
				boolValue, err := strconv.ParseBool(value)
				if err != nil {
					er = fmt.Errorf("invalid boolean value for %s: %v", envTag, err)
					return nil, er
				}
				v.Field(i).SetBool(boolValue)
			default:
				panic("unhandled default case")
			}
		}
	}

	return config, nil
}
