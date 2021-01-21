// Package datastore implements the functions necessary to query the database
package datastore

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

	"dnscoffee/model"

	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// format long SQL querises with
// https://poorsql.com/

// ErrNoResource a 404 for a resource
var ErrNoResource = errors.New("the requested object does not exist")

// DataStore stores references to the database and
// has methods for querying the database
type DataStore struct {
	db *pgxpool.Pool
}

// New Creates a new DataStore with the provided database configuration
// database connection variables are set from environment variables
func New(ctx context.Context) (*DataStore, error) {
	connPoolConfig, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.ConnectConfig(ctx, connPoolConfig)
	if err != nil {
		return nil, err
	}

	// test connection
	poolConn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	conn := poolConn.Conn()
	err = conn.Ping(ctx)
	if err != nil {
		return nil, err
	}
	err = conn.Close(ctx)

	ds := DataStore{pool}
	return &ds, err
}

// Close closes the database connection
func (ds *DataStore) Close() error {
	ds.db.Close()
	return nil
}

// GetDomainID gets the domain's ID and domain's zone's ID
func (ds *DataStore) GetDomainID(ctx context.Context, domain string) (int64, int64, error) {
	var id, zoneID int64
	err := ds.db.QueryRow(ctx, "SELECT id, zone_id FROM domains WHERE domain = $1", domain).Scan(&id, &zoneID)
	if err == pgx.ErrNoRows {
		err = ErrNoResource
	}
	return id, zoneID, err
}

// GetIPID gets the IPs ID, and the version (4 or 6)
func (ds *DataStore) GetIPID(ctx context.Context, ipStr string) (int64, int, error) {
	var id int64
	var version int
	var err error
	if strings.Contains(ipStr, ":") {
		version = 6
		err = ds.db.QueryRow(ctx, "SELECT id FROM aaaa WHERE ip = $1", ipStr).Scan(&id)
		if err == pgx.ErrNoRows {
			err = ErrNoResource
		}
		return id, version, err
	}
	version = 4
	err = ds.db.QueryRow(ctx, "SELECT id FROM a WHERE ip = $1", ipStr).Scan(&id)
	if err == pgx.ErrNoRows {
		err = ErrNoResource
	}
	return id, version, err
}

// GetZoneID gets the zoneID with the given name
func (ds *DataStore) GetZoneID(ctx context.Context, name string) (int64, error) {
	var id int64
	err := ds.db.QueryRow(ctx, "select id from zones where zone = $1 limit 1", name).Scan(&id)
	if err == pgx.ErrNoRows {
		err = ErrNoResource
	}
	return id, err
}

