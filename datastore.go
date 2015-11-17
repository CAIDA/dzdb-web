package main

import (
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

//TODO prepaired statments

var ErrNoResource = errors.New("The requested object does not exist.")

// connects to the Postgresql database
func getDB(cfg *DatabaseConfig) (*sql.DB, error) {
	os.Clearenv() /* because there is a bug when PGHOSTADDR is set */
	connStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host,
		/*cfg.Port,*/
		cfg.User,
		cfg.Password,
		cfg.Database,
	/*cfg.SSL*/
	)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	/* test connection */
	err = db.Ping()
	if err != nil {
		return db, err
	}
	return db, nil
}

// stores references to the database and
// has methods for querying the database
type DataStore struct {
	db *sql.DB
}

// Creates a new DataStore with the provided database configuration
// connects to the database on creation
func NewDataStore(cfg *DatabaseConfig) (*DataStore, error) {
	db, err := getDB(cfg)
	if err != nil {
		return nil, err
	}
	ds := DataStore{db}
	err = ds.setSQLTimeout(cfg.Timeout)
	return &ds, err
}

// sets the amount of time a SQL query can run before timing out
func (ds *DataStore) setSQLTimeout(sec int) error {
	_, err := ds.db.Exec(fmt.Sprintf("SET statement_timeout TO %d;", (1000 * sec)))
	return err
}

// closes the database connection
func (ds *DataStore) Close() error {
	return ds.db.Close()
}

func (ds *DataStore) getDomainID(domain string) (int64, error) {
	var id int64
	err := ds.db.QueryRow("SELECT id FROM domains WHERE domain = $1", domain).Scan(&id)
	if err == sql.ErrNoRows {
		err = ErrNoResource
	}
	return id, err
}

func (ds *DataStore) getNameServerID(domain string) (int64, error) {
	var id int64
	err := ds.db.QueryRow("SELECT id FROM nameservers WHERE domain = $1", domain).Scan(&id)
	if err == sql.ErrNoRows {
		err = ErrNoResource
	}
	return id, err
}

// gets information for the provided domain
func (ds *DataStore) getDomain(domain string) (*Domain, error) {
	var d Domain
	var err error
	d.Id, err = ds.getDomainID(domain)
	if err != nil {
		return nil, err
	}
	d.Domain = domain

	// get first_seen & last_seen
	err = ds.db.QueryRow("select first_seen from domains_nameservers where domain_id = $1 order by first_seen nulls first limit 1", d.Id).Scan(&d.FirstSeen)
	if err != nil {
		return nil, err
	}
	err = ds.db.QueryRow("select last_seen from domains_nameservers where domain_id = $1 order by last_seen nulls first limit 1", d.Id).Scan(&d.LastSeen)
	if err != nil {
		return nil, err
	}

	// get num NS
	err = ds.db.QueryRow("SELECT count(*) FROM domains_nameservers WHERE domain_id = $1 AND last_seen IS NULL", d.Id).Scan(&d.NameServerCount)
	if err != nil {
		return nil, err
	}

	// get num archive NS
	err = ds.db.QueryRow("SELECT count(*) FROM domains_nameservers WHERE domain_id = $1 AND last_seen IS NOT NULL", d.Id).Scan(&d.ArchiveNameServerCount)
	if err != nil {
		return nil, err
	}

	// get active NS
	rows, err := ds.db.Query("SELECT ns.id, ns.domain, dns.first_seen, dns.last_seen FROM domains_nameservers dns, nameservers ns WHERE dns.nameserver_id = ns.id AND dns.last_seen IS NULL AND dns.domain_id = $1 limit 100", d.Id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	d.NameServers = make([]*NameServer, 0, 4)
	for rows.Next() {
		var ns NameServer
		err = rows.Scan(&ns.Id, &ns.NameServer, &ns.FirstSeen, &ns.LastSeen)
		if err != nil {
			return nil, err
		}
		d.NameServers = append(d.NameServers, &ns)
	}

	// get archive NS
	archive_rows, err := ds.db.Query("SELECT ns.id, ns.domain, dns.first_seen, dns.last_seen FROM domains_nameservers dns, nameservers ns WHERE dns.nameserver_id = ns.id AND dns.last_seen IS NOT NULL AND dns.domain_id = $1 ORDER BY last_seen desc limit 100", d.Id)
	if err != nil {
		return nil, err
	}
	defer archive_rows.Close()
	d.ArchiveNameServers = make([]*NameServer, 0, 4)
	for archive_rows.Next() {
		var ns NameServer
		err = archive_rows.Scan(&ns.Id, &ns.NameServer, &ns.FirstSeen, &ns.LastSeen)
		if err != nil {
			return nil, err
		}
		d.ArchiveNameServers = append(d.ArchiveNameServers, &ns)
	}

	return &d, nil
}

// gets the number of domains in the system
func (ds *DataStore) getDomainCount() (int64, error) {
	row := ds.db.QueryRow("SELECT max(id) from domains;")
	var count int64
	err := row.Scan(&count)
	return count, err
}

// finds a random active domain
func (ds *DataStore) getRandomDomain() (*Domain, error) {
	count, err := ds.getDomainCount()
	if err != nil {
		return nil, err
	}
	var domain Domain
	err = sql.ErrNoRows
	for err == sql.ErrNoRows {
		rid := rand.Int63n(count)
		/* any domain */
		//row := db.QueryRow("Select domain from domains where id = $1", rid)
		/* active domains (slower) */
		row := ds.db.QueryRow("select domains.id, domain from domains, domains_nameservers dns where dns.domain_id = id and domain_id = $1 and last_seen is null limit 1;", rid)
		err = row.Scan(&domain.Id, &domain.Domain)
	}
	if err != nil {
		return nil, err
	}
	return &domain, nil
}

// gets information for the provided nameserver
func (ds *DataStore) getNameServerLegacy(domain string) (*NameServer, error) {
	domain = strings.ToUpper(domain)
	rows, err := ds.db.Query("select domains.domain, dns.first_seen, dns.last_seen from nameservers, domains, domains_nameservers dns, nameservers ns where ns.id = dns.nameserver_id and domains.id = dns.domain_id and dns.nameserver_id = nameservers.id and dns.last_seen is null and nameservers.domain = $1 order by first_seen desc nulls last limit 100", domain)
	if err != nil {
		return nil, err
	}
	//TODO dont return when no object found
	defer rows.Close()
	var data NameServer
	data.NameServer = domain
	data.Domains = make([]*Domain, 0, 4)
	for rows.Next() {
		var result Domain
		err = rows.Scan(&result.Domain, &result.FirstSeen, &result.LastSeen)
		if err != nil {
			return nil, err
		}
		data.Domains = append(data.Domains, &result)
	}
	return &data, nil
}

func (ds *DataStore) getZoneImportResults() (*ZoneImportResults, error) {
	var zirs ZoneImportResults
	zirs.Zones = make([]*ZoneImportResult, 0, 100)

	rows, err := ds.db.Query("select id, date, zone, records, domains, duration, old, moved, new, old_ns, new_ns, old_a, new_a, old_aaaa, new_aaaa from import_progress;")
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var r ZoneImportResult
		err = rows.Scan(&r.Id, &r.Date, &r.Zone, &r.Records, &r.Domains, &r.Duration, &r.Old, &r.Moved, &r.New, &r.NewNs, &r.OldNs, &r.NewA, &r.OldA, &r.NewAaaa, &r.OldAaaa)
		if err != nil {
			return nil, err
		}
		zirs.Zones = append(zirs.Zones, &r)
	}

	return &zirs, nil

}

