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
	Timeout  int
}

// API settings
type APIConfig struct {
	Timeout             int
	Requests_Per_Minute int
	Requests_Max_History int
	Requests_Burst	int
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
