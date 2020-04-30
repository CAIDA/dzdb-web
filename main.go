package main

import (
	"context"
	"dnscoffee/app"
	"dnscoffee/datastore"
	"dnscoffee/server"
	"flag"
	"log"
	"time"
)

var (
	listenAddr = flag.String("listen", "127.0.0.1:8080", "ip:port to listen on")
)

// main
func main() {
	flag.Parse()
	// get datstore
	// if no DB wait for valid connection
	var ds *datastore.DataStore
	var err error
	ctx := context.Background()
	for {
		ds, err = datastore.New(ctx)
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
	coffeeServer, err := server.New(*listenAddr, server.DefaultAPIConfig)
	if err != nil {
		log.Fatal(err)
	}
	app.Start(ds, coffeeServer)
	log.Printf("Server starting on %s", *listenAddr)
	log.Fatal(coffeeServer.Start())
}
