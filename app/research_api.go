package app

import (
	"net/http"
	"vdz-web/datastore"
	"vdz-web/server"

	"github.com/julienschmidt/httprouter"
)

func (app *appContext) apiIPNsZoneCount(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	ip := cleanDomain(params.ByName("ip"))

	data, err := app.ds.GetIPNsZoneCount(r.Context(), ip)
	if err != nil {
		if err == datastore.ErrNoResource {
			server.WriteJSONError(w, server.ErrResourceNotFound)
			return
		}
		panic(err)
	}

	data.GenerateMetaData()
	server.WriteJSON(w, data)
}
