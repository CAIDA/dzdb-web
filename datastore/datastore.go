package datastore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	"vdz-web/model"

	_ "github.com/lib/pq" // for postgresql
	"github.com/teepark/pqinterval"
)

//TODO prepaired statements

// ErrNoResource a 404 for a vdz resource
var ErrNoResource = errors.New("the requested object does not exist")

// connectDB connects to the Postgresql database
func connectDB(ctx context.Context) (*sql.DB, error) {
	db, err := sql.Open("postgres", "")
	if err != nil {
		return nil, err
	}
	// test connection
	err = db.PingContext(ctx)
	if err != nil {
		return db, err
	}
	return db, nil
}

// DataStore stores references to the database and
// has methods for querying the database
type DataStore struct {
	db *sql.DB
}

// New Creates a new DataStore with the provided database configuration
// database connection variables are set from environment variables
func New(ctx context.Context) (*DataStore, error) {
	db, err := connectDB(ctx)
	if err != nil {
		return nil, err
	}
	ds := DataStore{db}
	//err = ds.setSQLTimeout(cfg.Timeout)
	return &ds, err
}

// sets the amount of time a SQL query can run before timing out
func (ds *DataStore) setSQLTimeout(ctx context.Context, sec int) error {
	_, err := ds.db.ExecContext(ctx, fmt.Sprintf("SET statement_timeout TO %d;", (1000*sec)))
	return err
}

// Close closes the database connection
func (ds *DataStore) Close() error {
	return ds.db.Close()
}

// GetDomainID gets the domain's ID and domain's zone's ID
func (ds *DataStore) GetDomainID(ctx context.Context, domain string) (int64, int64, error) {
	var id, zoneID int64
	err := ds.db.QueryRowContext(ctx, "SELECT id, zone_id FROM domains WHERE domain = $1", domain).Scan(&id, &zoneID)
	if err == sql.ErrNoRows {
		err = ErrNoResource
	}
	return id, zoneID, err
}

// GetIPID gets the IPs ID, and the version (4 or 6)
func (ds *DataStore) GetIPID(ctx context.Context, ipStr string) (int64, int, error) {
	var id int64
	var version int
	var err error
	ip := net.ParseIP(ipStr)
	if ip.To4() != nil {
		version = 4
		// TODO use native golang IP types
		// https://github.com/lib/pq/pull/390
		// looks like solution is to use pgx
		err = ds.db.QueryRowContext(ctx, "SELECT id FROM a WHERE ip = $1", ip.String()).Scan(&id)
		if err == sql.ErrNoRows {
			err = ErrNoResource
		}
		return id, version, err
	}
	if ip.To16() != nil {
		version = 6
		err = ds.db.QueryRowContext(ctx, "SELECT id FROM aaaa WHERE ip = $1", ip.String()).Scan(&id)
		if err == sql.ErrNoRows {
			err = ErrNoResource
		}
		return id, version, err
	}
	return -1, 0, ErrNoResource
}

// GetZoneID gets the zoneID with the given name
func (ds *DataStore) GetZoneID(ctx context.Context, name string) (int64, error) {
	var id int64
	err := ds.db.QueryRowContext(ctx, "select id from zones where zone = $1 limit 1", name).Scan(&id)
	if err == sql.ErrNoRows {
		err = ErrNoResource
	}
	return id, err
}

