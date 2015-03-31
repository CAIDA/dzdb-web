package main

import (
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

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
	return &ds, nil
}

// closes the database connection
func (ds *DataStore) Close() error {
	return ds.db.Close()
}

// gets information for the provided domain
func (ds *DataStore) getDomain(domain string) (*Domain, error) {
	domain = strings.ToUpper(domain)
	rows, err := ds.db.Query("select ns.domain, dns.first_seen, dns.last_seen from nameservers, domains, domains_nameservers dns, nameservers ns where ns.id = dns.nameserver_id and domains.id = dns.domain_id and dns.nameserver_id = nameservers.id and dns.last_seen is null and domains.domain = $1 limit 10", domain)
	defer rows.Close()
	if err != nil {
		return nil, err
	}
	var data Domain
	data.Domain = domain
	data.NameServers = make([]NameServer, 0, 4)
	for rows.Next() {
		var result NameServer
		err = rows.Scan(&result.NameServer, &result.FirstSeen, &result.LastSeen)
		if err != nil {
			return nil, err
		}
		data.NameServers = append(data.NameServers, result)
	}
	return &data, nil
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
		row := ds.db.QueryRow("select domain from domains, domains_nameservers dns where dns.domain_id = id and domain_id = $1 and last_seen is null limit 1;", rid)
		err = row.Scan(&domain.Domain)
	}
	if err != nil {
		return nil, err
	}
	return &domain, nil
}

// gets information for the provided nameserver
func (ds *DataStore) getNameServer(domain string) (*NameServer, error) {
	domain = strings.ToUpper(domain)
	rows, err := ds.db.Query("select domains.domain, dns.first_seen, dns.last_seen from nameservers, domains, domains_nameservers dns, nameservers ns where ns.id = dns.nameserver_id and domains.id = dns.domain_id and dns.nameserver_id = nameservers.id and dns.last_seen is null and nameservers.domain = $1 order by first_seen desc nulls last limit 100", domain)
	defer rows.Close()
	if err != nil {
		return nil, err
	}
	var data NameServer
	data.NameServer = domain
	data.Domains = make([]Domain, 0, 4)
	for rows.Next() {
		var result Domain
		err = rows.Scan(&result.Domain, &result.FirstSeen, &result.LastSeen)
		if err != nil {
			return nil, err
		}
		data.Domains = append(data.Domains, result)
	}
	return &data, nil
}
