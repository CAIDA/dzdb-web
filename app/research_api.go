package app

import (
	"dnscoffee/datastore"
	"dnscoffee/server"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

func (app *appContext) apiIPNsZoneCount(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	ip := cleanDomain(params["ip"])

	data, err := app.ds.GetIPNsZoneCount(r.Context(), ip)
	if err != nil {
		if err == datastore.ErrNoResource {
			server.WriteJSONError(w, server.ErrResourceNotFound)
			return
		}
		panic(err)
	}

	server.WriteJSON(w, data)
}

// apiActiveIPs exposes GetActiveIPs as an API
func (app *appContext) apiActiveIPs(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	date, err := time.Parse("2006-01-02", params["date"])
	if err != nil {
		panic(err)
	}

	data, err := app.ds.GetActiveIPs(r.Context(), date)
	if err != nil {
		if err == datastore.ErrNoResource {
			server.WriteJSONError(w, server.ErrResourceNotFound)
			return
		}
		panic(err)
	}

	server.WriteJSON(w, data)
}
