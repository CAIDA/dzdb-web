package main

import (
	"log"
	"time"
	"vdz-web/app"
	"vdz-web/config"
	"vdz-web/datastore"
	"vdz-web/server"
)

// main
func main() {
	// get configuration
	// TODO move to env vars
	config, err := config.Parse("config.ini")
	if err != nil {
		log.Fatal(err)
	}

	// get datstore
	// if no DB wait for valid connection
	var ds *datastore.DataStore
	for {
		ds, err = datastore.New(&config.Postgresql)
		if err != nil {
			log.Println(err)
			time.Sleep(30 * time.Second)
		}
		break
	}
	defer ds.Close()

	// get server and start application
	vdzServer := server.New(config)
	app.Start(ds, vdzServer)
	log.Printf("Server starting on %s:%d", config.HTTP.IP, config.HTTP.Port)
	log.Fatal(vdzServer.Start())
}
