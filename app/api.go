package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"
	"vdz-web/datastore"
	"vdz-web/server"

	"github.com/julienschmidt/httprouter"
)

// APIStart entry point for starting application
// adds routes to the server so that the correct handlers are registered
func APIStart(app *appContext, vdzServer *server.Server) {
	app.api = make(map[string]string)

	// Adds a method to the router's GET handler but also adds it to the API index map
	// description is the API function description
	addAPI := func(path, description string, fn http.HandlerFunc) {
		re := regexp.MustCompile(":[a-zA-Z0-9_]*")
		paramPath := re.ReplaceAllStringFunc(path, func(s string) string { return fmt.Sprintf("{%s}", s[1:]) })
		if fn == nil {
			fn = server.HandlerNotImplemented
			description = fmt.Sprintf("[WIP] %s", description)
		}
		app.api[description] = paramPath
		vdzServer.Get("/api"+path, fn)
	}

	// imports
	addAPI("/stats/imports", "imports", app.apiImportStatusHandler)
	addAPI("/imports/:year/:month/:day", "import_day_view", nil)
	addAPI("/imports/:year/:month/:day/:zone", "import_day_view_zone", nil)

	// counts
	addAPI("/counts/", "zone_counts", app.apiInternetHistoryCountsHandler)
	addAPI("/counts/zone/:zone", "internet_counts", app.apiZoneHistoryCountsHandler)
	addAPI("/counts/all/", "all_zone_counts", app.apiAllZoneHistoryCountsHandler)

	// zones
	addAPI("/zones", "zones", app.apiLatestZonesHandler)
	addAPI("/zones/:zone", "zone_view", app.apiZoneHandler)
	addAPI("/zones/:zone/import", "zone_import", app.apiZoneImportHandler)
	addAPI("/zones/:zone/nameservers", "zone_nameservers", nil)
	addAPI("/zones/:zone/nameservers/current", "zone_nameservers_current", nil)
	addAPI("/zones/:zone/nameservers/archive", "zone_nameservers_archive", nil)
	addAPI("/zones/:zone/nameservers/archive/page/:page", "zone_nameservers_archive_paged", nil)

	// domains
	addAPI("/random", "random_domain", app.apiRandomDomainHandler)
	addAPI("/domains/:domain", "domain", app.apiDomainHandler)
	addAPI("/domains/:domain/nameservers", "domain_nameservers", nil)
	addAPI("/domains/:domain/nameservers/current", "domain_current_nameservers", nil)
	addAPI("/domains/:domain/nameservers/current/page/:page", "domain_current_nameservers_paged", nil)
	addAPI("/domains/:domain/nameservers/archive", "domain_archive_nameservers", nil)
	addAPI("/domains/:domain/nameservers/archive/page/:page", "domain_archive_nameservers_paged", nil)

	// nameservers
	addAPI("/nameservers/:domain", "nameserver", app.apiNameserverHandler)
	addAPI("/nameservers/:domain/domains", "nameserver_domains", nil)
	addAPI("/nameservers/:domain/domains/current", "nameserver_current_domains", nil)
	addAPI("/nameservers/:domain/domains/current/page/:page", "nameserver_current_domains_paged", nil)
	addAPI("/nameservers/:domain/domains/archive", "nameserver_archive_domains", nil)
	addAPI("/nameservers/:domain/domains/archive/page/:page", "nameserver_archive_domains_paged", nil)

	addAPI("/nameservers/:domain/ip", "nameserver_ips", nil)
	addAPI("/nameservers/:domain/ip/4", "nameserver_ipv4", nil)
	addAPI("/nameservers/:domain/ip/4/current", "nameserver_ipv4_current", nil)
	addAPI("/nameservers/:domain/ip/4/current/page/:page", "nameserver_ipv4_current_paged", nil)
	addAPI("/nameservers/:domain/ip/4/archive", "nameserver_ipv4_archive", nil)
	addAPI("/nameservers/:domain/ip/4/archive/page/:page", "nameserver_ipv4_archive_paged", nil)
	addAPI("/nameservers/:domain/ip/6", "nameserver_ipv6", nil)
	addAPI("/nameservers/:domain/ip/6/current", "nameserver_ipv6_current", nil)
	addAPI("/nameservers/:domain/ip/6/current/page/:page", "nameserver_ipv6_current_paged", nil)
	addAPI("/nameservers/:domain/ip/6/archive", "nameserver_ipv6_archive", nil)
	addAPI("/nameservers/:domain/ip/6/archive/page/:page", "nameserver_ipv6_archive_paged", nil)

	// ipv4 & ipv6
	addAPI("/ip", "ip", nil)
	addAPI("/ip/:ip", "ip_view", app.apiIPHandler)
	addAPI("/ip/:ip/nameservers", "ip_nameservers", nil)
	addAPI("/ip/:ip/nameservers/current", "ip_nameservers_current", nil)
	addAPI("/ip/:ip/nameservers/archive", "ip_nameservers_archive", nil)

	// feeds
	addAPI("/feeds/new", "feeds_new", nil)
	addAPI("/feeds/new/date/:date", "feeds_new_date", app.apiFeedsNewHandler)
	addAPI("/feeds/ns/new/date/:date", "feeds_ns_new_date", app.apiFeedsNsNewHandler)
	//addAPI("/feeds/new/page/:page", "feeds_new_paged", nil)
	//addAPI("/feeds/new/:year/:month/:day", "feeds_new_date", app.apiFeedsNewHandler)
	//addAPI("/feeds/new/:year/:month/:day/page/:page", "feeds_new_date_paged", nil)

	addAPI("/feeds/old", "feeds_old", nil)
	addAPI("/feeds/old/date/:date", "feeds_old_date", app.apiFeedsOldHandler)
	addAPI("/feeds/ns/old/date/:date", "feeds_ns_old_date", app.apiFeedsNsOldHandler)
	//addAPI("/feeds/old/page/:page", "feeds_old_paged", nil)
	//addAPI("/feeds/old/:year/:month/:day", "feeds_old_date", nil)
	//addAPI("/feeds/old/:year/:month/:day/page/:page", "feeds_old_date_paged", nil)

	addAPI("/feeds/moved", "feeds_moved", nil)
	addAPI("/feeds/moved/date/:date", "feeds_moved_date", app.apiFeedsMovedHandler)
	addAPI("/feeds/ns/moved/date/:date", "feeds_ns_moved_date", app.apiFeedsNsMovedHandler)
	//addAPI("/feeds/moved/page/:page", "feeds_moved_paged", nil)
	//addAPI("/feeds/moved/:year/:month/:day", "feeds_moved_date", nil)
	//addAPI("/feeds/moved/:year/:month/:day/page/:page", "feeds_moved_date_paged", nil)

	// research
	addAPI("/research/ipnszonecount/:ip", "ip_ns_zone_count", app.apiIPNsZoneCount)
	addAPI("/research/active_ips/:date", "active_ips", app.apiActiveIPs)

	// API index
	vdzServer.Get("/api/", app.apiIndex)
}

