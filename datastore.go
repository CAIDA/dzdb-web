package main

import (
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"os"
	//"strings"

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

func (ds *DataStore) getDomainID(domain string) (int64, int64, error) {
	var id, zone_id int64
	err := ds.db.QueryRow("SELECT id, zone_id FROM domains WHERE domain = $1", domain).Scan(&id, &zone_id)
	if err == sql.ErrNoRows {
		err = ErrNoResource
	}
	return id, zone_id, err
}

func (ds *DataStore) getIPID(ip string) (int64, int, error) {
	var id int64
	var version int = 4
	err := ds.db.QueryRow("SELECT id FROM a WHERE ip = $1", ip).Scan(&id)
	if err == sql.ErrNoRows {
		version = 6
		err = ds.db.QueryRow("SELECT id FROM aaaa WHERE ip = $1", ip).Scan(&id)
		if err == sql.ErrNoRows {
			err = ErrNoResource
		}
	}
	return id, version, err
}

func (ds *DataStore) getZoneID(name string) (int64, error) {
	var id int64
	err := ds.db.QueryRow("select id from zones where zone = $1 limit 1", name).Scan(&id)
	if err == sql.ErrNoRows {
		err = ErrNoResource
	}
	return id, err
}

func (ds *DataStore) getZone(name string) (*Zone, error) {
	var z Zone
	var err error

	z.Id, err = ds.getZoneID(name)
	if err != nil {
		return nil, err
	}
	z.Name = name

	// get first_seen & last_seen
	err = ds.db.QueryRow("select first_seen from zones_nameservers where zone_id = $1 order by first_seen nulls first limit 1", z.Id).Scan(&z.FirstSeen)
	if err != nil {
		return nil, err
	}
	err = ds.db.QueryRow("select last_seen from zones_nameservers where zone_id = $1 order by last_seen nulls first limit 1", z.Id).Scan(&z.LastSeen)
	if err != nil {
		return nil, err
	}

	// get num NS
	err = ds.db.QueryRow("SELECT count(*) FROM zones_nameservers WHERE zone_id = $1 AND last_seen IS NULL", z.Id).Scan(&z.NameServerCount)
	if err != nil {
		return nil, err
	}

	// get num archive NS
	err = ds.db.QueryRow("SELECT count(*) FROM zones_nameservers WHERE zone_id = $1 AND last_seen IS NOT NULL", z.Id).Scan(&z.ArchiveNameServerCount)
	if err != nil {
		return nil, err
	}

	// get active NS
	rows, err := ds.db.Query("SELECT ns.id, ns.domain, zns.first_seen, zns.last_seen FROM zones_nameservers zns, nameservers ns WHERE zns.nameserver_id = ns.id AND zns.last_seen IS NULL AND zns.zone_id = $1 limit 100", z.Id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	z.NameServers = make([]*NameServer, 0, 4)
	for rows.Next() {
		var ns NameServer
		err = rows.Scan(&ns.Id, &ns.Name, &ns.FirstSeen, &ns.LastSeen)
		if err != nil {
			return nil, err
		}
		z.NameServers = append(z.NameServers, &ns)
	}

	// get archive NS
	archive_rows, err := ds.db.Query("SELECT ns.id, ns.domain, zns.first_seen, zns.last_seen FROM zones_nameservers zns, nameservers ns WHERE zns.nameserver_id = ns.id AND zns.last_seen IS NOT NULL AND zns.zone_id = $1 ORDER BY last_seen desc limit 100", z.Id)
	if err != nil {
		return nil, err
	}
	defer archive_rows.Close()
	z.ArchiveNameServers = make([]*NameServer, 0, 4)
	for archive_rows.Next() {
		var ns NameServer
		err = archive_rows.Scan(&ns.Id, &ns.Name, &ns.FirstSeen, &ns.LastSeen)
		if err != nil {
			return nil, err
		}
		z.ArchiveNameServers = append(z.ArchiveNameServers, &ns)
	}

	return &z, err
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
	d.Id, d.Zone.Id, err = ds.getDomainID(domain)
	if err != nil {
		return nil, err
	}
	d.Name = domain

	// zone queries
	err = ds.db.QueryRow("select zones.zone, imports.date from zones, imports where zones.id = imports.zone_id and zones.id = $1 order by date desc limit 1", d.Zone.Id).Scan(&d.Zone.Name, &d.Zone.LastSeen)
	if err != nil {
		return nil, err
	}

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
		err = rows.Scan(&ns.Id, &ns.Name, &ns.FirstSeen, &ns.LastSeen)
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
		err = archive_rows.Scan(&ns.Id, &ns.Name, &ns.FirstSeen, &ns.LastSeen)
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
		err = row.Scan(&domain.Id, &domain.Name)
	}
	if err != nil {
		return nil, err
	}
	return &domain, nil
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
	zirs.Count = len(zirs.Zones)

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
	ns.Name = domain

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
	// TODO times out
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
		err = rows.Scan(&d.Id, &d.Name, &d.FirstSeen, &d.LastSeen)
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
		err = archive_rows.Scan(&d.Id, &d.Name, &d.FirstSeen, &d.LastSeen)
		if err != nil {
			return nil, err
		}
		ns.ArchiveDomains = append(ns.ArchiveDomains, &d)
	}

	// get num IP4
	err = ds.db.QueryRow("SELECT count(*) FROM a_nameservers WHERE nameserver_id = $1 AND last_seen IS NULL", ns.Id).Scan(&ns.IP4Count)
	if err != nil {
		return nil, err
	}

	// get num archive IP4
	err = ds.db.QueryRow("SELECT count(*) FROM a_nameservers WHERE nameserver_id = $1 AND last_seen IS NOT NULL", ns.Id).Scan(&ns.ArchiveIP4Count)
	if err != nil {
		return nil, err
	}

	// get current IP4
	rows, err = ds.db.Query("SELECT ip.id, ip.ip, dns.first_seen, dns.last_seen FROM a_nameservers dns, a ip WHERE ip.id = dns.a_id AND dns.last_seen IS NULL AND dns.nameserver_id = $1 limit 100", ns.Id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ns.IP4 = make([]*IP4, 0, 4)
	for rows.Next() {
		var ip IP4
		err = rows.Scan(&ip.Id, &ip.Name, &ip.FirstSeen, &ip.LastSeen)
		if err != nil {
			return nil, err
		}
		ip.Version = 4
		ns.IP4 = append(ns.IP4, &ip)
	}

	//get archive ipv4
	rows, err = ds.db.Query("SELECT ip.id, ip.ip, dns.first_seen, dns.last_seen FROM a_nameservers dns, a ip WHERE ip.id = dns.a_id AND dns.last_seen IS NOT NULL AND dns.nameserver_id = $1 limit 100", ns.Id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ns.ArchiveIP4 = make([]*IP4, 0, 4)
	for rows.Next() {
		var ip IP4
		err = rows.Scan(&ip.Id, &ip.Name, &ip.FirstSeen, &ip.LastSeen)
		if err != nil {
			return nil, err
		}
		ip.Version = 4
		ns.ArchiveIP4 = append(ns.ArchiveIP4, &ip)
	}

	// get num IP6
	err = ds.db.QueryRow("SELECT count(*) FROM aaaa_nameservers WHERE nameserver_id = $1 AND last_seen IS NULL", ns.Id).Scan(&ns.IP6Count)
	if err != nil {
		return nil, err
	}

	// get num archive IP6
	err = ds.db.QueryRow("SELECT count(*) FROM aaaa_nameservers WHERE nameserver_id = $1 AND last_seen IS NOT NULL", ns.Id).Scan(&ns.ArchiveIP6Count)
	if err != nil {
		return nil, err
	}

	// get current IP6
	rows, err = ds.db.Query("SELECT ip.id, ip.ip, dns.first_seen, dns.last_seen FROM aaaa_nameservers dns, aaaa ip WHERE ip.id = dns.aaaa_id AND dns.last_seen IS NULL AND dns.nameserver_id = $1 limit 100", ns.Id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ns.IP6 = make([]*IP6, 0, 4)
	for rows.Next() {
		var ip IP6
		err = rows.Scan(&ip.Id, &ip.Name, &ip.FirstSeen, &ip.LastSeen)
		if err != nil {
			return nil, err
		}
		ip.Version = 6
		ns.IP6 = append(ns.IP6, &ip)
	}

	//get archive ipv6
	rows, err = ds.db.Query("SELECT ip.id, ip.ip, dns.first_seen, dns.last_seen FROM aaaa_nameservers dns, aaaa ip WHERE ip.id = dns.aaaa_id AND dns.last_seen IS NOT NULL AND dns.nameserver_id = $1 limit 100", ns.Id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ns.ArchiveIP6 = make([]*IP6, 0, 4)
	for rows.Next() {
		var ip IP6
		err = rows.Scan(&ip.Id, &ip.Name, &ip.FirstSeen, &ip.LastSeen)
		if err != nil {
			return nil, err
		}
		ip.Version = 6
		ns.ArchiveIP6 = append(ns.ArchiveIP6, &ip)
	}

	return &ns, nil
}

// gets information for the provided domain
func (ds *DataStore) getIP(name string) (*IP, error) {
	var ip IP
	var err error
	ip.Id, ip.Version, err = ds.getIPID(name)
	if err != nil {
		return nil, err
	}
	ip.Name = name

	if ip.Version == 4 {
		// get first_seen & last_seen
		err = ds.db.QueryRow("select first_seen from a_nameservers where a_id = $1 order by first_seen nulls first limit 1", ip.Id).Scan(&ip.FirstSeen)
		if err != nil {
			return nil, err
		}
		err = ds.db.QueryRow("select last_seen from a_nameservers where a_id = $1 order by last_seen nulls first limit 1", ip.Id).Scan(&ip.LastSeen)
		if err != nil {
			return nil, err
		}

		// get num NS
		err = ds.db.QueryRow("SELECT count(*) FROM a_nameservers WHERE a_id = $1 AND last_seen IS NULL", ip.Id).Scan(&ip.NameServerCount)
		if err != nil {
			return nil, err
		}

		// get num archive NS
		err = ds.db.QueryRow("SELECT count(*) FROM a_nameservers WHERE a_id = $1 AND last_seen IS NOT NULL", ip.Id).Scan(&ip.ArchiveNameServerCount)
		if err != nil {
			return nil, err
		}

		// get current NS
		rows, err := ds.db.Query("SELECT ns.id, ns.domain, dns.first_seen, dns.last_seen FROM a_nameservers dns, nameservers ns WHERE dns.nameserver_id = ns.id AND dns.last_seen IS NULL AND dns.a_id = $1 limit 100", ip.Id)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		ip.NameServers = make([]*NameServer, 0, 4)
		for rows.Next() {
			var ns NameServer
			err = rows.Scan(&ns.Id, &ns.Name, &ns.FirstSeen, &ns.LastSeen)
			if err != nil {
				return nil, err
			}
			ip.NameServers = append(ip.NameServers, &ns)
		}

		// get archive NS
		rows, err = ds.db.Query("SELECT ns.id, ns.domain, dns.first_seen, dns.last_seen FROM a_nameservers dns, nameservers ns WHERE dns.nameserver_id = ns.id AND dns.last_seen IS NOT NULL AND dns.a_id = $1 ORDER BY last_seen desc limit 100", ip.Id)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		ip.ArchiveNameServers = make([]*NameServer, 0, 4)
		for rows.Next() {
			var ns NameServer
			err = rows.Scan(&ns.Id, &ns.Name, &ns.FirstSeen, &ns.LastSeen)
			if err != nil {
				return nil, err
			}
			ip.ArchiveNameServers = append(ip.ArchiveNameServers, &ns)
		}
	} else {
		// get first_seen & last_seen
		err = ds.db.QueryRow("select first_seen from aaaa_nameservers where aaaa_id = $1 order by first_seen nulls first limit 1", ip.Id).Scan(&ip.FirstSeen)
		if err != nil {
			return nil, err
		}
		err = ds.db.QueryRow("select last_seen from aaaa_nameservers where aaaa_id = $1 order by last_seen nulls first limit 1", ip.Id).Scan(&ip.LastSeen)
		if err != nil {
			return nil, err
		}

		// get num NS
		err = ds.db.QueryRow("SELECT count(*) FROM aaaa_nameservers WHERE aaaa_id = $1 AND last_seen IS NULL", ip.Id).Scan(&ip.NameServerCount)
		if err != nil {
			return nil, err
		}

		// get num archive NS
		err = ds.db.QueryRow("SELECT count(*) FROM aaaa_nameservers WHERE aaaa_id = $1 AND last_seen IS NOT NULL", ip.Id).Scan(&ip.ArchiveNameServerCount)
		if err != nil {
			return nil, err
		}

		// get current NS
		rows, err := ds.db.Query("SELECT ns.id, ns.domain, dns.first_seen, dns.last_seen FROM aaaa_nameservers dns, nameservers ns WHERE dns.nameserver_id = ns.id AND dns.last_seen IS NULL AND dns.aaaa_id = $1 limit 100", ip.Id)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		ip.NameServers = make([]*NameServer, 0, 4)
		for rows.Next() {
			var ns NameServer
			err = rows.Scan(&ns.Id, &ns.Name, &ns.FirstSeen, &ns.LastSeen)
			if err != nil {
				return nil, err
			}
			ip.NameServers = append(ip.NameServers, &ns)
		}

		// get archive NS
		rows, err = ds.db.Query("SELECT ns.id, ns.domain, dns.first_seen, dns.last_seen FROM aaaa_nameservers dns, nameservers ns WHERE dns.nameserver_id = ns.id AND dns.last_seen IS NOT NULL AND dns.aaaa_id = $1 ORDER BY last_seen desc limit 100", ip.Id)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		ip.ArchiveNameServers = make([]*NameServer, 0, 4)
		for rows.Next() {
			var ns NameServer
			err = rows.Scan(&ns.Id, &ns.Name, &ns.FirstSeen, &ns.LastSeen)
			if err != nil {
				return nil, err
			}
			ip.ArchiveNameServers = append(ip.ArchiveNameServers, &ns)
		}
	}

	return &ip, nil
}
