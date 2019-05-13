package model

import (
	"os"
	"strconv"
)

// GetStringEnv returns the environment variable with the key
// if it is not set the fallback is provided
func GetStringEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = fallback
	}
	return value
}

// GetIntEnv returns the environment variable with the key
// if it is not set the fallback is provided
func GetIntEnv(key string, fallback int) (int, error) {
	value, exists := os.LookupEnv(key)
	if !exists {
		return fallback, nil
	}
	v, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return v, nil
}

// GetBoolEnv returns the environment variable with the key
// if it is not set the fallback is provided
func GetBoolEnv(key string, fallback bool) (bool, error) {
	value, exists := os.LookupEnv(key)
	if !exists {
		return fallback, nil
	}
	v, err := strconv.ParseBool(value)
	if err != nil {
		return false, err
	}
	return v, nil
}
