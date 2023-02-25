// Package app defines the helpers (web & api) for the frontend web service
package app

import (
	"dnscoffee/datastore"
	"dnscoffee/server"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
	"net"

	"github.com/gorilla/mux"
)

// APIStart entry point for starting application
// adds routes to the server so that the correct handlers are registered
func APIStart(app *appContext, coffeeServer *server.Server) {
	app.api = make(map[string]string)

	// Adds a method to the router's GET handler but also adds it to the API index map
	// description is the API function description
	addAPI := func(path string, params []string, description string, fn http.HandlerFunc) {
		re := regexp.MustCompile(":[a-zA-Z0-9_]*")
		paramPath := re.ReplaceAllStringFunc(path, func(s string) string { return fmt.Sprintf("{%s}", s[1:]) })
		if fn == nil { // hide WIP
			return
			//fn = server.HandlerNotImplemented
			//description = fmt.Sprintf("[WIP] %s", description)
		}
		if params != nil {
			paramPath += "?" + strings.Join(params, "&")
		}
		app.api[description] = paramPath
		coffeeServer.Get("/api"+path, fn)
	}

	// imports

	addAPI("/imports/{year}/{month}/{day}", nil, "import_day_view", nil)
	addAPI("/imports/{year}/{month}/{day}/{zone}", nil, "import_day_view_zone", nil)

	// counts
	addAPI("/counts", nil, "zone_counts", app.apiInternetHistoryCountsHandler)
	addAPI("/counts/zone/{zone}", nil, "internet_counts", app.apiZoneHistoryCountsHandler)
	addAPI("/counts/root", nil, "internet_counts", app.apiZoneHistoryCountsHandler)
	addAPI("/counts/all", nil, "all_zone_counts", app.apiAllZoneHistoryCountsHandler)
	//addAPI("/counts/top", nil, "top_zone_counts", app.apiTopZonesHandler)

	// zones
	addAPI("/root", nil, "zone_view", app.apiZoneHandler)
	addAPI("/zones", nil, "zones", app.apiLatestZonesHandler)
	addAPI("/zones/{zone}", nil, "zone_view", app.apiZoneHandler)
	addAPI("/zones/{zone}/import", nil, "zone_import", app.apiZoneImportHandler)
	addAPI("/zones/{zone}/nameservers", nil, "zone_nameservers", nil)
	addAPI("/zones/{zone}/nameservers/current", nil, "zone_nameservers_current", nil)
	addAPI("/zones/{zone}/nameservers/archive", nil, "zone_nameservers_archive", nil)
	addAPI("/zones/{zone}/nameservers/archive/page/{page}", nil, "zone_nameservers_archive_paged", nil)

	// domains
	addAPI("/random", nil, "random_domain", app.apiRandomDomainHandler)
	addAPI("/domains/{domain}", nil, "domain", app.apiDomainHandler)
	addAPI("/domains/{domain}/nameservers", nil, "domain_nameservers", nil)
	addAPI("/domains/{domain}/nameservers/current", nil, "domain_current_nameservers", nil)
	addAPI("/domains/{domain}/nameservers/current/page/{page}", nil, "domain_current_nameservers_paged", nil)
	addAPI("/domains/{domain}/nameservers/archive", nil, "domain_archive_nameservers", nil)
	addAPI("/domains/{domain}/nameservers/archive/page/{page}", nil, "domain_archive_nameservers_paged", nil)

	// nameservers
	addAPI("/nameservers/{domain}", nil, "nameserver", app.apiNameserverHandler)
	addAPI("/nameservers/{domain}/domains", nil, "nameserver_domains", nil)
	addAPI("/nameservers/{domain}/domains/current", nil, "nameserver_current_domains", nil)
	addAPI("/nameservers/{domain}/domains/current/page/{page}", nil, "nameserver_current_domains_paged", nil)
	addAPI("/nameservers/{domain}/domains/archive", nil, "nameserver_archive_domains", nil)
	addAPI("/nameservers/{domain}/domains/archive/page/{page}", nil, "nameserver_archive_domains_paged", nil)

	addAPI("/nameservers/{domain}/ip", nil, "nameserver_ips", nil)
	addAPI("/nameservers/{domain}/ip/4", nil, "nameserver_ipv4", nil)
	addAPI("/nameservers/{domain}/ip/4/current", nil, "nameserver_ipv4_current", nil)
	addAPI("/nameservers/{domain}/ip/4/current/page/{page}", nil, "nameserver_ipv4_current_paged", nil)
	addAPI("/nameservers/{domain}/ip/4/archive", nil, "nameserver_ipv4_archive", nil)
	addAPI("/nameservers/{domain}/ip/4/archive/page/{page}", nil, "nameserver_ipv4_archive_paged", nil)
	addAPI("/nameservers/{domain}/ip/6", nil, "nameserver_ipv6", nil)
	addAPI("/nameservers/{domain}/ip/6/current", nil, "nameserver_ipv6_current", nil)
	addAPI("/nameservers/{domain}/ip/6/current/page/{page}", nil, "nameserver_ipv6_current_paged", nil)
	addAPI("/nameservers/{domain}/ip/6/archive", nil, "nameserver_ipv6_archive", nil)
	addAPI("/nameservers/{domain}/ip/6/archive/page/{page}", nil, "nameserver_ipv6_archive_paged", nil)

	// ipv4 & ipv6
	addAPI("/ip", []string{"ipprefix={prefix_to_search}"}, "ip", app.apiIPListHandler)
	addAPI("/ip/{ip}", nil, "ip_view", app.apiIPHandler)
	addAPI("/ip/{ip}/nameservers", nil, "ip_nameservers", nil)
	addAPI("/ip/{ip}/nameservers/current", nil, "ip_nameservers_current", nil)
	addAPI("/ip/{ip}/nameservers/archive", nil, "ip_nameservers_archive", nil)

	// feeds
	addAPI("/feeds/new", nil, "feeds_new", nil)
	addAPI("/feeds/new/search/{search}", nil, "feeds_new_search", app.apiFeedsSearchNewHandler)
	addAPI("/feeds/new/date/{date}", nil, "feeds_new_date", app.apiFeedsNewHandler)
	addAPI("/feeds/ns/new/date/{date}", nil, "feeds_ns_new_date", app.apiFeedsNsNewHandler)
	//addAPI("/feeds/new/page/{page}", nil, "feeds_new_paged", nil)
	//addAPI("/feeds/new/{year}/{month}/{day}", nil, "feeds_new_date", app.apiFeedsNewHandler)
	//addAPI("/feeds/new/{year}/{month}/{day}/page/{page}", nil, "feeds_new_date_paged", nil)

	addAPI("/feeds/old", nil, "feeds_old", nil)
	addAPI("/feeds/old/search/{search}", nil, "feeds_old_search", app.apiFeedsSearchOldHandler)
	addAPI("/feeds/old/date/{date}", nil, "feeds_old_date", app.apiFeedsOldHandler)
	addAPI("/feeds/ns/old/date/{date}", nil, "feeds_ns_old_date", app.apiFeedsNsOldHandler)
	//addAPI("/feeds/old/page/{page}", nil, "feeds_old_paged", nil)
	//addAPI("/feeds/old/{year}/{month}/{day}", nil, "feeds_old_date", nil)
	//addAPI("/feeds/old/{year}/{month}/{day}/page/{page}", nil, "feeds_old_date_paged", nil)

	addAPI("/feeds/moved", nil, "feeds_moved", nil)
	addAPI("/feeds/moved/search/{search}", nil, "feeds_moved_search", app.apiFeedsSearchMovedHandler)
	addAPI("/feeds/moved/date/{date}", nil, "feeds_moved_date", app.apiFeedsMovedHandler)
	addAPI("/feeds/ns/moved/date/{date}", nil, "feeds_ns_moved_date", app.apiFeedsNsMovedHandler)
	//addAPI("/feeds/moved/page/{page}", nil, "feeds_moved_paged", nil)
	//addAPI("/feeds/moved/{year}/{month}/{day}", nil, "feeds_moved_date", nil)
	//addAPI("/feeds/moved/{year}/{month}/{day}/page/{page}", nil, "feeds_moved_date_paged", nil)

	// research
	addAPI("/research/ipnszonecount/{ip}", nil, "ip_ns_zone_count", app.apiIPNsZoneCount)
	addAPI("/research/active_ips/{date}", nil, "active_ips", app.apiActiveIPs)

	// API index
//	coffeeServer.Get("/api", app.apiIndex)
}

