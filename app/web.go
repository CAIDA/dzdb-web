package app

import (
	"html/template"
	"net/http"
	"strings"

	"vdz-web/app/temfun"
	"vdz-web/datastore"
	"vdz-web/server"

	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/idna"
	//"log"
)

// object to hold application context and persistent storage
type appContext struct {
	ds *datastore.DataStore

	// used for creating the API index
	api map[string]string

	templates *template.Template
}

// Page holds information for rendered HTML pages
type Page struct {
	Title string
	Tab   string
	Data  interface{} // TODO set this to interface type with generate metadata
}

// Start entry point for starting application
// adds routes to the server so that the correct handlers are registered
func Start(ds *datastore.DataStore, server *server.Server) {
	var app appContext
	app.ds = ds
	// compile all templates and cache them
	//app.templates = template.Must(template.ParseGlob("templates/*.tmpl").Funcs(temfun.Funcs))
	app.templates = template.Must(template.New("main").Funcs(temfun.Funcs).ParseGlob("templates/*.tmpl"))

	// load the api
	APIStart(&app, server)

	//TODO
	//server.Get("/feeds", app.TodoHandler)
	server.Get("/about", app.AboutHandler)

	server.Get("/", app.IndexHandler)
	server.Get("/robots.txt", app.robotsTxtHandler) // TODO move to server static handling

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
	server.Get("/prefix/:prefix", app.prefixHandler)

	// research
	server.Get("/research/ipnszonecount/:ip", app.ipNsZoneCountHandler)

}

func (app *appContext) searchHandler(w http.ResponseWriter, r *http.Request) {
	query := cleanDomain(r.FormValue("query"))
	//t := r.FormValue("type")
	// TODO re-enable this
	var err error
	//log.Print("Type: ", t)

	_, err = app.ds.GetZoneID(r.Context(), query)
	if err == nil {
		// redirect
		http.Redirect(w, r, "/zones/"+query, http.StatusFound)
		return
	}

	_, _, err = app.ds.GetDomainID(r.Context(), query)
	if err == nil {
		// redirect
		http.Redirect(w, r, "/domains/"+query, http.StatusFound)
		return
	}

	_, err = app.ds.GetNameServerID(r.Context(), query)
	if err == nil {
		// redirect
		http.Redirect(w, r, "/nameservers/"+query, http.StatusFound)
		return
	}

	_, _, err = app.ds.GetIPID(r.Context(), query)
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
	data, err := app.ds.GetImportProgress(r.Context())
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
	data, err := app.ds.GetZoneImportResults(r.Context())
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
	params := httprouter.ParamsFromContext(r.Context())
	name := cleanDomain(params.ByName("zone"))
	data, err := app.ds.GetZone(r.Context(), name)
	if err != nil {
		if err == datastore.ErrNoResource {
			// TODO make http err (not json)
			server.WriteJSONError(w, server.ErrResourceNotFound)
			return
		}
		panic(err)
	}
	importData, err := app.ds.GetZoneImport(r.Context(), name)
	if err == nil {
		// TODO check for datastore.ErrNoResource and sql.NoRows
		// TODO in fact, make ErrNoResource include? sql.NowRows as well
		data.ImportData = importData
	}

	p := Page{name, "Zones", data}
	err = app.templates.ExecuteTemplate(w, "zone.tmpl", p)
	if err != nil {
		panic(err)
	}
}

func (app *appContext) nameserverHandler(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	name := cleanDomain(params.ByName("nameserver"))
	data, err := app.ds.GetNameServer(r.Context(), name)
	if err != nil {
		if err == datastore.ErrNoResource {
			// TODO make http err (not json)
			server.WriteJSONError(w, server.ErrResourceNotFound)
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
	params := httprouter.ParamsFromContext(r.Context())
	domain := cleanDomain(params.ByName("domain"))
	data, err := app.ds.GetDomain(r.Context(), domain)
	if err != nil {
		if err == datastore.ErrNoResource {
			// TODO make http err (not json)
			server.WriteJSONError(w, server.ErrResourceNotFound)
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
	params := httprouter.ParamsFromContext(r.Context())
	name := cleanDomain(params.ByName("ip"))
	data, err := app.ds.GetIP(r.Context(), name)
	if err != nil {
		if err == datastore.ErrNoResource {
			// TODO make http err (not json)
			server.WriteJSONError(w, server.ErrResourceNotFound)
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

// prefixHandler returns avaible prefixes for the queried prefix
func (app *appContext) prefixHandler(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	name := cleanDomain(params.ByName("prefix"))
	data, err := app.ds.GetAvaiblePrefixes(r.Context(), name)
	if err != nil {
		if err == datastore.ErrNoResource {
			// TODO make http err (not json)
			server.WriteJSONError(w, server.ErrResourceNotFound)
			return
		}
		panic(err)
	}

	p := Page{name, "", data}
	err = app.templates.ExecuteTemplate(w, "prefix.tmpl", p)
	if err != nil {
		panic(err)
	}
}

func (app *appContext) AboutHandler(w http.ResponseWriter, r *http.Request) {
	p := Page{"About", "About", nil}
	err := app.templates.ExecuteTemplate(w, "about.tmpl", p)
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

func (app *appContext) robotsTxtHandler(w http.ResponseWriter, r *http.Request) {
	err := app.templates.ExecuteTemplate(w, "robots.tmpl", nil)
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

// research
func (app *appContext) ipNsZoneCountHandler(w http.ResponseWriter, r *http.Request) {
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

	p := Page{"IP NS Zone Count", "IP NS Zone Count", data}
	err = app.templates.ExecuteTemplate(w, "ipnszonecount.tmpl", p)
	if err != nil {
		panic(err)
	}
}

// helper
func cleanDomain(domain string) string {
	domain = strings.TrimSpace(domain)
	domain, err := idna.ToASCII(strings.ToUpper(domain))
	if err != nil {
		panic(err)
	}
	return domain
}
