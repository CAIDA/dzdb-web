package main

import (
	"fmt"
	"time"
)

var (
	DomainType            = "domain"
	NameServerType        = "nameserver"
	IPType                = "ip"
	ImportProgressType    = "import_progress"
	ZoneImportResultType  = "zone_import_result"
	ZoneImportResultsType = "zone_import_results"
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

// Import Progress
type ImportProgress struct {
	Type    *string        `json:"type"`
	Link    string         `json:"link"`
	Imports int64          `json:"imports_left"`
	Days    int            `json:"days_left"`
	Dates   [15]ImportDate `json:"dates"` // gets last 15 days
}

type ImportDate struct {
	Date *time.Time `json:"date"`
	//TODO make duration
	//Took			time.Duration `json:"took"`
	Took  string `json:"took"`
	Count uint64 `json:"count"`
}

func (ip *ImportProgress) generateMetaData() {
	ip.Type = &ImportProgressType
	//ip.Link = "/imports/status"
	ip.Link = "/imports"
}

type ZoneImportResults struct {
	Type  *string             `json:"type"`
	Count int                 `json:"count"`
	Zones []*ZoneImportResult `json:"zones"`
}

func (zirs *ZoneImportResults) generateMetaData() {
	zirs.Type = &ZoneImportResultsType

	for _, v := range zirs.Zones {
		v.generateMetaData()
	}
}

type ZoneImportResult struct {
	Type    *string    `json:"type"`
	Id      int64      `json:"id"`
	Link    string     `json:"link"`
	Date    *time.Time `json:"date"`
	Zone    string     `json:"zone"`
	Records int64      `json:"records"`
	Domains int64      `json:"domains"`
	// TODO make duration
	Duration *string `json:"duration"`
	Old	     *int64  `json:"domains_old"`
	Moved    *int64  `json:"domains_moved"`
	New	     *int64  `json:"domains_new"`
	NewNs    *int64  `json:"ns_new"`
	OldNs    *int64  `json:"ns_old"`
	NewA     *int64  `json:"a_new"`
	OldA     *int64  `json:"a_old"`
	NewAaaa  *int64  `json:"aaaa_new"`
	OldAaaa  *int64  `json:"aaaa_old"`
}

func (zir *ZoneImportResult) generateMetaData() {
	zir.Type = &ZoneImportResultType
	zir.Link = fmt.Sprintf("/zones/%s", zir.Zone)
}

type Zone struct {
	Type 	*string 	`json:"type"`
	Link                   string        `json:"link"`
	Id 		int64		`json:"id"`
	Name 	string		`json:"name"`
	FirstSeen              *time.Time    `json:"firstseen,omitempty"`
	LastSeen               *time.Time    `json:"lastseen,omitempty"`
	NameServers            []*NameServer `json:"nameservers,omitempty"`
	ArchiveNameServers     []*NameServer `json:"archive_nameservers,omitempty"`
	NameServerCount        *int64        `json:"nameserver_count,omitempty"`
	ArchiveNameServerCount *int64        `json:"archive_nameserver_count,omitempty"`
}

func (z *Zone) generateMetaData() {
	z.Type = &DomainType
	z.Link = fmt.Sprintf("/zones/%s", z.Name)
}

// domain object
type Domain struct {
	Type                   *string       `json:"type"`
	Id                     int64         `json:"id"`
	Name                 string        `json:"name"`
	Link                   string        `json:"link"`
	FirstSeen              *time.Time    `json:"firstseen,omitempty"`
	LastSeen               *time.Time    `json:"lastseen,omitempty"`
	NameServers            []*NameServer `json:"nameservers,omitempty"`
	ArchiveNameServers     []*NameServer `json:"archive_nameservers,omitempty"`
	NameServerCount        *int64        `json:"nameserver_count,omitempty"`
	ArchiveNameServerCount *int64        `json:"archive_nameserver_count,omitempty"`
	Zone				   Zone
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
	d.Link = fmt.Sprintf("/domains/%s", d.Name)
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
	Name        	   string     `json:"name"`
	Link               string     `json:"link"`
	FirstSeen          *time.Time `json:"firstseen,omitempty"`
	LastSeen           *time.Time `json:"lastseen,omitempty"`
	Domains            []*Domain  `json:"domains,omitempty"`
	ArchiveDomains     []*Domain  `json:"archive_domains,omitempty"`
	DomainCount        *int64     `json:"domain_count,omitempty"`
	ArchiveDomainCount *int64     `json:"archive_domain_count,omitempty"`
	IP4 			   []*IP4      `json:"ipv4,omitempty"`
	ArchiveIP4 			   []*IP4      `json:"archive_ipv4,omitempty"`
	IP4Count *int64     `json:"ipv4_count,omitempty"`
	ArchiveIP4Count *int64     `json:"archive_ipv4_count,omitempty"`
	IP6 			   []*IP6      `json:"ipv6,omitempty"`
	ArchiveIP6 			   []*IP6      `json:"archive_ipv6,omitempty"`
	IP6Count *int64     `json:"ipv6_count,omitempty"`
	ArchiveIP6Count *int64     `json:"archive_ipv6_count,omitempty"`
}

func (ns *NameServer) generateMetaData() {
	ns.Type = &NameServerType
	ns.Link = fmt.Sprintf("/nameservers/%s", ns.Name)
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
	for _, ip := range ns.IP4 {
		if ip.Type == nil {
			ip.generateMetaData()
		}
	}
	for _, ip := range ns.ArchiveIP4 {
		if ip.Type == nil {
			ip.generateMetaData()
		}
	}
	for _, ip := range ns.IP6 {
		if ip.Type == nil {
			ip.generateMetaData()
		}
	}
	for _, ip := range ns.ArchiveIP6 {
		if ip.Type == nil {
			ip.generateMetaData()
		}
	}
}

type IP struct {
	Type               *string    `json:"type"`
	Id                 int64      `json:"id"`
	Name        	   string     `json:"name"`
	Version			int `json:"version"`
	Link               string     `json:"link"`
	FirstSeen          *time.Time `json:"firstseen,omitempty"`
	LastSeen           *time.Time `json:"lastseen,omitempty"`
	NameServers            []*NameServer `json:"nameservers,omitempty"`
	ArchiveNameServers     []*NameServer `json:"archive_nameservers,omitempty"`
	NameServerCount        *int64        `json:"nameserver_count,omitempty"`
	ArchiveNameServerCount *int64        `json:"archive_nameserver_count,omitempty"`
}


type IP4 struct {
	IP
}

type IP6 struct {
	IP
}

func (ip *IP) generateMetaData() {
	ip.Type = &IPType
	ip.Link = fmt.Sprintf("/ip/%s", ip.Name)
	for _, ns := range ip.NameServers {
		if ns.Type == nil {
			ns.generateMetaData()
		}
	}
	for _, ns := range ip.ArchiveNameServers {
		if ns.Type == nil {
			ns.generateMetaData()
		}
	}
}
