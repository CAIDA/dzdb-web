package main

import (
	"log"
)

// main
func main() {
	// get configuration
	config, config_err := GetConfig("config.ini")
	if config_err != nil {
		log.Fatal(config_err)
	}

	// get datstore
	ds, ds_err := NewDataStore(&config.Postgresql)
	defer ds.Close()
	if ds_err != nil {
		log.Fatal(ds_err)
	}

	// get server and start application
	server := NewServer(config)
	AppStart(ds, server)
	log.Fatal(server.Start())
}