func (app *appContext) apiLatestZonesHandler(w http.ResponseWriter, r *http.Request) {
	zoneImportResults, err := app.ds.GetZoneImportResults(r.Context())
	if err != nil {
		panic(err)
	}

	server.WriteJSON(w, zoneImportResults)
}

/*
TODO
func (app *appContext) apiTopZonesHandler(w http.ResponseWriter, r *http.Request) {
	zoneImportResults, err := app.ds.GetZoneImportResults(r.Context())
	if err != nil {
		panic(err)
	}

	server.WriteJSON(w, zoneImportResults)
}*/

func (app *appContext) apiZoneImportHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	zone := cleanDomain(params["zone"])
	zoneImportResult, err := app.ds.GetZoneImport(r.Context(), zone)
	if err != nil {
		// TODO handle error no rows found
		panic(err)
	}

	server.WriteJSON(w, zoneImportResult)
}

func (app *appContext) apiFeedsNewHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	date, err := time.Parse("2006-01-02", params["date"])
	if err != nil {
		panic(err)
	}
	data, err := app.ds.GetFeedNew(r.Context(), date)
	if err != nil {
		if err == datastore.ErrNoResource {
			server.WriteJSONError(w, server.ErrResourceNotFound)
			return
		}
		panic(err)
	}

	server.WriteJSON(w, data)
}

func (app *appContext) apiFeedsSearchMovedHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	search := params["search"]
	data, err := app.ds.GetMovedFeedCount(r.Context(), search)
	if err != nil {
		if err == datastore.ErrNoResource {
			server.WriteJSONError(w, server.ErrResourceNotFound)
			return
		}
		panic(err)
	}

	server.WriteJSON(w, data)
}

func (app *appContext) apiFeedsSearchOldHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	search := params["search"]
	data, err := app.ds.GetOldFeedCount(r.Context(), search)
	if err != nil {
		if err == datastore.ErrNoResource {
			server.WriteJSONError(w, server.ErrResourceNotFound)
			return
		}
		panic(err)
	}

	server.WriteJSON(w, data)
}

func (app *appContext) apiFeedsSearchNewHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	search := strings.ToLower(params["search"])
	data, err := app.ds.GetNewFeedCount(r.Context(), search)
	if err != nil {
		if err == datastore.ErrNoResource {
			server.WriteJSONError(w, server.ErrResourceNotFound)
			return
		}
		panic(err)
	}

	server.WriteJSON(w, data)
}

func (app *appContext) apiFeedsMovedHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	date, err := time.Parse("2006-01-02", params["date"])
	if err != nil {
		panic(err)
	}
	data, err := app.ds.GetFeedMoved(r.Context(), date)
	if err != nil {
		if err == datastore.ErrNoResource {
			server.WriteJSONError(w, server.ErrResourceNotFound)
			return
		}
		panic(err)
	}

	server.WriteJSON(w, data)
}

func (app *appContext) apiFeedsOldHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	date, err := time.Parse("2006-01-02", params["date"])
	if err != nil {
		panic(err)
	}
	data, err := app.ds.GetFeedOld(r.Context(), date)
	if err != nil {
		if err == datastore.ErrNoResource {
			server.WriteJSONError(w, server.ErrResourceNotFound)
			return
		}
		panic(err)
	}

	server.WriteJSON(w, data)
}

func (app *appContext) apiFeedsNsNewHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	date, err := time.Parse("2006-01-02", params["date"])
	if err != nil {
		panic(err)
	}
	data, err := app.ds.GetFeedNsNew(r.Context(), date)
	if err != nil {
		if err == datastore.ErrNoResource {
			server.WriteJSONError(w, server.ErrResourceNotFound)
			return
		}
		panic(err)
	}

	server.WriteJSON(w, data)
}
func (app *appContext) apiFeedsNsMovedHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	date, err := time.Parse("2006-01-02", params["date"])
	if err != nil {
		panic(err)
	}
	data, err := app.ds.GetFeedNsMoved(r.Context(), date)
	if err != nil {
		if err == datastore.ErrNoResource {
			server.WriteJSONError(w, server.ErrResourceNotFound)
			return
		}
		panic(err)
	}

	server.WriteJSON(w, data)
}
func (app *appContext) apiFeedsNsOldHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	date, err := time.Parse("2006-01-02", params["date"])
	if err != nil {
		panic(err)
	}
	data, err := app.ds.GetFeedNsOld(r.Context(), date)
	if err != nil {
		if err == datastore.ErrNoResource {
			server.WriteJSONError(w, server.ErrResourceNotFound)
			return
		}
		panic(err)
	}

	server.WriteJSON(w, data)
}

