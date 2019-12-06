package main

import (
	"log"
	"time"
	"vdz-web/app"
	"vdz-web/datastore"
	"vdz-web/server"
)

// main
func main() {
	// get datstore
	// if no DB wait for valid connection
	var ds *datastore.DataStore
	var err error
	for {
		ds, err = datastore.New("") // use standard env vars
		if err != nil {
			log.Println(err)
			log.Println("waiting for 30s")
			time.Sleep(30 * time.Second)
		} else {
			break
		}
	}
	defer ds.Close()

	// get server and start application
	vdzServer, err := server.New()
	if err != nil {
		log.Fatal(err)
	}
	app.Start(ds, vdzServer)
	// TODO allow setting addr/port
	log.Printf("Server starting on %s", vdzServer.ListenAddr)
	log.Fatal(vdzServer.Start())
}
