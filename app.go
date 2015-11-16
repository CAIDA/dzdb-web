package main

import (
	"net/http"
    "html/template"
	"strings"

	//"github.com/gorilla/context"
	//"github.com/julienschmidt/httprouter"
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
    server.Get("/apple", app.AppleHandler)
    server.Get("/orange", app.OrangeHandler)

}


func (app *appContext) IndexHandler(w http.ResponseWriter, r *http.Request) {
    p := Page{"Home"}
    err := app.templates.ExecuteTemplate(w, "home.tmpl", p)
    if err != nil {
		panic(err)
    }
}

func (app *appContext) AppleHandler(w http.ResponseWriter, r *http.Request) {
    p := Page{"Apples"}
    err := app.templates.ExecuteTemplate(w, "apple.tmpl", p)
    if err != nil {
		panic(err)
    }
}

func (app *appContext) OrangeHandler(w http.ResponseWriter, r *http.Request) {
    p := Page{"Oranges"}
    err := app.templates.ExecuteTemplate(w, "orange.tmpl", p)
    if err != nil {
		panic(err)
    }
}

func cleanDomain(domain string) string {
	domain, err := idna.ToASCII(strings.ToUpper(domain))
	if err != nil {
		panic(err)
	}
	return domain
}