// domainHandler returns domain object for the queried domain
func (app *appContext) apiDomainHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	domain := cleanDomain(params["domain"])
	data, err := app.ds.GetDomain(r.Context(), domain)
	if err != nil {
		if err == datastore.ErrNoResource {
			server.WriteJSONError(w, server.ErrResourceNotFound)
			return
		}
		panic(err)
	}

	server.WriteJSON(w, data)
}

func (app *appContext) apiIPHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	ip := cleanDomain(params["ip"])
	data, err := app.ds.GetIP(r.Context(), ip)
	if err != nil {
		if err == datastore.ErrNoResource {
			server.WriteJSONError(w, server.ErrResourceNotFound)
			return
		}
		panic(err)
	}

	server.WriteJSON(w, data)
}

func (app *appContext) apiIPListHandler(w http.ResponseWriter, r *http.Request) {
	queryVars := r.URL.Query()
	_, ipPrefix, err := net.ParseCIDR(queryVars.Get("ipprefix"))

	data, err := app.ds.GetIPs(r.Context(), ipPrefix)
	if err != nil {
		if err == datastore.ErrNoResource {
			server.WriteJSONError(w, server.ErrResourceNotFound)
			return
		}

		panic(err)
	}

	server.WriteJSON(w, data)
}


func (app *appContext) apiZoneHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	domain := cleanDomain(params["zone"])
	data, err1 := app.ds.GetZone(r.Context(), domain)
	if err1 != nil {
		if err1 == datastore.ErrNoResource {
			server.WriteJSONError(w, server.ErrResourceNotFound)
			return
		}
		panic(err1)
	}
	// add some metadata to the zone response
	importData, err := app.ds.GetZoneImport(r.Context(), domain)
	if err == nil {
		// TODO check for datastore.ErrNoResource and sql.NoRows
		// TODO in fact, make ErrNoResource include? sql.NowRows as well
		data.ImportData = importData
	}
	server.WriteJSON(w, data)
}

func (app *appContext) apiZoneHistoryCountsHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	zone := cleanDomain(params["zone"])
	data, err1 := app.ds.GetZoneHistoryCounts(r.Context(), zone)
	if err1 != nil {
		if err1 == datastore.ErrNoResource {
			server.WriteJSONError(w, server.ErrResourceNotFound)
			return
		}
		panic(err1)
	}

	server.WriteJSON(w, data)
}

func (app *appContext) apiAllZoneHistoryCountsHandler(w http.ResponseWriter, r *http.Request) {
	data, err := app.ds.GetAllZoneHistoryCounts(r.Context())
	if err != nil {
		if err == datastore.ErrNoResource {
			server.WriteJSONError(w, server.ErrResourceNotFound)
			return
		}
		panic(err)
	}

	server.WriteJSON(w, data)
}

func (app *appContext) apiInternetHistoryCountsHandler(w http.ResponseWriter, r *http.Request) {
	data, err := app.ds.GetInternetHistoryCounts(r.Context())
	if err != nil {
		if err == datastore.ErrNoResource {
			server.WriteJSONError(w, server.ErrResourceNotFound)
			return
		}
		panic(err)
	}

	server.WriteJSON(w, data)
}

// randomDomainHandler returns a random domain from the system
func (app *appContext) apiRandomDomainHandler(w http.ResponseWriter, r *http.Request) {
	domain, err := app.ds.GetRandomDomain(r.Context())
	if err != nil {
		panic(err)
	}
	server.WriteJSON(w, domain)
}

// nameserverHandler returns nameserver object for the queried domain
func (app *appContext) apiNameserverHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	domain := cleanDomain(params["domain"])

	data, err1 := app.ds.GetNameServer(r.Context(), domain)
	if err1 != nil {
		//TODO combine common code below
		if err1 == datastore.ErrNoResource {
			server.WriteJSONError(w, server.ErrResourceNotFound)
			return
		}

		panic(err1)
	}

	server.WriteJSON(w, data)
}

// API Index handler
// Displays the map of the API methods available
func (app *appContext) apiIndex(w http.ResponseWriter, req *http.Request) {
	// TODO change to use mux.Get API documentation functions
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(app.api)
	if err != nil && err != http.ErrHandlerTimeout {
		panic(err)
	}
}