func (ds *DataStore) getImportProgress() (*ImportProgress, error) {
	var ip ImportProgress
	err := ds.db.QueryRow("select count(*) imports, count(distinct date) days from unimported").Scan(&ip.Imports, &ip.Days)
	if err != nil {
		return nil, err
	}

	rows, err := ds.db.Query("select * from import_date_timer limit $1", len(ip.Dates))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var i int
	for rows.Next() {
		ipd := &ip.Dates[i]
		err = rows.Scan(&ipd.Date, &ipd.Took, &ipd.Count)
		if err != nil {
			return nil, err
		}
		i++
	}

	return &ip, nil
}

// gets information for the provided nameserver
func (ds *DataStore) getNameServer(domain string) (*NameServer, error) {
	var ns NameServer
	var err error
	ns.Id, err = ds.getNameServerID(domain)
	if err != nil {
		return nil, err
	}
	ns.NameServer = domain

	// get first_seen & last_seen
	// times out
	/*err = ds.db.QueryRow("select first_seen from domains_nameservers where nameserver_id = $1 order by first_seen nulls first limit 1", ns.Id).Scan(&ns.FirstSeen)
	if err != nil {
		return nil, err
	}
	err = ds.db.QueryRow("select last_seen from domains_nameservers where nameserver_id = $1 order by last_seen nulls first limit 1", ns.Id).Scan(&ns.LastSeen)
	if err != nil {
		return nil, err
	}*/

	// get num Domains
	// times out
	/*err = ds.db.QueryRow("SELECT count(*) FROM domains_nameservers WHERE nameserver_id = $1 AND last_seen IS NULL", ns.Id).Scan(&ns.DomainCount)
	if err != nil {
		return nil, err
	}

	// get num archive Domains
	err = ds.db.QueryRow("SELECT count(*) FROM domains_nameservers WHERE nameserver_id = $1 AND last_seen IS NOT NULL", ns.Id).Scan(&ns.ArchiveDomainCount)
	if err != nil {
		return nil, err
	}*/

	// get some active Domains
	rows, err := ds.db.Query("SELECT d.id, d.domain, dns.first_seen, dns.last_seen FROM domains_nameservers dns, domains d WHERE d.id = dns.domain_id AND dns.last_seen IS NULL AND dns.nameserver_id = $1 limit 100", ns.Id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ns.Domains = make([]*Domain, 0, 4)
	for rows.Next() {
		var d Domain
		err = rows.Scan(&d.Id, &d.Domain, &d.FirstSeen, &d.LastSeen)
		if err != nil {
			return nil, err
		}
		ns.Domains = append(ns.Domains, &d)
	}

	// get some old Domains
	archive_rows, err := ds.db.Query("SELECT d.id, d.domain, dns.first_seen, dns.last_seen FROM domains_nameservers dns, domains d WHERE d.id = dns.domain_id AND dns.last_seen IS NOT NULL AND dns.nameserver_id = $1 limit 100", ns.Id)
	if err != nil {
		return nil, err
	}
	defer archive_rows.Close()
	ns.ArchiveDomains = make([]*Domain, 0, 4)
	for archive_rows.Next() {
		var d Domain
		err = archive_rows.Scan(&d.Id, &d.Domain, &d.FirstSeen, &d.LastSeen)
		if err != nil {
			return nil, err
		}
		ns.ArchiveDomains = append(ns.ArchiveDomains, &d)
	}

	return &ns, nil
}
