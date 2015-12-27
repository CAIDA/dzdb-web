package main

import (
	"net/http"
    "html/template"
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
    Tab string
    Data interface{}
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
	server.Get("/", app.IndexHandler)

    server.Get("/nameservers", app.TodoHandler)
    server.Get("/ip", app.TodoHandler)
    server.Get("/zones", app.TodoHandler)
    server.Get("/feeds", app.TodoHandler)
    server.Get("/stats", app.TodoHandler)

    server.Get("/domains", app.DomainIndexHandler)
	server.Get("/domains/:domain", app.domainHandler)
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
    err = app.templates.ExecuteTemplate(w, "domains.tmpl", p)
    if err != nil {
		panic(err)
    }

    // TODO remove this
	//WriteJSON(w, data)
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

func (app *appContext) DomainIndexHandler(w http.ResponseWriter, r *http.Request) {
    p := Page{"Domains", "Domains", nil}
    err := app.templates.ExecuteTemplate(w, "domainIndex.tmpl", p)
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

