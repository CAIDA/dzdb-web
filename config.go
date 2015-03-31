package main

import (
	"code.google.com/p/gcfg"
)

// HTTP server configuration
type HttpConfig struct {
	IP   string
	Port int
}

// database configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSL      bool
}

// API settings
type APIConfig struct {
	SQL_Timeout     int
	API_Timeout     int
	Requests_PerMin int
}

// main configuration object
type Config struct {
	Http       HttpConfig
	Postgresql DatabaseConfig
	API        APIConfig
}

// parsed the configuration file identified by file
// and returns a Config object
func GetConfig(file string) (*Config, error) {
	var cfg Config
	err := gcfg.ReadFileInto(&cfg, file)
	return &cfg, err
}
