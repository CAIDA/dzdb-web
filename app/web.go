package app

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"dnscoffee/app/temfun"
	"dnscoffee/datastore"
	"dnscoffee/model"
	"dnscoffee/server"
	"dnscoffee/version"

	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/idna"
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
	Data  interface{}
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

	//TODO add feeds page
	//server.Get("/feeds", app.TodoHandler)
	server.Get("/version", app.VersionHandler)
	server.Get("/about", app.AboutHandler)

	server.Get("/", app.IndexHandler)

	server.Get("/nameservers", app.nameserverIndexHandler)
	server.Get("/ip", app.ipIndexHandler)

	server.Post("/search", app.searchHandler)
	server.Get("/search", app.searchIndexHandler)
	server.Get("/search/prefix", app.prefixIndexHandler)
	server.Get("/search/feed", app.searchFeedHandler)
	server.Get("/search/prefix/:type/:prefix", app.prefixHandler)

	server.Get("/domains", app.domainIndexHandler)
	server.Get("/domains/:domain", app.domainHandler)
	server.Get("/ip/:ip", app.ipHandler)
	server.Get("/nameservers/:nameserver", app.nameserverHandler)
	server.Get("/zones/:zone", app.zoneHandler)
	server.Get("/zones", app.zoneIndexHandler)
	server.Get("/tlds", app.tldIndexHandler)
	server.Get("/tlds/graveyard", app.tldGraveyardIndexHandler)
	server.Get("/stats", app.statsHandler)

	// research
	server.Get("/research/ipnszonecount/:ip", app.ipNsZoneCountHandler)
	server.Get("/research/trust-tree", app.trustTreeHandler)
}

func (app *appContext) searchIndexHandler(w http.ResponseWriter, r *http.Request) {
	var s model.Search
	p := Page{"Search", "", s}
	err := app.templates.ExecuteTemplate(w, "search.tmpl", p)
	if err != nil {
		panic(err)
	}
}

func (app *appContext) searchHandler(w http.ResponseWriter, r *http.Request) {
	var s model.Search
	s.Query = cleanDomain(r.FormValue("query"))
	s.Type = r.FormValue("type")
	var err error

	// first handle when there is only a single result and single result type
	switch s.Type {
	case "zone":
		_, err = app.ds.GetZoneID(r.Context(), s.Query)
		if err == nil {
			http.Redirect(w, r, "/zones/"+s.Query, http.StatusFound)
			return
		}
	case "domain":
		_, _, err = app.ds.GetDomainID(r.Context(), s.Query)
		if err == nil {
			http.Redirect(w, r, "/domains/"+s.Query, http.StatusFound)
			return
		}
	case "nameserver":
		_, err = app.ds.GetNameServerID(r.Context(), s.Query)
		if err == nil {
			http.Redirect(w, r, "/nameservers/"+s.Query, http.StatusFound)
			return
		}
	case "ip":
		_, _, err = app.ds.GetIPID(r.Context(), s.Query)
		if err == nil {
			http.Redirect(w, r, "/ip/"+s.Query, http.StatusFound)
			return
		}
	default:
		s.Results = make([]model.SearchResult, 0)
		// now handle multiple results types
		// this is a very poor exact match search... add prefix too?
		if _, err = app.ds.GetZoneID(r.Context(), s.Query); err == nil {
			s.Results = append(s.Results, model.SearchResult{Name: s.Query, Link: "/zones/" + s.Query, Type: "zone"})
		}
		if _, _, err = app.ds.GetDomainID(r.Context(), s.Query); err == nil {
			s.Results = append(s.Results, model.SearchResult{Name: s.Query, Link: "/domains/" + s.Query, Type: "domain"})
		}
		if _, err = app.ds.GetNameServerID(r.Context(), s.Query); err == nil {
			s.Results = append(s.Results, model.SearchResult{Name: s.Query, Link: "/nameservers/" + s.Query, Type: "nameserver"})
		}
		if _, _, err = app.ds.GetIPID(r.Context(), s.Query); err == nil {
			s.Results = append(s.Results, model.SearchResult{Name: s.Query, Link: "/ip/" + s.Query, Type: "IP"})
		}

		// still want to redirect if only one type is found
		if len(s.Results) == 1 {
			http.Redirect(w, r, s.Results[0].Link, http.StatusFound)
			return
		}
	}

	// render search page
	p := Page{"Search", "", s}
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

func (app *appContext) tldIndexHandler(w http.ResponseWriter, r *http.Request) {
	data, err := app.ds.GetZoneImportResults(r.Context())
	if err != nil {
		panic(err)
	}

	p := Page{"TLDs", "TLDs", data}
	err = app.templates.ExecuteTemplate(w, "tlds.tmpl", p)
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

		domains, err := app.ds.GetDomainsInZoneID(r.Context(), data.ID)
		if err != nil {
			panic(err)
		}
		data.Domains = &domains
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

// prefixIndexHandler shows the prefix search page
func (app *appContext) prefixIndexHandler(w http.ResponseWriter, r *http.Request) {
	var data model.PrefixList
	p := Page{"Prefix", "Search", data}
	err := app.templates.ExecuteTemplate(w, "prefix.tmpl", p)
	if err != nil {
		panic(err)
	}
}

// searchFeedHandler shows the feed search page
func (app *appContext) searchFeedHandler(w http.ResponseWriter, r *http.Request) {
	var data model.PrefixList
	p := Page{"Feed", "Search", data}
	err := app.templates.ExecuteTemplate(w, "feed_search.tmpl", p)
	if err != nil {
		panic(err)
	}
}

// prefixHandler returns prefixes for the queried prefix
func (app *appContext) prefixHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var data *model.PrefixList
	params := httprouter.ParamsFromContext(r.Context())
	prefixType := strings.ToLower(params.ByName("type"))
	name := cleanDomain(params.ByName("prefix"))
	if prefixType == "active" {
		data, err = app.ds.GetTakenPrefixes(r.Context(), name)

	} else if prefixType == "available" {
		data, err = app.ds.GetAvailablePrefixes(r.Context(), name)
	} else {
		server.WriteJSONError(w, server.ErrResourceNotFound)
		return
	}
	if err != nil {
		if err == datastore.ErrNoResource {
			// TODO make http err (not json)
			server.WriteJSONError(w, server.ErrResourceNotFound)
			return
		}
		panic(err)
	}

	p := Page{name + " Prefix", "Search", data}
	err = app.templates.ExecuteTemplate(w, "prefix.tmpl", p)
	if err != nil {
		panic(err)
	}
}

func (app *appContext) VersionHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%s\n", version.String())
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

func (app *appContext) trustTreeHandler(w http.ResponseWriter, r *http.Request) {
	p := Page{"Trust Tree", "Trust Tree", nil}
	err := app.templates.ExecuteTemplate(w, "trusttree.tmpl", p)
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

func (app *appContext) tldGraveyardIndexHandler(w http.ResponseWriter, r *http.Request) {
	data, err := app.ds.GetDeadTLDs(r.Context())
	if err != nil {
		panic(err)
	}
	p := Page{"TLD Graveyard", "Zones", data}
	err = app.templates.ExecuteTemplate(w, "tld_graveyard.tmpl", p)
	if err != nil {
		panic(err)
	}
}
