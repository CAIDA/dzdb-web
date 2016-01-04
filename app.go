package main

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/gorilla/context"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/idna"
)

// object to hold application context and persistent storage
type appContext struct {
	ds *DataStore

	// used for creating the API index
	api map[string]string

	templates *template.Template
}

// compile all templates and cache them

type Page struct {
	Title string
	Tab   string
	Data  interface{}
}

// Entry point for starting application
// adds routes to the server so that the correct handlers are registered
func AppStart(ds *DataStore, server *server) {
	var app appContext
	app.ds = ds
	app.templates = template.Must(template.ParseGlob("templates/*.tmpl"))

	// load the api
	APIStart(&app, server)

	//TODO
	server.Get("/feeds", app.TodoHandler)

	server.Get("/", app.IndexHandler)

	server.Get("/nameservers", app.nameserverIndexHandler)
	server.Get("/ip", app.ipIndexHandler)

	server.Post("/search", app.searchHandler)

	server.Get("/domains", app.domainIndexHandler)
	server.Get("/domains/:domain", app.domainHandler)
	server.Get("/ip/:ip", app.ipHandler)
	server.Get("/nameservers/:nameserver", app.nameserverHandler)
	server.Get("/zones/:zone", app.zoneHandler)
	server.Get("/zones", app.zoneIndexHandler)
	server.Get("/stats", app.statsHandler)
}

func (app *appContext) searchHandler(w http.ResponseWriter, r *http.Request) {
	query := cleanDomain(r.FormValue("query"))
	//t := r.FormValue("type")
	var err error

	_, err = app.ds.getZoneID(query)
	if err == nil {
		// redirect
		http.Redirect(w, r, "/zones/"+query, http.StatusFound)
		return
	}

	_, _, err = app.ds.getDomainID(query)
	if err == nil {
		// redirect
		http.Redirect(w, r, "/domains/"+query, http.StatusFound)
		return
	}

	_, err = app.ds.getNameServerID(query)
	if err == nil {
		// redirect
		http.Redirect(w, r, "/nameservers/"+query, http.StatusFound)
		return
	}

	_, _, err = app.ds.getIPID(query)
	if err == nil {
		// redirect
		http.Redirect(w, r, "/ip/"+query, http.StatusFound)
		return
	}

	// no results found
	p := Page{"Search", "", query}
	err = app.templates.ExecuteTemplate(w, "search.tmpl", p)
	if err != nil {
		panic(err)
	}
}

func (app *appContext) statsHandler(w http.ResponseWriter, r *http.Request) {
	data, err := app.ds.getImportProgress()
	if err != nil {
		panic(err)
	}

	p := Page{"Stats", "Stats", data}
	err = app.templates.ExecuteTemplate(w, "stats.tmpl", p)
	if err != nil {
		panic(err)
	}
}

func (app *appContext) zoneIndexHandler(w http.ResponseWriter, r *http.Request) {
	data, err := app.ds.getZoneImportResults()
	if err != nil {
		panic(err)
	}

	p := Page{"Zones", "Zones", data}
	err = app.templates.ExecuteTemplate(w, "zones.tmpl", p)
	if err != nil {
		panic(err)
	}
}

func (app *appContext) zoneHandler(w http.ResponseWriter, r *http.Request) {
	params := context.Get(r, "params").(httprouter.Params)
	name := cleanDomain(params.ByName("zone"))
	data, err := app.ds.getZone(name)
	if err != nil {
		if err == ErrNoResource {
			// TODO make http err (not json)
			WriteJSONError(w, ErrResourceNotFound)
			return
		}
		panic(err)
	}

	p := Page{name, "Zones", data}
	err = app.templates.ExecuteTemplate(w, "zone.tmpl", p)
	if err != nil {
		panic(err)
	}
}

func (app *appContext) nameserverHandler(w http.ResponseWriter, r *http.Request) {
	params := context.Get(r, "params").(httprouter.Params)
	name := cleanDomain(params.ByName("nameserver"))
	data, err := app.ds.getNameServer(name)
	if err != nil {
		if err == ErrNoResource {
			// TODO make http err (not json)
			WriteJSONError(w, ErrResourceNotFound)
			return
		}
		panic(err)
	}

	p := Page{name, "Nameservers", data}
	err = app.templates.ExecuteTemplate(w, "nameserver.tmpl", p)
	if err != nil {
		panic(err)
	}
}

// domainHandler returns domain object for the queried domain
func (app *appContext) domainHandler(w http.ResponseWriter, r *http.Request) {
	params := context.Get(r, "params").(httprouter.Params)
	domain := cleanDomain(params.ByName("domain"))
	data, err := app.ds.getDomain(domain)
	if err != nil {
		if err == ErrNoResource {
			// TODO make http err (not json)
			WriteJSONError(w, ErrResourceNotFound)
			return
		}
		panic(err)
	}

	p := Page{domain, "Domains", data}
	err = app.templates.ExecuteTemplate(w, "domain.tmpl", p)
	if err != nil {
		panic(err)
	}
}

// ipHandler returns ip object for the queried domain
func (app *appContext) ipHandler(w http.ResponseWriter, r *http.Request) {
	params := context.Get(r, "params").(httprouter.Params)
	name := cleanDomain(params.ByName("ip"))
	data, err := app.ds.getIP(name)
	if err != nil {
		if err == ErrNoResource {
			// TODO make http err (not json)
			WriteJSONError(w, ErrResourceNotFound)
			return
		}
		panic(err)
	}

	p := Page{name, "IPs", data}
	err = app.templates.ExecuteTemplate(w, "ip.tmpl", p)
	if err != nil {
		panic(err)
	}
}

func (app *appContext) TodoHandler(w http.ResponseWriter, r *http.Request) {
	p := Page{"TODO", "", nil}
	err := app.templates.ExecuteTemplate(w, "todo.tmpl", p)
	if err != nil {
		panic(err)
	}
}

func (app *appContext) IndexHandler(w http.ResponseWriter, r *http.Request) {
	p := Page{"Home", "Home", nil}
	err := app.templates.ExecuteTemplate(w, "home.tmpl", p)
	if err != nil {
		panic(err)
	}
}

func (app *appContext) domainIndexHandler(w http.ResponseWriter, r *http.Request) {
	p := Page{"Domains", "Domains", nil}
	err := app.templates.ExecuteTemplate(w, "domains.tmpl", p)
	if err != nil {
		panic(err)
	}
}

func (app *appContext) nameserverIndexHandler(w http.ResponseWriter, r *http.Request) {
	p := Page{"Name Servers", "Nameservers", nil}
	err := app.templates.ExecuteTemplate(w, "nameservers.tmpl", p)
	if err != nil {
		panic(err)
	}
}

func (app *appContext) ipIndexHandler(w http.ResponseWriter, r *http.Request) {
	p := Page{"IPs", "IPs", nil}
	err := app.templates.ExecuteTemplate(w, "ips.tmpl", p)
	if err != nil {
		panic(err)
	}
}

// helper
func cleanDomain(domain string) string {
	domain, err := idna.ToASCII(strings.ToUpper(domain))
	if err != nil {
		panic(err)
	}
	return domain
}
