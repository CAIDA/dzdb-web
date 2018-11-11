package main

import (
	"log"
	"vdz-web/app"
	"vdz-web/config"
	"vdz-web/datastore"
	"vdz-web/server"
)

// main
func main() {
	// get configuration
	config, err := config.Parse("config.ini")
	if err != nil {
		log.Fatal(err)
	}

	// get datstore
	ds, err := datastore.New(&config.Postgresql)
	defer ds.Close()
	if err != nil {
		log.Fatal(err)
	}

	// get server and start application
	vdzServer := server.New(config)
	app.Start(ds, vdzServer)
	log.Printf("Server starting on %s:%d", config.HTTP.IP, config.HTTP.Port)
	log.Fatal(vdzServer.Start())
}
