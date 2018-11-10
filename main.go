package main

import (
	"log"
)

// main
func main() {
	// get configuration
	config, err := GetConfig("config.ini")
	if err != nil {
		log.Fatal(err)
	}

	// get datstore
	ds, err := NewDataStore(&config.Postgresql)
	defer ds.Close()
	if err != nil {
		log.Fatal(err)
	}

	// get server and start application
	server := NewServer(config)
	AppStart(ds, server)
	log.Printf("Server starting on %s:%d", config.HTTP.IP, config.HTTP.Port)
	log.Fatal(server.Start())
}