// GetZone gets the Zone with the given name
func (ds *DataStore) GetZone(ctx context.Context, name string) (*model.Zone, error) {
	// TODO support getting NS from root zone?
	var z model.Zone
	var err error

	z.ID, err = ds.GetZoneID(ctx, name)
	if err != nil {
		return nil, err
	}
	z.Name = name

	// get first_seen & last_seen
	err = ds.db.QueryRowContext(ctx, "select first_seen from zones_nameservers where zone_id = $1 order by first_seen asc nulls first limit 1", z.ID).Scan(&z.FirstSeen)
	if err != nil {
		if err == sql.ErrNoRows {
			z.FirstSeen = nil
		} else {
			return nil, err
		}
	}

	err = ds.db.QueryRowContext(ctx, "select last_seen from zones_nameservers where zone_id = $1 order by last_seen desc nulls first limit 1", z.ID).Scan(&z.LastSeen)
	if err != nil {
		if err == sql.ErrNoRows {
			z.LastSeen = nil
		} else {
			return nil, err
		}
	}

	// get num NS
	err = ds.db.QueryRowContext(ctx, "SELECT count(*) FROM zones_nameservers WHERE zone_id = $1 AND last_seen IS NULL", z.ID).Scan(&z.NameServerCount)
	if err != nil {
		return nil, err
	}

	// get num archive NS
	err = ds.db.QueryRowContext(ctx, "SELECT count(*) FROM zones_nameservers WHERE zone_id = $1 AND last_seen IS NOT NULL", z.ID).Scan(&z.ArchiveNameServerCount)
	if err != nil {
		return nil, err
	}

	// get num domains
	err = ds.db.QueryRowContext(ctx, "SELECT count(*) FROM zones_nameservers WHERE zone_id = $1 AND last_seen IS NOT NULL", z.ID).Scan(&z.ArchiveNameServerCount)
	if err != nil {
		return nil, err
	}

	// get active NS
	rows, err := ds.db.QueryContext(ctx, "SELECT ns.ID, ns.domain, zns.first_seen, zns.last_seen FROM zones_nameservers zns, nameservers ns WHERE zns.nameserver_id = ns.ID AND zns.last_seen IS NULL AND zns.zone_id = $1 limit 100", z.ID)
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
	archiveRows, err := ds.db.QueryContext(ctx, "SELECT ns.ID, ns.domain, zns.first_seen, zns.last_seen FROM zones_nameservers zns, nameservers ns WHERE zns.nameserver_id = ns.ID AND zns.last_seen IS NOT NULL AND zns.zone_id = $1 ORDER BY last_seen desc limit 100", z.ID)
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

	return &z, err
}

// GetNameServerID given a nameserver, find its ID
func (ds *DataStore) GetNameServerID(ctx context.Context, domain string) (int64, error) {
	var id int64
	err := ds.db.QueryRowContext(ctx, "SELECT id FROM nameservers WHERE domain = $1", domain).Scan(&id)
	if err == sql.ErrNoRows {
		err = ErrNoResource
	}
	return id, err
}

func (ds *DataStore) GetFeedNew(ctx context.Context, date time.Time) (*model.Feed, error) {
	var f model.Feed
	f.Change = "new"
	var err error
	f.Date = date

	// TODO limit?
	rows, err := ds.db.QueryContext(ctx, "SELECT domain_id, domain from recent_new_domains where date = $1", date)
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

	// TODO limit?
	rows, err := ds.db.QueryContext(ctx, "SELECT domain_id, domain from recent_old_domains where date = $1", date)
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

	// TODO limit?
	rows, err := ds.db.QueryContext(ctx, "SELECT domain_id, domain from recent_moved_domains where date = $1", date)
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

	// TODO limit?
	rows, err := ds.db.QueryContext(ctx, "SELECT nameserver_id, nameserver, version from recent_moved_ns where date = $1", date)
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
			log.Printf("Got NS Feed wth unknown IP version %d for %s\n", v, date)
			// skip unknown versions for now
			continue
		}
	}

	return &f, err
}

