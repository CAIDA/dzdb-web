package app

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"unicode"

	"dnscoffee/app/temfun"
	"dnscoffee/datastore"
	"dnscoffee/model"
	"dnscoffee/server"
	"dnscoffee/version"

	"github.com/gorilla/mux"
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
	server.Get("/search/trends", app.searchTrendsHandler)
	server.Get("/search/prefix/{type}/{prefix}", app.prefixHandler)

	server.Get("/domains", app.domainIndexHandler)
	server.Get("/domains/{domain}", app.domainHandler)
	server.Get("/ip/{ip}", app.ipHandler)
	server.Get("/nameservers/{nameserver}", app.nameserverHandler)
	server.Get("/root", app.rootHandler)
	server.Get("/zones/{zone}", app.zoneHandler)
	server.Get("/zones", app.zoneIndexHandler)
	server.Get("/tlds", app.tldIndexHandler)
	server.Get("/tlds/graveyard", app.tldGraveyardIndexHandler)

	// research
	server.Get("/research/trust-tree", app.trustTreeHandler)
	server.Get("/research/ipnszonecount/{ip}", app.ipNsZoneCountHandler)
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

	// since the root zone is the empty string, this prevents empty searches from redirecting to the zones page
	if len(s.Query) > 0 {
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
		case "_":
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
		default:
			http.Redirect(w, r, app.findObjectLinkByName(r.Context(), s.Query), http.StatusFound)
		}
	}

	// render search page
	p := Page{"Search", "Search", s}
	err = app.templates.ExecuteTemplate(w, "search.tmpl", p)
	if err != nil {
		panic(err)
	}
}

// used for the search redirect
func (app *appContext) findObjectLinkByName(ctx context.Context, s string) string {
	if _, err := app.ds.GetZoneID(ctx, s); err == nil {
		return "/zones/" + s
	}
	if _, _, err := app.ds.GetDomainID(ctx, s); err == nil {
		return "/domains/" + s
	}
	if _, err := app.ds.GetNameServerID(ctx, s); err == nil {
		return "/nameservers/" + s
	}
	if _, _, err := app.ds.GetIPID(ctx, s); err == nil {
		return "/ip/" + s
	}
	return ""
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

	p := Page{"TLDs", "Zones", data}
	err = app.templates.ExecuteTemplate(w, "tlds.tmpl", p)
	if err != nil {
		panic(err)
	}
}

func (app *appContext) rootHandler(w http.ResponseWriter, r *http.Request) {
	name := ""
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

	p := Page{"ROOT Zone", "Zones", data}
	err = app.templates.ExecuteTemplate(w, "root.tmpl", p)
	if err != nil {
		panic(err)
	}
}

func (app *appContext) zoneHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	name := cleanDomain(params["zone"])
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
	params := mux.Vars(r)
	name := cleanDomain(params["nameserver"])
	data, err := app.ds.GetNameServer(r.Context(), name)
	if err != nil {
		if err == datastore.ErrNoResource {
			// TODO make http err (not json)
			server.WriteJSONError(w, server.ErrResourceNotFound)
			return
		}
		panic(err)
	}

	p := Page{name, "Records", data}
	err = app.templates.ExecuteTemplate(w, "nameserver.tmpl", p)
	if err != nil {
		panic(err)
	}
}

// domainHandler returns domain object for the queried domain
func (app *appContext) domainHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	domain := cleanDomain(params["domain"])
	data, err := app.ds.GetDomain(r.Context(), domain)
	if err != nil {
		if err == datastore.ErrNoResource {
			// TODO make http err (not json)
			server.WriteJSONError(w, server.ErrResourceNotFound)
			return
		}
		panic(err)
	}

	p := Page{domain, "Records", data}
	err = app.templates.ExecuteTemplate(w, "domain.tmpl", p)
	if err != nil {
		panic(err)
	}
}

