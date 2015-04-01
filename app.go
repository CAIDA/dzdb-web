package main

import (
	"net/http"
	"strings"

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
	server.Get_Index("/domains/:domain/nameservers", "domain_nameservers", HandlerNotImplemented)                 //TODO
	server.Get_Index("/domains/:domain/archive_nameservers", "domain_archive_nameservers", HandlerNotImplemented) //TODO
	server.Get_Index("/nameservers/:domain", "nameserver", app.nameserverHandler)
	server.Get_Index("/nameservers/:domain/domains", "nameserver_domains", HandlerNotImplemented)                 //TODO
	server.Get_Index("/nameservers/:domain/archive_domains", "nameserver_archive_domains", HandlerNotImplemented) //TODO
}

// domainHandler returns domain object for the queried domain
func (app *appContext) domainHandler(w http.ResponseWriter, r *http.Request) {
	params := context.Get(r, "params").(httprouter.Params)
	domain, err := idna.ToASCII(strings.ToUpper(params.ByName("domain")))
	if err != nil {
		panic(err)
	}
	data, err1 := app.ds.getDomain(domain)
	if err1 != nil {
		if err1 == ErrNoResource {
			WriteJSONError(w, ErrResourceNotFound)
			return
		}
		panic(err1)
	}

	data.generateMetaData()
	WriteJSON(w, data)
}

// randomDomainHandler returns a random domain from the system
func (app *appContext) randomDomainHandler(w http.ResponseWriter, r *http.Request) {
	domain, err := app.ds.getRandomDomain()
	if err != nil {
		panic(err)
	}
	domain.generateMetaData()
	WriteJSON(w, domain)
}

// nameserverHandler returns nameserver object for the queried domain
func (app *appContext) nameserverHandler(w http.ResponseWriter, r *http.Request) {
	params := context.Get(r, "params").(httprouter.Params)
	domain, err := idna.ToASCII(strings.ToUpper(params.ByName("domain")))
	if err != nil {
		panic(err)
	}

	data, err1 := app.ds.getNameServer(domain)
	if err1 != nil {
		//TODO combine common code below
		if err1 == ErrNoResource {
			WriteJSONError(w, ErrResourceNotFound)
			return
		}

		panic(err1)
	}

	data.generateMetaData()
	WriteJSON(w, data)
}