func (ds *DataStore) GetFeedNsNew(ctx context.Context, date time.Time) (*model.NSFeed, error) {
	var f model.NSFeed
	f.Change = "new"
	var err error
	f.Date = date

	// TODO limit?
	rows, err := ds.db.QueryContext(ctx, "SELECT nameserver_id, nameserver, version from recent_new_ns where date = $1", date)
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
			log.Printf("Got NS Feed wth unknown IP version %d for %s\n", v, date)
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

	// TODO limit?
	rows, err := ds.db.QueryContext(ctx, "SELECT nameserver_id, nameserver, version from recent_old_ns where date = $1", date)
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
			log.Printf("Got NS Feed wth unknown IP version %d for %s\n", v, date)
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
	if err != nil {
		return nil, err
	}
	d.Name = domain

	// zone queries
	err = ds.db.QueryRowContext(ctx, "select zones.zone, imports.date from zones, imports where zones.ID = imports.zone_id and imports.imported = true and zones.ID = $1 order by date desc limit 1", d.Zone.ID).Scan(&d.Zone.Name, &d.Zone.LastSeen)
	if err != nil {
		return nil, err
	}

	// get first_seen & last_seen
	err = ds.db.QueryRowContext(ctx, "select first_seen from domains_nameservers where domain_id = $1 order by first_seen asc nulls first limit 1", d.ID).Scan(&d.FirstSeen)
	if err != nil {
		return nil, err
	}
	err = ds.db.QueryRowContext(ctx, "select last_seen from domains_nameservers where domain_id = $1 order by last_seen desc nulls first limit 1", d.ID).Scan(&d.LastSeen)
	if err != nil {
		return nil, err
	}

	// get num NS
	err = ds.db.QueryRowContext(ctx, "SELECT count(*) FROM domains_nameservers WHERE domain_id = $1 AND last_seen IS NULL", d.ID).Scan(&d.NameServerCount)
	if err != nil {
		return nil, err
	}

	// get num archive NS
	err = ds.db.QueryRowContext(ctx, "SELECT count(*) FROM domains_nameservers WHERE domain_id = $1 AND last_seen IS NOT NULL", d.ID).Scan(&d.ArchiveNameServerCount)
	if err != nil {
		return nil, err
	}

	// get active NS
	rows, err := ds.db.QueryContext(ctx, "SELECT ns.ID, ns.domain, dns.first_seen, dns.last_seen FROM domains_nameservers dns, nameservers ns WHERE dns.nameserver_id = ns.ID AND dns.last_seen IS NULL AND dns.domain_id = $1 limit 100", d.ID)
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
	archiveRows, err := ds.db.QueryContext(ctx, "SELECT ns.ID, ns.domain, dns.first_seen, dns.last_seen FROM domains_nameservers dns, nameservers ns WHERE dns.nameserver_id = ns.ID AND dns.last_seen IS NOT NULL AND dns.domain_id = $1 ORDER BY last_seen desc limit 100", d.ID)
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
	row := ds.db.QueryRowContext(ctx, "SELECT max(id) from domains;")
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
	err = sql.ErrNoRows
	for err == sql.ErrNoRows {
		rid := rand.Int63n(count)
		/* any domain */
		//row := db.QueryRow("Select domain from domains where id = $1", rid)
		/* active domains (slower) */
		row := ds.db.QueryRowContext(ctx, "select domains.ID, domain from domains, domains_nameservers dns where dns.domain_id = id and domain_id = $1 and last_seen is null limit 1;", rid)
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

	err := ds.db.QueryRowContext(ctx, "select zones.zone, import_zone_counts.domains, import_zone_counts.records, zones_imports.first_import_date, zones_imports.first_import_id, zones_imports.last_import_date,zones_imports.last_import_id, zones_imports.count from zones, zones_imports, import_zone_counts where zones.id = zones_imports.zone_id and zones_imports.last_import_id = import_zone_counts.import_id and zones.zone = $1", zone).Scan(&r.Zone, &r.Domains, &r.Records, &r.FirstImportDate, &r.FirstImportID, &r.LastImportDate, &r.LastImportID, &r.Count)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// GetZoneImportResults gets the most-recent recent ZoneImportResults for every zone (TODO slow?)
func (ds *DataStore) GetZoneImportResults(ctx context.Context) (*model.ZoneImportResults, error) {
	var zoneImportResults model.ZoneImportResults
	zoneImportResults.Zones = make([]*model.ZoneImportResult, 0, 100)

	rows, err := ds.db.QueryContext(ctx, "select zones.zone, import_zone_counts.domains, import_zone_counts.records, zones_imports.first_import_date, zones_imports.first_import_id, zones_imports.last_import_date,zones_imports.last_import_id, zones_imports.count from zones, zones_imports, import_zone_counts where zones.id = zones_imports.zone_id and zones_imports.last_import_id = import_zone_counts.import_id order by zone asc")
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

// GetZoneHistoryCounts returns the counts averages weekly for the past imports for a given zone
func (ds *DataStore) GetZoneHistoryCounts(ctx context.Context, zone string) (*model.ZoneCount, error) {
	var zc model.ZoneCount
	zc.History = make([]*model.ZoneCounts, 0, 100)
	zc.Zone = zone

	rows, err := ds.db.QueryContext(ctx, "select date_trunc('week', date) AS week, floor(AVG(domains)) as domains, floor(AVG(old)) as old, floor(AVG(moved)) as moved, floor(AVG(new)) as new from import_zone_counts, zones where zone_id = zones.id and zones.zone = $1 group by 1 order by 1 desc limit 300", zone)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var c model.ZoneCounts
		err = rows.Scan(&c.Week, &c.Domains, &c.Old, &c.Moved, &c.New)
		if err != nil {
			return nil, err
		}
		zc.History = append(zc.History, &c)
	}

	return &zc, nil
}

// GetImportProgress gets information on the progress of unimported zones
func (ds *DataStore) GetImportProgress(ctx context.Context) (*model.ImportProgress, error) {
	var ip model.ImportProgress
	err := ds.db.QueryRowContext(ctx, "select count(*), count(distinct date) from imports where imported = false").Scan(&ip.Imports, &ip.Days)
	if err != nil {
		return nil, err
	}

	rows, err := ds.db.QueryContext(ctx, "select date, sum(diff_duration) took_diff, sum(import_duration) took_import, count(CASE WHEN imports.imported THEN 1 END) from imports, import_progress where imports.id = import_progress.import_id group by date order by date desc limit $1", len(ip.Dates))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var i int
	var diffDuration, importDuration pqinterval.Interval

	for rows.Next() {
		ipd := &ip.Dates[i]
		err = rows.Scan(&ipd.Date, &diffDuration, &importDuration, &ipd.Count)
		if err != nil {
			return nil, err
		}
		ipd.DiffDuration, err = diffDuration.Duration()
		if err != nil {
			return nil, err
		}
		ipd.ImportDuration, err = importDuration.Duration()
		if err != nil {
			return nil, err
		}
		ipd.DiffDuration = ipd.DiffDuration.Round(time.Second)
		ipd.ImportDuration = ipd.ImportDuration.Round(time.Second)
		i++
	}

	return &ip, nil
}

// GetNameServer gets information for the provided nameserver
func (ds *DataStore) GetNameServer(ctx context.Context, domain string) (*model.NameServer, error) {
	var ns model.NameServer
	var err error
	ns.ID, err = ds.GetNameServerID(ctx, domain)
	if err != nil {
		return nil, err
	}
	ns.Name = domain

	// get NS metadata
	err = ds.db.QueryRowContext(ctx, "select first_seen, last_seen, domains_count, domains_archive_count, a_count, a_archive_count, aaaa_count, aaaa_archive_count from nameserver_metadata where nameserver_id = $1", ns.ID).Scan(&ns.FirstSeen, &ns.LastSeen, &ns.DomainCount, &ns.ArchiveDomainCount, &ns.IP4Count, &ns.ArchiveIP4Count, &ns.IP6Count, &ns.IP6Count)
	if err != nil {
		return nil, err
	}

	// get some active Domains
	rows, err := ds.db.QueryContext(ctx, "SELECT d.ID, d.domain, dns.first_seen, dns.last_seen FROM domains_nameservers dns, domains d WHERE d.ID = dns.domain_id AND dns.last_seen IS NULL AND dns.nameserver_id = $1 limit 100", ns.ID)
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
	archiveRows, err := ds.db.QueryContext(ctx, "SELECT d.ID, d.domain, dns.first_seen, dns.last_seen FROM domains_nameservers dns, domains d WHERE d.ID = dns.domain_id AND dns.last_seen IS NOT NULL AND dns.nameserver_id = $1 limit 100", ns.ID)
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
	rows, err = ds.db.QueryContext(ctx, "SELECT ip.ID, ip.ip, dns.first_seen, dns.last_seen FROM a_nameservers dns, a ip WHERE ip.ID = dns.a_id AND dns.last_seen IS NULL AND dns.nameserver_id = $1 limit 100", ns.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ns.IP4 = make([]*model.IP4, 0, 4)
	for rows.Next() {
		var ip model.IP4
		err = rows.Scan(&ip.ID, &ip.Name, &ip.FirstSeen, &ip.LastSeen)
		if err != nil {
			return nil, err
		}
		ip.Version = 4
		ns.IP4 = append(ns.IP4, &ip)
	}

	//get archive ipv4
	rows, err = ds.db.QueryContext(ctx, "SELECT ip.ID, ip.ip, dns.first_seen, dns.last_seen FROM a_nameservers dns, a ip WHERE ip.ID = dns.a_id AND dns.last_seen IS NOT NULL AND dns.nameserver_id = $1 limit 100", ns.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ns.ArchiveIP4 = make([]*model.IP4, 0, 4)
	for rows.Next() {
		var ip model.IP4
		err = rows.Scan(&ip.ID, &ip.Name, &ip.FirstSeen, &ip.LastSeen)
		if err != nil {
			return nil, err
		}
		ip.Version = 4
		ns.ArchiveIP4 = append(ns.ArchiveIP4, &ip)
	}

	// get current IP6
	rows, err = ds.db.QueryContext(ctx, "SELECT ip.ID, ip.ip, dns.first_seen, dns.last_seen FROM aaaa_nameservers dns, aaaa ip WHERE ip.ID = dns.aaaa_id AND dns.last_seen IS NULL AND dns.nameserver_id = $1 limit 100", ns.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ns.IP6 = make([]*model.IP6, 0, 4)
	for rows.Next() {
		var ip model.IP6
		err = rows.Scan(&ip.ID, &ip.Name, &ip.FirstSeen, &ip.LastSeen)
		if err != nil {
			return nil, err
		}
		ip.Version = 6
		ns.IP6 = append(ns.IP6, &ip)
	}

	//get archive ipv6
	rows, err = ds.db.QueryContext(ctx, "SELECT ip.ID, ip.ip, dns.first_seen, dns.last_seen FROM aaaa_nameservers dns, aaaa ip WHERE ip.ID = dns.aaaa_id AND dns.last_seen IS NOT NULL AND dns.nameserver_id = $1 limit 100", ns.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ns.ArchiveIP6 = make([]*model.IP6, 0, 4)
	for rows.Next() {
		var ip model.IP6
		err = rows.Scan(&ip.ID, &ip.Name, &ip.FirstSeen, &ip.LastSeen)
		if err != nil {
			return nil, err
		}
		ip.Version = 6
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
	ip.Name = name

	if ip.Version == 4 {
		// get first_seen & last_seen
		err = ds.db.QueryRowContext(ctx, "select first_seen from a_nameservers where a_id = $1 order by first_seen asc nulls first limit 1", ip.ID).Scan(&ip.FirstSeen)
		if err != nil {
			return nil, err
		}
		err = ds.db.QueryRowContext(ctx, "select last_seen from a_nameservers where a_id = $1 order by last_seen desc nulls first limit 1", ip.ID).Scan(&ip.LastSeen)
		if err != nil {
			return nil, err
		}

		// get num NS
		err = ds.db.QueryRowContext(ctx, "SELECT count(*) FROM a_nameservers WHERE a_id = $1 AND last_seen IS NULL", ip.ID).Scan(&ip.NameServerCount)
		if err != nil {
			return nil, err
		}

		// get num archive NS
		err = ds.db.QueryRowContext(ctx, "SELECT count(*) FROM a_nameservers WHERE a_id = $1 AND last_seen IS NOT NULL", ip.ID).Scan(&ip.ArchiveNameServerCount)
		if err != nil {
			return nil, err
		}

		// get current NS
		rows, err := ds.db.QueryContext(ctx, "SELECT ns.ID, ns.domain, dns.first_seen, dns.last_seen FROM a_nameservers dns, nameservers ns WHERE dns.nameserver_id = ns.ID AND dns.last_seen IS NULL AND dns.a_id = $1 limit 100", ip.ID)
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
		rows, err = ds.db.QueryContext(ctx, "SELECT ns.ID, ns.domain, dns.first_seen, dns.last_seen FROM a_nameservers dns, nameservers ns WHERE dns.nameserver_id = ns.ID AND dns.last_seen IS NOT NULL AND dns.a_id = $1 ORDER BY last_seen desc limit 100", ip.ID)
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
		err = ds.db.QueryRowContext(ctx, "select first_seen from aaaa_nameservers where aaaa_id = $1 order by first_seen asc nulls first limit 1", ip.ID).Scan(&ip.FirstSeen)
		if err != nil {
			return nil, err
		}
		err = ds.db.QueryRowContext(ctx, "select last_seen from aaaa_nameservers where aaaa_id = $1 order by last_seen desc nulls first limit 1", ip.ID).Scan(&ip.LastSeen)
		if err != nil {
			return nil, err
		}

		// get num NS
		err = ds.db.QueryRowContext(ctx, "SELECT count(*) FROM aaaa_nameservers WHERE aaaa_id = $1 AND last_seen IS NULL", ip.ID).Scan(&ip.NameServerCount)
		if err != nil {
			return nil, err
		}

		// get num archive NS
		err = ds.db.QueryRowContext(ctx, "SELECT count(*) FROM aaaa_nameservers WHERE aaaa_id = $1 AND last_seen IS NOT NULL", ip.ID).Scan(&ip.ArchiveNameServerCount)
		if err != nil {
			return nil, err
		}

		// get current NS
		rows, err := ds.db.QueryContext(ctx, "SELECT ns.ID, ns.domain, dns.first_seen, dns.last_seen FROM aaaa_nameservers dns, nameservers ns WHERE dns.nameserver_id = ns.ID AND dns.last_seen IS NULL AND dns.aaaa_id = $1 limit 100", ip.ID)
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
		rows, err = ds.db.QueryContext(ctx, "SELECT ns.ID, ns.domain, dns.first_seen, dns.last_seen FROM aaaa_nameservers dns, nameservers ns WHERE dns.nameserver_id = ns.ID AND dns.last_seen IS NOT NULL AND dns.aaaa_id = $1 ORDER BY last_seen desc limit 100", ip.ID)
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

// GetAvaiblePrefixes returns avaible prefixes for the queried prefix
func (ds *DataStore) GetAvaiblePrefixes(ctx context.Context, name string) (*model.PrefixList, error) {
	var prefixes model.PrefixList
	var err error
	prefixes.Prefix = name
	prefixes.Domains = make([]model.PrefixResult, 0, 10)

	rows, err := ds.db.QueryContext(ctx, `With available_domains as 
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
			 zones_imports.zone_id 
		  FROM
		  	 zones,
			 zones_imports 
			 LEFT JOIN
				taken 
				ON taken.zone_id = zones_imports.zone_id 
		  WHERE
			 taken.zone_id IS NULL
			 AND zones.id = zones_imports.zone_id
			 AND zones.zone != 'ROOT'
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
	   max(Domains_nameservers.last_seen) last_Seen 
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
		var lastSeen sql.NullTime
		err = rows.Scan(&domain.Domain, &lastSeen)
		if err != nil {
			return nil, err
		}
		if lastSeen.Valid {
			domain.LastSeen = &lastSeen.Time
		}
		prefixes.Domains = append(prefixes.Domains, domain)
	}

	return &prefixes, nil
}