// ipHandler returns ip object for the queried domain
func (app *appContext) ipHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	name := cleanDomain(params["ip"])
	data, err := app.ds.GetIP(r.Context(), name)
	if err != nil {
		if err == datastore.ErrNoResource {
			// TODO make http err (not json)
			server.WriteJSONError(w, server.ErrResourceNotFound)
			return
		}
		panic(err)
	}

	p := Page{data.Name, "Records", data}
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

// searchTrendsHandler shows the trend search page
func (app *appContext) searchTrendsHandler(w http.ResponseWriter, r *http.Request) {
	var data model.PrefixList
	p := Page{"Trends", "Search", data}
	err := app.templates.ExecuteTemplate(w, "trends_search.tmpl", p)
	if err != nil {
		panic(err)
	}
}

// prefixHandler returns prefixes for the queried prefix
func (app *appContext) prefixHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var data *model.PrefixList
	params := mux.Vars(r)
	prefixType := strings.ToLower(params["type"])
	name := cleanDomain(params["prefix"])
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
	var data model.Dataset
	var err error
	data.TopNameServers, err = app.ds.GetTopNameservers(r.Context(), 20)
	if err != nil {
		panic(err)
	}

	p := Page{"Home", "", data}
	err = app.templates.ExecuteTemplate(w, "home.tmpl", p)
	if err != nil {
		panic(err)
	}
}

func (app *appContext) domainIndexHandler(w http.ResponseWriter, r *http.Request) {
	p := Page{"Domains", "Records", nil}
	err := app.templates.ExecuteTemplate(w, "domains.tmpl", p)
	if err != nil {
		panic(err)
	}
}

func (app *appContext) nameserverIndexHandler(w http.ResponseWriter, r *http.Request) {
	p := Page{"Name Servers", "Records", nil}
	err := app.templates.ExecuteTemplate(w, "nameservers.tmpl", p)
	if err != nil {
		panic(err)
	}
}

func (app *appContext) ipIndexHandler(w http.ResponseWriter, r *http.Request) {
	p := Page{"IPs", "Records", nil}
	err := app.templates.ExecuteTemplate(w, "ips.tmpl", p)
	if err != nil {
		panic(err)
	}
}

// research
func (app *appContext) ipNsZoneCountHandler(w http.ResponseWriter, r *http.Request) {
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

	p := Page{"IP NS Zone Count", "Research", data}
	err = app.templates.ExecuteTemplate(w, "ipnszonecount.tmpl", p)
	if err != nil {
		panic(err)
	}
}

func (app *appContext) trustTreeHandler(w http.ResponseWriter, r *http.Request) {
	p := Page{"Trust Tree", "Research", nil}
	err := app.templates.ExecuteTemplate(w, "trusttree.tmpl", p)
	if err != nil {
		panic(err)
	}
}

// From zonetools/parser/clean.go
var punyCode = idna.Registration

// cleanDomain lowercases all inputs and converts to punycode if necessary
// assumes input domain to be unicode, if it is not, we will guess the encoding
// and convert to UTF-8 return value should always be ASCII
func cleanDomain(domain string) string {
	domain = strings.TrimSpace(domain)
	// if ASCII just lowercase
	if isASCII(domain) {
		domain = strings.ToUpper(domain)
		return domain
	}
	// for non ASCII domains, only lowercase ASCII portions
	domain = asciiLower(domain)
	// convert unicode to ascii via puny code
	punycode, err := punyCode.ToASCII(domain)
	if err != nil {
		panic(fmt.Errorf("idna parse error on (%q -> %q) %w", domain, punycode, err))
	}
	punycode = strings.ToUpper(punycode)
	return punycode
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > unicode.MaxASCII {
			return false
		}
	}
	return true
}

// asciiLower only lowercases ASCII chars
func asciiLower(s string) string {
	out := make([]rune, 0, len(s))
	for _, char := range s {
		if char <= unicode.MaxASCII {
			char = unicode.ToLower(char)
		}
		out = append(out, char)
	}
	return string(out)
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
