package main

import (
	"fmt"
	"time"
)

var (
	DomainType     = "domain"
	NameServerType = "nameserver"
)

// JSON-API root data object
type JSONResponse struct {
	Data interface{} `json:"data,omitempty"`
}

// JSON-API root error object
type JSONErrors struct {
	Errors []*JSONError `json:"errors"`
}

// JSON-API error object
type JSONError struct {
	Id     string `json:"id"`
	Status int    `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

// Err implements the error interface.
func (err JSONError) Error() string {
	return err.Detail
}

// domain object
type Domain struct {
	Type                   *string       `json:"type"`
	Id                     int64         `json:"id"`
	Domain                 string        `json:"domain"`
	Link                   string        `json:"link"`
	FirstSeen              *time.Time    `json:"firstseen,omitempty"`
	LastSeen               *time.Time    `json:"lastseen,omitempty"`
	NameServers            []*NameServer `json:"nameservers,omitempty"`
	ArchiveNameServers     []*NameServer `json:"archive_nameservers,omitempty"`
	NameServerCount        *int64        `json:"nameserver_count,omitempty"`
	ArchiveNameServerCount *int64        `json:"archive_nameserver_count,omitempty"`
}

/*func NewDomain(id int64, domain, string) *Domain {
	d :=Domain{}
	d.Id = id
	d.Domain = domain
	d.NameServers = make([]*NameServer, 0, 4)
	d.ArchiveNameServers = make([]*NameServer, 0, 4)
	return &d
}*/

func (d *Domain) generateMetaData() {
	d.Type = &DomainType
	d.Link = fmt.Sprintf("/domains/%s", d.Domain)
	for _, ns := range d.NameServers {
		if ns.Type == nil {
			ns.generateMetaData()
		}
	}
	for _, ns := range d.ArchiveNameServers {
		if ns.Type == nil {
			ns.generateMetaData()
		}
	}
}

// nameserver object
type NameServer struct {
	Type               *string    `json:"type"`
	Id                 int64      `json:"id"`
	NameServer         string     `json:"nameserver"`
	Link               string     `json:"link"`
	FirstSeen          *time.Time `json:"firstseen,omitempty"`
	LastSeen           *time.Time `json:"lastseen,omitempty"`
	Domains            []*Domain  `json:"domains,omitempty"`
	ArchiveDomains     []*Domain  `json:"archive_domains,omitempty"`
	DomainCount        *int64     `json:"domain_count,omitempty"`
	ArchiveDomainCount *int64     `json:"archive_domain_count,omitempty"`
}

/*func NewNameServer(id int64, domain string) *NameServer {
	ns := NameServer{}
	ns.Id = id
	ns.NameServer = domain
	d.Domains = make([]*Domains, 0, 4)
	d.ArchiveDomains = make([]*Domains, 0, 4)
	return &ns
}*/

func (ns *NameServer) generateMetaData() {
	ns.Type = &NameServerType
	ns.Link = fmt.Sprintf("/nameservers/%s", ns.NameServer)
	for _, d := range ns.Domains {
		if d.Type == nil {
			d.generateMetaData()
		}
	}
	for _, d := range ns.ArchiveDomains {
		if d.Type == nil {
			d.generateMetaData()
		}
	}
}
