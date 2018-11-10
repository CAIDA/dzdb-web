package main

import (
	"gopkg.in/gcfg.v1"
)

// HTTPConfig server configuration
type HTTPConfig struct {
	IP   string
	Port int
}

// DatabaseConfig database configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSL      bool
	Timeout  int
}

// APIConfig API settings
type APIConfig struct {
	Timeout            int
	RequestsPerMinute  int
	RequestsMaxHistory int
	RequestsBurst      int
}

// Config main configuration object
type Config struct {
	HTTP       HTTPConfig
	Postgresql DatabaseConfig
	API        APIConfig
}

// GetConfig parses the configuration file identified by file
// and returns a Config object
func GetConfig(file string) (*Config, error) {
	var cfg Config
	err := gcfg.ReadFileInto(&cfg, file)
	return &cfg, err
}