// GetZone gets the Zone with the given name from zones_nameservers
func (ds *DataStore) GetZone(ctx context.Context, name string) (*model.Zone, error) {
	var z model.Zone
	var err error

	z.ID, err = ds.GetZoneID(ctx, name)
	if err != nil {
		return nil, err
	}
	z.Name = name

	// get first_seen & last_seen
	err = ds.db.QueryRow(ctx, "select first_seen from zones_nameservers where zone_id = $1 order by first_seen asc nulls first limit 1", z.ID).Scan(&z.FirstSeen)
	if err != nil {
		if err == pgx.ErrNoRows {
			z.FirstSeen = nil
		} else {
			return nil, err
		}
	}

	err = ds.db.QueryRow(ctx, "select last_seen from zones_nameservers where zone_id = $1 order by last_seen desc nulls first limit 1", z.ID).Scan(&z.LastSeen)
	if err != nil {
		if err == pgx.ErrNoRows {
			z.LastSeen = nil
		} else {
			return nil, err
		}
	}

	// get num NS
	err = ds.db.QueryRow(ctx, "SELECT count(*) FROM zones_nameservers WHERE zone_id = $1 AND last_seen IS NULL", z.ID).Scan(&z.NameServerCount)
	if err != nil {
		return nil, err
	}

	// get num archive NS
	err = ds.db.QueryRow(ctx, "SELECT count(*) FROM zones_nameservers WHERE zone_id = $1 AND last_seen IS NOT NULL", z.ID).Scan(&z.ArchiveNameServerCount)
	if err != nil {
		return nil, err
	}

	// get active NS
	rows, err := ds.db.Query(ctx, "SELECT ns.ID, ns.domain, zns.first_seen, zns.last_seen FROM zones_nameservers zns, nameservers ns WHERE zns.nameserver_id = ns.ID AND zns.last_seen IS NULL AND zns.zone_id = $1 limit 100", z.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	z.NameServers = make([]*model.NameServer, 0, 4)
	for rows.Next() {
		var ns model.NameServer
		err = rows.Scan(&ns.ID, &ns.Name, &ns.FirstSeen, &ns.LastSeen)
		if err != nil {
			return nil, err
		}
		z.NameServers = append(z.NameServers, &ns)
	}

	// get archive NS
	archiveRows, err := ds.db.Query(ctx, "SELECT ns.ID, ns.domain, zns.first_seen, zns.last_seen FROM zones_nameservers zns, nameservers ns WHERE zns.nameserver_id = ns.ID AND zns.last_seen IS NOT NULL AND zns.zone_id = $1 ORDER BY last_seen desc limit 100", z.ID)
	if err != nil {
		return nil, err
	}
	defer archiveRows.Close()
	z.ArchiveNameServers = make([]*model.NameServer, 0, 4)
	for archiveRows.Next() {
		var ns model.NameServer
		err = archiveRows.Scan(&ns.ID, &ns.Name, &ns.FirstSeen, &ns.LastSeen)
		if err != nil {
			return nil, err
		}
		z.ArchiveNameServers = append(z.ArchiveNameServers, &ns)
	}

	// get some root metadata
	var root model.RootZone
	err = ds.db.QueryRow(ctx, "select first_import_date, last_import_date from zone_imports, zones where zones.id = zone_imports.zone_id and zones.zone = ''").Scan(&root.FirstImport, &root.LastImport)
	if err != nil {
		return nil, err
	}
	z.RootImport = &root

	return &z, err
}

// GetNameServerID given a nameserver, find its ID
func (ds *DataStore) GetNameServerID(ctx context.Context, domain string) (int64, error) {
	var id int64
	err := ds.db.QueryRow(ctx, "SELECT id FROM nameservers WHERE domain = $1", domain).Scan(&id)
	if err == pgx.ErrNoRows {
		err = ErrNoResource
	}
	return id, err
}

func (ds *DataStore) GetFeedNew(ctx context.Context, date time.Time) (*model.Feed, error) {
	var f model.Feed
	f.Change = "new"
	var err error
	f.Date = date

	rows, err := ds.db.Query(ctx, "SELECT domain_id, domain from recent_new_domains where date = $1", date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	f.Domains = make([]*model.Domain, 0, 100)
	for rows.Next() {
		var d model.Domain
		err = rows.Scan(&d.ID, &d.Name)
		if err != nil {
			return nil, err
		}
		f.Domains = append(f.Domains, &d)
	}

	return &f, err
}

func (ds *DataStore) GetFeedOld(ctx context.Context, date time.Time) (*model.Feed, error) {
	var f model.Feed
	f.Change = "old"
	var err error
	f.Date = date

	rows, err := ds.db.Query(ctx, "SELECT domain_id, domain from recent_old_domains where date = $1", date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	f.Domains = make([]*model.Domain, 0, 100)
	for rows.Next() {
		var d model.Domain
		err = rows.Scan(&d.ID, &d.Name)
		if err != nil {
			return nil, err
		}
		f.Domains = append(f.Domains, &d)
	}

	return &f, err
}

func (ds *DataStore) GetFeedMoved(ctx context.Context, date time.Time) (*model.Feed, error) {
	var f model.Feed
	f.Change = "moved"
	var err error
	f.Date = date

	rows, err := ds.db.Query(ctx, "SELECT domain_id, domain from recent_moved_domains where date = $1", date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	f.Domains = make([]*model.Domain, 0, 100)
	for rows.Next() {
		var d model.Domain
		err = rows.Scan(&d.ID, &d.Name)
		if err != nil {
			return nil, err
		}
		f.Domains = append(f.Domains, &d)
	}

	return &f, err
}

func (ds *DataStore) GetFeedNsMoved(ctx context.Context, date time.Time) (*model.NSFeed, error) {
	var f model.NSFeed
	f.Change = "moved"
	var err error
	f.Date = date

	rows, err := ds.db.Query(ctx, "SELECT nameserver_id, nameserver, version from recent_moved_ns where date = $1", date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	f.Nameservers4 = make([]*model.NameServer, 0, 10)
	f.Nameservers6 = make([]*model.NameServer, 0, 10)
	for rows.Next() {
		var ns model.NameServer
		var v int
		err = rows.Scan(&ns.ID, &ns.Name, &v)
		if err != nil {
			return nil, err
		}
		if v == 4 {
			f.Nameservers4 = append(f.Nameservers4, &ns)
		} else if v == 6 {
			f.Nameservers6 = append(f.Nameservers6, &ns)
		} else {
			// log this
			log.Printf("Got NS Feed with unknown IP version %d for %s\n", v, date)
			// skip unknown versions for now
			continue
		}
	}

	return &f, err
}

func (ds *DataStore) GetNewFeedCount(ctx context.Context, search string) (*model.FeedCountList, error) {
	list, err := ds.getFeedCount(ctx, "recent_new_domains", search)
	if err != nil {
		return nil, err
	}
	list.Type = "new"
	return list, nil
}

func (ds *DataStore) GetOldFeedCount(ctx context.Context, search string) (*model.FeedCountList, error) {
	list, err := ds.getFeedCount(ctx, "recent_old_domains", search)
	if err != nil {
		return nil, err
	}
	list.Type = "old"
	return list, nil
}

func (ds *DataStore) GetMovedFeedCount(ctx context.Context, search string) (*model.FeedCountList, error) {
	list, err := ds.getFeedCount(ctx, "recent_moved_domains", search)
	if err != nil {
		return nil, err
	}
	list.Type = "moved"
	return list, nil
}

func (ds *DataStore) getFeedCount(ctx context.Context, table, search string) (*model.FeedCountList, error) {
	var fc model.FeedCountList
	fc.Search = search
	var err error

	if len(search) < 4 {
		return nil, fmt.Errorf("search term must be at least %d long", 4)
	}

	search = strings.ToUpper(search)

	// TODO add index here for like substring search
	query := fmt.Sprintf("SELECT date, count(domain) FROM %s where domain like '%%' || $1 || '%%' group by date order by date desc", table)
	rows, err := ds.db.Query(ctx, query, search)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	fc.Counts = make([]model.FeedCount, 0, 20)

	for rows.Next() {
		var f model.FeedCount
		err = rows.Scan(&f.Date, &f.Count)
		if err != nil {
			return nil, err
		}
		fc.Counts = append(fc.Counts, f)
	}

	return &fc, err
}

func (ds *DataStore) GetFeedNsNew(ctx context.Context, date time.Time) (*model.NSFeed, error) {
	var f model.NSFeed
	f.Change = "new"
	var err error
	f.Date = date

	rows, err := ds.db.Query(ctx, "SELECT nameserver_id, nameserver, version from recent_new_ns where date = $1", date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	f.Nameservers4 = make([]*model.NameServer, 0, 10)
	f.Nameservers6 = make([]*model.NameServer, 0, 10)
	for rows.Next() {
		var ns model.NameServer
		var v int
		err = rows.Scan(&ns.ID, &ns.Name, &v)
		if err != nil {
			return nil, err
		}
		if v == 4 {
			f.Nameservers4 = append(f.Nameservers4, &ns)
		} else if v == 6 {
			f.Nameservers6 = append(f.Nameservers6, &ns)
		} else {
			// log this
			log.Printf("Got NS Feed with unknown IP version %d for %s\n", v, date)
			// skip unknown versions for now
			continue
		}
	}

	return &f, err
}

func (ds *DataStore) GetFeedNsOld(ctx context.Context, date time.Time) (*model.NSFeed, error) {
	var f model.NSFeed
	f.Change = "old"
	var err error
	f.Date = date

	rows, err := ds.db.Query(ctx, "SELECT nameserver_id, nameserver, version from recent_old_ns where date = $1", date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	f.Nameservers4 = make([]*model.NameServer, 0, 10)
	f.Nameservers6 = make([]*model.NameServer, 0, 10)
	for rows.Next() {
		var ns model.NameServer
		var v int
		err = rows.Scan(&ns.ID, &ns.Name, &v)
		if err != nil {
			return nil, err
		}
		if v == 4 {
			f.Nameservers4 = append(f.Nameservers4, &ns)
		} else if v == 6 {
			f.Nameservers6 = append(f.Nameservers6, &ns)
		} else {
			// log this
			log.Printf("Got NS Feed with unknown IP version %d for %s\n", v, date)
			// skip unknown versions for now
			continue
		}
	}

	return &f, err
}

// GetDomain gets information for the provided domain
func (ds *DataStore) GetDomain(ctx context.Context, domain string) (*model.Domain, error) {
	var d model.Domain
	var z model.Zone
	d.Zone = &z
	var err error
	d.ID, d.Zone.ID, err = ds.GetDomainID(ctx, domain)
	// TODO err can be ErrNoRows and not ErrNoResource
	// fix here and for other methods too
	if err != nil {
		return nil, err
	}
	d.Name = domain

	// zone queries
	err = ds.db.QueryRow(ctx, "select zones.zone, zone_imports.first_import_date, zone_imports.last_import_date from zones, zone_imports where zones.id = zone_imports.zone_id and zones.id = $1 limit 1;", d.Zone.ID).Scan(&d.Zone.Name, &d.Zone.FirstSeen, &d.Zone.LastSeen)
	if err != nil {
		return nil, err
	}

	// get first_seen & last_seen
	err = ds.db.QueryRow(ctx, "select first_seen from domains_nameservers where domain_id = $1 order by first_seen asc nulls first limit 1", d.ID).Scan(&d.FirstSeen)
	if err != nil {
		return nil, err
	}
	err = ds.db.QueryRow(ctx, "select last_seen from domains_nameservers where domain_id = $1 order by last_seen desc nulls first limit 1", d.ID).Scan(&d.LastSeen)
	if err != nil {
		return nil, err
	}

	// get num NS
	err = ds.db.QueryRow(ctx, "SELECT count(*) FROM domains_nameservers WHERE domain_id = $1 AND last_seen IS NULL", d.ID).Scan(&d.NameServerCount)
	if err != nil {
		return nil, err
	}

	// get num archive NS
	err = ds.db.QueryRow(ctx, "SELECT count(*) FROM domains_nameservers WHERE domain_id = $1 AND last_seen IS NOT NULL", d.ID).Scan(&d.ArchiveNameServerCount)
	if err != nil {
		return nil, err
	}

	// get active NS
	rows, err := ds.db.Query(ctx, "SELECT ns.ID, ns.domain, dns.first_seen, dns.last_seen FROM domains_nameservers dns, nameservers ns WHERE dns.nameserver_id = ns.ID AND dns.last_seen IS NULL AND dns.domain_id = $1 limit 100", d.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	d.NameServers = make([]*model.NameServer, 0, 4)
	for rows.Next() {
		var ns model.NameServer
		err = rows.Scan(&ns.ID, &ns.Name, &ns.FirstSeen, &ns.LastSeen)
		if err != nil {
			return nil, err
		}
		d.NameServers = append(d.NameServers, &ns)
	}

	// get archive NS
	archiveRows, err := ds.db.Query(ctx, "SELECT ns.ID, ns.domain, dns.first_seen, dns.last_seen FROM domains_nameservers dns, nameservers ns WHERE dns.nameserver_id = ns.ID AND dns.last_seen IS NOT NULL AND dns.domain_id = $1 ORDER BY last_seen desc limit 100", d.ID)
	if err != nil {
		return nil, err
	}
	defer archiveRows.Close()
	d.ArchiveNameServers = make([]*model.NameServer, 0, 4)
	for archiveRows.Next() {
		var ns model.NameServer
		err = archiveRows.Scan(&ns.ID, &ns.Name, &ns.FirstSeen, &ns.LastSeen)
		if err != nil {
			return nil, err
		}
		d.ArchiveNameServers = append(d.ArchiveNameServers, &ns)
	}

	return &d, nil
}

// GetDomainCount gets the number of domains in the system (approx)
func (ds *DataStore) GetDomainCount(ctx context.Context) (int64, error) {
	row := ds.db.QueryRow(ctx, "SELECT max(id) from domains;")
	var count int64
	err := row.Scan(&count)
	return count, err
}

// GetRandomDomain finds a random active domain
func (ds *DataStore) GetRandomDomain(ctx context.Context) (*model.Domain, error) {
	count, err := ds.GetDomainCount(ctx)
	if err != nil {
		return nil, err
	}
	var domain model.Domain
	err = pgx.ErrNoRows
	for err == pgx.ErrNoRows {
		rid := rand.Int63n(count)
		row := ds.db.QueryRow(ctx, "select domains.ID, domain from domains, domains_nameservers dns where dns.domain_id = id and domain_id = $1 and last_seen is null limit 1;", rid)
		err = row.Scan(&domain.ID, &domain.Name)
	}
	if err != nil {
		return nil, err
	}
	return &domain, nil
}

// GetZoneImport gets the most-recent recent ZoneImportResult for the given zone
func (ds *DataStore) GetZoneImport(ctx context.Context, zone string) (*model.ZoneImportResult, error) {
	var r model.ZoneImportResult
	err := ds.db.QueryRow(ctx,
		`SELECT
			zones.zone,
			import_counts.domains,
			import_counts.records,
			zone_imports.first_import_date,
			zone_imports.first_import_id,
			zone_imports.last_import_date,
			zone_imports.last_import_id,
			zone_imports.count
		from
			zones,
			zone_imports,
			import_counts
		where
			zones.id = zone_imports.zone_id
			and zone_imports.last_import_id = import_counts.import_id
			and zones.zone = $1`,
		zone).Scan(&r.Zone, &r.Domains, &r.Records, &r.FirstImportDate, &r.FirstImportID, &r.LastImportDate, &r.LastImportID, &r.Count)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// GetZoneImportResults gets the most-recent recent ZoneImportResults for every zone
func (ds *DataStore) GetZoneImportResults(ctx context.Context) (*model.ZoneImportResults, error) {
	var zoneImportResults model.ZoneImportResults
	zoneImportResults.Zones = make([]*model.ZoneImportResult, 0, 100)

	rows, err := ds.db.Query(ctx, "select zones.zone, import_counts.domains, import_counts.records, zone_imports.first_import_date, zone_imports.first_import_id, zone_imports.last_import_date,zone_imports.last_import_id, zone_imports.count from zones, zone_imports, import_counts where zones.id = zone_imports.zone_id and zone_imports.last_import_id = import_counts.import_id order by zone asc")
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var r model.ZoneImportResult
		err = rows.Scan(&r.Zone, &r.Domains, &r.Records, &r.FirstImportDate, &r.FirstImportID, &r.LastImportDate, &r.LastImportID, &r.Count)
		if err != nil {
			return nil, err
		}
		zoneImportResults.Zones = append(zoneImportResults.Zones, &r)
	}
	zoneImportResults.Count = len(zoneImportResults.Zones)

	return &zoneImportResults, nil
}

// GetInternetHistoryCounts returns the counts averages weekly for the past imports for all zones
func (ds *DataStore) GetInternetHistoryCounts(ctx context.Context) (*model.ZoneCount, error) {
	var zc model.ZoneCount
	zc.History = make([]*model.ZoneCounts, 0, 100)
	zc.Zone = ""
	limit := 300

	rows, err := ds.db.Query(ctx, "with s as (select date, sum(domains) as domains, sum(old) as old, sum(moved) as moved, sum(new) as new from weighted_counts where date not in (select distinct date from imports where imported = false) group by 1 order by 1 desc limit (52 * $1)) select date_trunc('week', date) AS week, floor(AVG(domains)) as domains, sum(old) as old, sum(moved) as moved, sum(new) as new from s group by 1 order by 1 desc limit $1", limit)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var c model.ZoneCounts
		err = rows.Scan(&c.Date, &c.Domains, &c.Old, &c.Moved, &c.New)
		if err != nil {
			return nil, err
		}
		zc.History = append(zc.History, &c)
	}

	return &zc, nil
}

// GetZoneHistoryCounts returns the counts averages weekly for the past imports for a given zone
func (ds *DataStore) GetZoneHistoryCounts(ctx context.Context, zone string) (*model.ZoneCount, error) {
	var zc model.ZoneCount
	zc.History = make([]*model.ZoneCounts, 0, 100)
	zc.Zone = zone

	rows, err := ds.db.Query(ctx, `with g as (
		select 
		  (row_number() over (order by date desc) - 1) / 7 as seqnum, 
		  date, 
		  domains, 
		  feed_old, 
		  feed_moved, 
		  feed_new 
		from 
		  import_counts, 
		  zones 
		where 
		  zone_id = zones.id 
		  and zones.zone = $1 
		limit 
		  7 * 52 * 5
	  ) 
	  select 
		max(date) as date, 
		floor(AVG(domains)) as domains, 
		sum(feed_old) as old, 
		sum(feed_moved) as moved, 
		sum(feed_new) as new 
	  from 
		g 
	  group by 
		seqnum 
	  order by 
		1 desc`, zone)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var c model.ZoneCounts
		err = rows.Scan(&c.Date, &c.Domains, &c.Old, &c.Moved, &c.New)
		if err != nil {
			return nil, err
		}
		zc.History = append(zc.History, &c)
	}

	return &zc, nil
}

// GetAllZoneHistoryCounts returns the counts averages monthly for the past imports for all zones
func (ds *DataStore) GetAllZoneHistoryCounts(ctx context.Context) (*model.AllZoneCounts, error) {
	var all model.AllZoneCounts
	all.Counts = make(map[string]*model.ZoneCount)

	rows, err := ds.db.Query(ctx, "select date_trunc('month', date) AS month, zone, floor(AVG(domains)) as domains, sum(old) as old, sum(moved) as moved, sum(new) as new from weighted_counts, zones where zones.id = zone_id group by 1, 2 order by 1 desc, 2 limit 300 * 1200")
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var c model.ZoneCounts
		var zone string
		err = rows.Scan(&c.Date, &zone, &c.Domains, &c.Old, &c.Moved, &c.New)
		if err != nil {
			return nil, err
		}
		if _, ok := all.Counts[zone]; !ok {
			all.Counts[zone] = new(model.ZoneCount)
			all.Counts[zone].Zone = zone
			all.Counts[zone].History = make([]*model.ZoneCounts, 0, 100)
		}

		all.Counts[zone].History = append(all.Counts[zone].History, &c)
	}

	return &all, nil
}

// GetImportProgress gets information on the progress of unimported zones
func (ds *DataStore) GetImportProgress(ctx context.Context) (*model.ImportProgress, error) {
	history := 60
	var ip model.ImportProgress
	err := ds.db.QueryRow(ctx, "select count(*), count(distinct date) from imports where imported = false").Scan(&ip.Imports, &ip.Days)
	if err != nil {
		return nil, err
	}

	// get diffs remaining
	// this only gets forward counting diffs
	err = ds.db.QueryRow(ctx,
		`select
		count(id)
	  from
		imports,
		import_progress m1
	  where
		m1.import_id = id
		and imported = false
		and m1.zonediff_path is null
		and m1.zonefile_path is not null`).Scan(&ip.Diffs)
	if err != nil {
		return nil, err
	}

	rows, err := ds.db.Query(ctx, "select date, sum(coalesce(diff_duration, '0'::interval)) took_diff, sum(coalesce(import_duration,'0'::interval)) took_import, count(CASE WHEN imports.imported THEN 1 END) from imports, import_progress where imports.id = import_progress.import_id group by date order by date desc limit $1", history)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var diffDuration, importDuration pgtype.Interval
	ip.Dates = make([]model.ImportDate, 0, history)

	for rows.Next() {
		var ipd model.ImportDate
		err = rows.Scan(&ipd.Date, &diffDuration, &importDuration, &ipd.Count)
		if err != nil {
			return nil, err
		}
		err = diffDuration.AssignTo(&ipd.DiffDuration)
		if err != nil {
			return nil, err
		}
		err = importDuration.AssignTo(&ipd.ImportDuration)
		if err != nil {
			return nil, err
		}
		ipd.DiffDuration = ipd.DiffDuration.Round(time.Second)
		ipd.ImportDuration = ipd.ImportDuration.Round(time.Second)
		ip.Dates = append(ip.Dates, ipd)
	}

	return &ip, nil
}

// GetNameServer gets information for the provided nameserver
func (ds *DataStore) GetNameServer(ctx context.Context, domain string) (*model.NameServer, error) {
	var ns model.NameServer
	var z model.Zone

	var err error
	ns.ID, err = ds.GetNameServerID(ctx, domain)
	if err != nil {
		return nil, err
	}
	ns.Name = domain

	// get NS metadata
	err = ds.db.QueryRow(ctx, "select first_seen, last_seen, domains_count, domains_archive_count, a_count, a_archive_count, aaaa_count, aaaa_archive_count from nameserver_metadata where nameserver_id = $1", ns.ID).Scan(&ns.FirstSeen, &ns.LastSeen, &ns.DomainCount, &ns.ArchiveDomainCount, &ns.IP4Count, &ns.ArchiveIP4Count, &ns.IP6Count, &ns.ArchiveIP6Count)
	if err != nil {
		return nil, err
	}

	// get some active Domains
	rows, err := ds.db.Query(ctx, "SELECT d.ID, d.domain, dns.first_seen, dns.last_seen FROM domains_nameservers dns, domains d WHERE d.ID = dns.domain_id AND dns.last_seen IS NULL AND dns.nameserver_id = $1 limit 100", ns.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ns.Domains = make([]*model.Domain, 0, 4)
	for rows.Next() {
		var d model.Domain
		err = rows.Scan(&d.ID, &d.Name, &d.FirstSeen, &d.LastSeen)
		if err != nil {
			return nil, err
		}
		ns.Domains = append(ns.Domains, &d)
	}

	// get some old Domains
	archiveRows, err := ds.db.Query(ctx, "SELECT d.ID, d.domain, dns.first_seen, dns.last_seen FROM domains_nameservers dns, domains d WHERE d.ID = dns.domain_id AND dns.last_seen IS NOT NULL AND dns.nameserver_id = $1 limit 100", ns.ID)
	if err != nil {
		return nil, err
	}
	defer archiveRows.Close()
	ns.ArchiveDomains = make([]*model.Domain, 0, 4)
	for archiveRows.Next() {
		var d model.Domain
		err = archiveRows.Scan(&d.ID, &d.Name, &d.FirstSeen, &d.LastSeen)
		if err != nil {
			return nil, err
		}
		ns.ArchiveDomains = append(ns.ArchiveDomains, &d)
	}

	// get current IP4
	rows, err = ds.db.Query(ctx, "SELECT ip.ID, ip.ip, dns.first_seen, dns.last_seen FROM a_nameservers dns, a ip WHERE ip.ID = dns.a_id AND dns.last_seen IS NULL AND dns.nameserver_id = $1 limit 100", ns.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ns.IP4 = make([]*model.IP4, 0, 4)
	for rows.Next() {
		var ip model.IP4
		err = rows.Scan(&ip.ID, &ip.IP.IP, &ip.FirstSeen, &ip.LastSeen)
		if err != nil {
			return nil, err
		}
		ip.Version = 4
		ip.Name = ip.IPString()
		ns.IP4 = append(ns.IP4, &ip)
	}

	//get archive ipv4
	rows, err = ds.db.Query(ctx, "SELECT ip.ID, ip.ip, ans.first_seen, ans.last_seen FROM a_nameservers ans, a ip WHERE ip.ID = ans.a_id AND ans.last_seen IS NOT NULL AND ans.nameserver_id = $1 limit 100", ns.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ns.ArchiveIP4 = make([]*model.IP4, 0, 4)
	for rows.Next() {
		var ip model.IP4
		err = rows.Scan(&ip.ID, &ip.IP.IP, &ip.FirstSeen, &ip.LastSeen)
		if err != nil {
			return nil, err
		}
		ip.Version = 4
		ip.Name = ip.IPString()
		ns.ArchiveIP4 = append(ns.ArchiveIP4, &ip)
	}

	// get zone_id for nameserver
	// we need the zone_id for the IP Timeline
	err = ds.db.QueryRow(ctx, "SELECT ans.zone_id FROM a_nameservers ans WHERE ans.nameserver_id = $1 limit 1", ns.ID).Scan(&z.ID)
	if err != nil {
		// If we do not have an glue record for the nameserver
		// then there is no timeline so we do not need to worry
		// about the zone_id
		if err == pgx.ErrNoRows {
			z.ID = 0
		} else {
			return nil, err
		}
	}

	// If we do not import the nameserver zone then
	// we do not worry about populating the Zone fields
	if z.ID != 0 {
		err = ds.db.QueryRow(ctx, "select zones.zone, zone_imports.first_import_date, zone_imports.last_import_date from zones, zone_imports where zones.id = zone_imports.zone_id and zones.id = $1 limit 1", z.ID).Scan(&z.Name, &z.FirstSeen, &z.LastSeen)
		if err != nil {
			return nil, err
		}
		ns.Zone = &z
	}

	// get current IP6
	rows, err = ds.db.Query(ctx, "SELECT ip.ID, ip.ip, dns.first_seen, dns.last_seen FROM aaaa_nameservers dns, aaaa ip WHERE ip.ID = dns.aaaa_id AND dns.last_seen IS NULL AND dns.nameserver_id = $1 limit 100", ns.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ns.IP6 = make([]*model.IP6, 0, 4)
	for rows.Next() {
		var ip model.IP6
		err = rows.Scan(&ip.ID, &ip.IP.IP, &ip.FirstSeen, &ip.LastSeen)
		if err != nil {
			return nil, err
		}
		ip.Version = 6
		ip.Name = ip.IPString()
		ns.IP6 = append(ns.IP6, &ip)
	}

	//get archive ipv6
	rows, err = ds.db.Query(ctx, "SELECT ip.ID, ip.ip, dns.first_seen, dns.last_seen FROM aaaa_nameservers dns, aaaa ip WHERE ip.ID = dns.aaaa_id AND dns.last_seen IS NOT NULL AND dns.nameserver_id = $1 limit 100", ns.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ns.ArchiveIP6 = make([]*model.IP6, 0, 4)
	for rows.Next() {
		var ip model.IP6
		err = rows.Scan(&ip.ID, &ip.IP.IP, &ip.FirstSeen, &ip.LastSeen)
		if err != nil {
			return nil, err
		}
		ip.Version = 6
		ip.Name = ip.IPString()
		ns.ArchiveIP6 = append(ns.ArchiveIP6, &ip)
	}

	return &ns, nil
}

// GetIP gets information for the provided IP
func (ds *DataStore) GetIP(ctx context.Context, name string) (*model.IP, error) {
	var ip model.IP
	var err error
	ip.ID, ip.Version, err = ds.GetIPID(ctx, name)
	if err != nil {
		return nil, err
	}
	netIP := net.ParseIP(name)
	if netIP == nil {
		return nil, fmt.Errorf("unable top parse IP %s", name)
	}
	ip.IP = &netIP
	ip.Name = ip.IPString()

	if ip.Version == 4 {
		// get first_seen & last_seen
		err = ds.db.QueryRow(ctx, "select first_seen from a_nameservers where a_id = $1 order by first_seen asc nulls first limit 1", ip.ID).Scan(&ip.FirstSeen)
		if err != nil {
			return nil, err
		}
		err = ds.db.QueryRow(ctx, "select last_seen from a_nameservers where a_id = $1 order by last_seen desc nulls first limit 1", ip.ID).Scan(&ip.LastSeen)
		if err != nil {
			return nil, err
		}

		// get num NS
		err = ds.db.QueryRow(ctx, "SELECT count(*) FROM a_nameservers WHERE a_id = $1 AND last_seen IS NULL", ip.ID).Scan(&ip.NameServerCount)
		if err != nil {
			return nil, err
		}

		// get num archive NS
		err = ds.db.QueryRow(ctx, "SELECT count(*) FROM a_nameservers WHERE a_id = $1 AND last_seen IS NOT NULL", ip.ID).Scan(&ip.ArchiveNameServerCount)
		if err != nil {
			return nil, err
		}

		// get current NS
		rows, err := ds.db.Query(ctx, "SELECT ns.ID, ns.domain, dns.first_seen, dns.last_seen FROM a_nameservers dns, nameservers ns WHERE dns.nameserver_id = ns.ID AND dns.last_seen IS NULL AND dns.a_id = $1 limit 100", ip.ID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		ip.NameServers = make([]*model.NameServer, 0, 4)
		for rows.Next() {
			var ns model.NameServer
			err = rows.Scan(&ns.ID, &ns.Name, &ns.FirstSeen, &ns.LastSeen)
			if err != nil {
				return nil, err
			}
			ip.NameServers = append(ip.NameServers, &ns)
		}

		// get archive NS
		rows, err = ds.db.Query(ctx, "SELECT ns.ID, ns.domain, dns.first_seen, dns.last_seen FROM a_nameservers dns, nameservers ns WHERE dns.nameserver_id = ns.ID AND dns.last_seen IS NOT NULL AND dns.a_id = $1 ORDER BY last_seen desc limit 100", ip.ID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		ip.ArchiveNameServers = make([]*model.NameServer, 0, 4)
		for rows.Next() {
			var ns model.NameServer
			err = rows.Scan(&ns.ID, &ns.Name, &ns.FirstSeen, &ns.LastSeen)
			if err != nil {
				return nil, err
			}
			ip.ArchiveNameServers = append(ip.ArchiveNameServers, &ns)
		}
	} else {
		// get first_seen & last_seen
		err = ds.db.QueryRow(ctx, "select first_seen from aaaa_nameservers where aaaa_id = $1 order by first_seen asc nulls first limit 1", ip.ID).Scan(&ip.FirstSeen)
		if err != nil {
			return nil, err
		}
		err = ds.db.QueryRow(ctx, "select last_seen from aaaa_nameservers where aaaa_id = $1 order by last_seen desc nulls first limit 1", ip.ID).Scan(&ip.LastSeen)
		if err != nil {
			return nil, err
		}

		// get num NS
		err = ds.db.QueryRow(ctx, "SELECT count(*) FROM aaaa_nameservers WHERE aaaa_id = $1 AND last_seen IS NULL", ip.ID).Scan(&ip.NameServerCount)
		if err != nil {
			return nil, err
		}

		// get num archive NS
		err = ds.db.QueryRow(ctx, "SELECT count(*) FROM aaaa_nameservers WHERE aaaa_id = $1 AND last_seen IS NOT NULL", ip.ID).Scan(&ip.ArchiveNameServerCount)
		if err != nil {
			return nil, err
		}

		// get current NS
		rows, err := ds.db.Query(ctx, "SELECT ns.ID, ns.domain, dns.first_seen, dns.last_seen FROM aaaa_nameservers dns, nameservers ns WHERE dns.nameserver_id = ns.ID AND dns.last_seen IS NULL AND dns.aaaa_id = $1 limit 100", ip.ID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		ip.NameServers = make([]*model.NameServer, 0, 4)
		for rows.Next() {
			var ns model.NameServer
			err = rows.Scan(&ns.ID, &ns.Name, &ns.FirstSeen, &ns.LastSeen)
			if err != nil {
				return nil, err
			}
			ip.NameServers = append(ip.NameServers, &ns)
		}

		// get archive NS
		rows, err = ds.db.Query(ctx, "SELECT ns.ID, ns.domain, dns.first_seen, dns.last_seen FROM aaaa_nameservers dns, nameservers ns WHERE dns.nameserver_id = ns.ID AND dns.last_seen IS NOT NULL AND dns.aaaa_id = $1 ORDER BY last_seen desc limit 100", ip.ID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		ip.ArchiveNameServers = make([]*model.NameServer, 0, 4)
		for rows.Next() {
			var ns model.NameServer
			err = rows.Scan(&ns.ID, &ns.Name, &ns.FirstSeen, &ns.LastSeen)
			if err != nil {
				return nil, err
			}
			ip.ArchiveNameServers = append(ip.ArchiveNameServers, &ns)
		}
	}

	return &ip, nil
}

// GetAvailablePrefixes returns available prefixes for the queried prefix
func (ds *DataStore) GetAvailablePrefixes(ctx context.Context, name string) (*model.PrefixList, error) {
	var prefixes model.PrefixList
	var err error
	prefixes.Active = false
	prefixes.Prefix = name
	prefixes.Domains = make([]model.PrefixResult, 0, 10)

	rows, err := ds.db.Query(ctx, `With available_domains as 
	(
	   WITH available AS 
	   (
		  WITH taken AS 
		  (
			 SELECT DISTINCT
				domains.zone_id 
			 FROM
				domains,
				domains_nameservers 
			 WHERE
				domains.id = domains_nameservers.domain_id 
				AND domains_nameservers.last_seen IS NULL 
				AND domains.domain LIKE $1 || '.%'
		  )
		  SELECT
			 zone_imports.zone_id 
		  FROM
		  	 zones,
			 zone_imports 
			 LEFT JOIN
				taken 
				ON taken.zone_id = zone_imports.zone_id 
		  WHERE
			 taken.zone_id IS NULL
			 AND zones.id = zone_imports.zone_id
			 AND zones.zone != ''
			 AND zones.zone != 'ARPA'
	   )
	   SELECT
		  $1 || '.' || zones.zone AS domain 
	   FROM
		  zones,
		  available 
	   WHERE
		  available.zone_id = zones.id 
	)
	Select
	   available_domains.domain,
	   max(Domains_nameservers.last_seen) last_seen 
	from
	   available_domains 
	   Left join
		  domains 
		  on domains.domain = available_domains.domain 
	   Left join
		  domains_nameservers 
		  on domains.id = domains_nameservers.domain_id 
	Group by
	   available_domains.domain 
	ORDER BY
	   Char_length(available_domains.domain),
	   1,  2`, name)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var domain model.PrefixResult
		var lastSeen pgtype.Date
		err = rows.Scan(&domain.Domain, &lastSeen)
		if err != nil {
			return nil, err
		}
		if lastSeen.Status == pgtype.Present {
			domain.LastSeen = &lastSeen.Time
		}
		prefixes.Domains = append(prefixes.Domains, domain)
	}

	return &prefixes, nil
}

// GetTakenPrefixes searched for domain prefixes that match the given pattern that are active
func (ds *DataStore) GetTakenPrefixes(ctx context.Context, name string) (*model.PrefixList, error) {
	var prefixes model.PrefixList
	var err error
	prefixes.Prefix = name
	prefixes.Active = true
	prefixes.Domains = make([]model.PrefixResult, 0, 10)
	rows, err := ds.db.Query(ctx, "select domains.domain, min(domains_nameservers.first_seen) first_seen from domains, domains_nameservers where domains.id = domains_nameservers.domain_id and domain LIKE $1 || '.%' and last_seen is null group by domains.domain order by domains.domain", prefixes.Prefix)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var domain model.PrefixResult
		var firstSeen pgtype.Date
		err = rows.Scan(&domain.Domain, &firstSeen)
		if err != nil {
			return nil, err
		}
		if firstSeen.Status == pgtype.Present {
			domain.FirstSeen = &firstSeen.Time
		}
		prefixes.Domains = append(prefixes.Domains, domain)
	}
	return &prefixes, nil
}

// GetDeadTLDs returns zones that have been removed from the root and their ages
func (ds *DataStore) GetDeadTLDs(ctx context.Context) ([]*model.TLDLife, error) {
	out := make([]*model.TLDLife, 0, 20)

	rows, err := ds.db.Query(ctx, `with dead_zones as (SELECT zones.zone, zones.id FROM zones WHERE NOT (EXISTS ( SELECT zones_nameservers.zone_id FROM zones_nameservers WHERE zones.id = zones_nameservers.zone_id AND zones_nameservers.last_seen IS NULL)))
	select zone,
		min(first_seen) as created,
		max(last_Seen) as removed,
		age(max(last_seen), min(first_seen))::text as age,
		max(domains) as domains
	from dead_zones, zones_nameservers
	left join import_counts
		on import_counts.zone_id = zones_nameservers.zone_id
	where
		dead_zones.id = zones_nameservers.zone_id
	group by zone
	order by 3 desc, 2 asc, 1`)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var t model.TLDLife
		err = rows.Scan(&t.Zone, &t.Created, &t.Removed, &t.Age, &t.Domains)
		if err != nil {
			return nil, err
		}
		out = append(out, &t)
	}

	return out, nil
}

// GetDomainsInZoneID returns a sample of 50 domains in a given zoneID
// note: when joining with zones to turn the zone into a zone ID it is extremely slow
// useing a zoneId is fast
func (ds *DataStore) GetDomainsInZoneID(ctx context.Context, zoneID int64) ([]model.Domain, error) {
	out := make([]model.Domain, 0, 50)
	rows, err := ds.db.Query(ctx, "with dupes as (select domain, last_seen from domains, domains_nameservers, zones where domains.id = domains_nameservers.domain_id and domains_nameservers.zone_id = zones.id and zones.id = $1 order by last_Seen desc limit 150) select domain, max(last_seen) last_seen from dupes group by domain limit 50", zoneID)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var d model.Domain
		err = rows.Scan(&d.Name, &d.LastSeen)
		if err != nil {
			return out, err
		}
		out = append(out, d)
	}
	return out, nil
}