func (app *appContext) apiImportStatusHandler(w http.ResponseWriter, r *http.Request) {
	ip, err := app.ds.GetImportProgress(r.Context())
	if err != nil {
		panic(err)
	}

	server.WriteJSON(w, ip)
}

func (app *appContext) apiLatestZonesHandler(w http.ResponseWriter, r *http.Request) {
	zoneImportResults, err := app.ds.GetZoneImportResults(r.Context())
	if err != nil {
		panic(err)
	}

	server.WriteJSON(w, zoneImportResults)
}

func (app *appContext) apiZoneImportHandler(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	zone := cleanDomain(params.ByName("zone"))
	zoneImportResult, err := app.ds.GetZoneImport(r.Context(), zone)
	if err != nil {
		// TODO handle error no rows found
		panic(err)
	}

	server.WriteJSON(w, zoneImportResult)
}

func (app *appContext) apiFeedsNewHandler(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	date, err := time.Parse("2006-01-02", params.ByName("date"))
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
func (app *appContext) apiFeedsMovedHandler(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	date, err := time.Parse("2006-01-02", params.ByName("date"))
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
	params := httprouter.ParamsFromContext(r.Context())
	date, err := time.Parse("2006-01-02", params.ByName("date"))
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
	params := httprouter.ParamsFromContext(r.Context())
	date, err := time.Parse("2006-01-02", params.ByName("date"))
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
	params := httprouter.ParamsFromContext(r.Context())
	date, err := time.Parse("2006-01-02", params.ByName("date"))
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
	params := httprouter.ParamsFromContext(r.Context())
	date, err := time.Parse("2006-01-02", params.ByName("date"))
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
	params := httprouter.ParamsFromContext(r.Context())
	domain := cleanDomain(params.ByName("domain"))
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
	params := httprouter.ParamsFromContext(r.Context())
	ip := cleanDomain(params.ByName("ip"))
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

func (app *appContext) apiZoneHandler(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	domain := cleanDomain(params.ByName("zone"))
	data, err1 := app.ds.GetZone(r.Context(), domain)
	if err1 != nil {
		if err1 == datastore.ErrNoResource {
			server.WriteJSONError(w, server.ErrResourceNotFound)
			return
		}
		panic(err1)
	}

	server.WriteJSON(w, data)
}

func (app *appContext) apiZoneHistoryCountsHandler(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	zone := cleanDomain(params.ByName("zone"))
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
	params := httprouter.ParamsFromContext(r.Context())
	domain := cleanDomain(params.ByName("domain"))

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
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(app.api)
	if err != nil && err != http.ErrHandlerTimeout {
		panic(err)
	}
}
