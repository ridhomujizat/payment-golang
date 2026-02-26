package helper

import (
	"os"
	"strconv"
)

// GetEnv retrieves an environment variable or returns the default value
func GetEnv(key string, defaultValue ...string) string {
	value := os.Getenv(key)
	if value == "" && len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return value
}

// GetEnvAsInt retrieves an environment variable as an integer
func GetEnvAsInt(name string) int {
	if val, ok := os.LookupEnv(name); ok {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return 0
}

// GetEnvAsIntWithDefault retrieves an environment variable as an integer with a default value
func GetEnvAsIntWithDefault(name string, defaultValue int) int {
	if val, ok := os.LookupEnv(name); ok {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultValue
}
