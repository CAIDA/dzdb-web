package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/context"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/idna"
)

// object to hold application context and persistent storage
type appContext struct {
	ds *DataStore
}

// Entry point for starting application
// adds routes to the server so that the correct handlers are registered
func AppStart(ds *DataStore, server *server) {
	app := appContext{ds}
	server.Get_Index("/random", "random_domain", app.randomDomainHandler)
	server.Get_Index("/domains/:domain", "domain", app.domainHandler)
	server.Get_Index("/nameservers/:domain", "nameserver", app.nameserverHandler)
}

// domainHandler returns domain object for the queried domain
func (app *appContext) domainHandler(w http.ResponseWriter, r *http.Request) {
	params := context.Get(r, "params").(httprouter.Params)
	domain, err := idna.ToASCII(params.ByName("domain"))
	if err != nil {
		panic(err)
	}
	data, err1 := app.ds.getDomain(domain)
	if err1 != nil {
		panic(err1)
	}

	w.Header().Set("Content-Type", "application/vnd.api+json")
	err = json.NewEncoder(w).Encode(JSONResponse{data})
	if err != nil && err != http.ErrHandlerTimeout {
		panic(err)
	}
}

// randomDomainHandler returns a random domain from the system
func (app *appContext) randomDomainHandler(w http.ResponseWriter, r *http.Request) {
	domain, err := app.ds.getRandomDomain()
	if err != nil {
		panic(err)
	}
	w.Header().Set("Content-Type", "application/vnd.api+json")
	err = json.NewEncoder(w).Encode(JSONResponse{domain})
	if err != nil && err != http.ErrHandlerTimeout {
		panic(err)
	}
}

// nameserverHandler returns nameserver object for the queried domain
func (app *appContext) nameserverHandler(w http.ResponseWriter, r *http.Request) {
	params := context.Get(r, "params").(httprouter.Params)
	domain, err := idna.ToASCII(params.ByName("domain"))
	if err != nil {
		panic(err)
	}

	data, err1 := app.ds.getNameServer(domain)
	if err1 != nil {
		panic(err1)
	}

	w.Header().Set("Content-Type", "application/vnd.api+json")
	err = json.NewEncoder(w).Encode(JSONResponse{data})
	if err != nil && err != http.ErrHandlerTimeout {
		panic(err)
	}
}
