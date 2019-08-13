package model

import (
	"fmt"
	"time"
)

var (
	domainType            = "domain"
	feedType              = "feed"
	feedNsType            = "feed_ns"
	nameServerType        = "nameserver"
	ipType                = "ip"
	importProgressType    = "import_progress"
	zoneImportResultType  = "zone_import_result"
	zoneImportResultsType = "zone_import_results"
)

// JSONResponse JSON-API root data object
type JSONResponse struct {
	Data interface{} `json:"data,omitempty"`
}

// JSONErrors JSON-API root error object
type JSONErrors struct {
	Errors []*JSONError `json:"errors"`
}

// JSONError JSON-API error object
type JSONError struct {
	ID     string `json:"id"`
	Status int    `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

// NewJSONError returns a New JSONError
func NewJSONError(id string, status int, title, detail string) *JSONError {
	jsonErr := &JSONError{
		ID:     id,
		Status: status,
		Title:  title,
		Detail: detail,
	}
	return jsonErr
}

// Err implements the error interface.
func (err JSONError) Error() string {
	return err.Detail
}

// ImportProgress Import Progress
type ImportProgress struct {
	Type    *string        `json:"type"`
	Link    string         `json:"link"`
	Imports int64          `json:"imports_left"`
	Days    int            `json:"days_left"`
	Dates   [15]ImportDate `json:"dates"` // gets last 15 days
}

// ImportDate import date data
type ImportDate struct {
	Date *time.Time `json:"date"`
	//TODO make duration
	//Took			time.Duration `json:"took"`
	Took  string `json:"took"`
	Count uint64 `json:"count"`
}

// GenerateMetaData generates metadata recursively of member models
func (ip *ImportProgress) GenerateMetaData() {
	ip.Type = &importProgressType
	//ip.Link = "/imports/status"
	ip.Link = "/imports"
}

// ZoneImportResults results for imports
type ZoneImportResults struct {
	Type  *string             `json:"type"`
	Count int                 `json:"count"`
	Zones []*ZoneImportResult `json:"zones"`
}

// GenerateMetaData generates metadata recursively of member models
func (zirs *ZoneImportResults) GenerateMetaData() {
	zirs.Type = &zoneImportResultsType

	for _, v := range zirs.Zones {
		v.GenerateMetaData()
	}
}

// ZoneImportResult holds data about the results of a single import
type ZoneImportResult struct {
	Type    *string    `json:"type"`
	ID      int64      `json:"id"`
	Link    string     `json:"link"`
	Date    *time.Time `json:"date"`
	Zone    string     `json:"zone"`
	Records int64      `json:"records"`
	Domains int64      `json:"domains"`
	// TODO make duration
	Duration *string `json:"duration"`
	Old      *int64  `json:"domains_old"`
	Moved    *int64  `json:"domains_moved"`
	New      *int64  `json:"domains_new"`
	NewNs    *int64  `json:"ns_new"`
	OldNs    *int64  `json:"ns_old"`
	NewA     *int64  `json:"a_new"`
	OldA     *int64  `json:"a_old"`
	NewAaaa  *int64  `json:"aaaa_new"`
	OldAaaa  *int64  `json:"aaaa_old"`
}

// GenerateMetaData generates metadata recursively of member models
func (zir *ZoneImportResult) GenerateMetaData() {
	zir.Type = &zoneImportResultType
	zir.Link = fmt.Sprintf("/zones/%s", zir.Zone)
}

// Zone holds information about a zone
// TODO change time.Time to nulltime?
type Zone struct {
	Type                   *string       `json:"type"`
	Link                   string        `json:"link"`
	ID                     int64         `json:"id"`
	Name                   string        `json:"name"`
	FirstSeen              *time.Time    `json:"firstseen,omitempty"`
	LastSeen               *time.Time    `json:"lastseen,omitempty"`
	NameServers            []*NameServer `json:"nameservers,omitempty"`
	ArchiveNameServers     []*NameServer `json:"archive_nameservers,omitempty"`
	NameServerCount        *int64        `json:"nameserver_count,omitempty"`
	ArchiveNameServerCount *int64        `json:"archive_nameserver_count,omitempty"`
}

// GenerateMetaData generates metadata recursively of member models
func (z *Zone) GenerateMetaData() {
	z.Type = &domainType
	z.Link = fmt.Sprintf("/zones/%s", z.Name)
}

// Domain domain object
type Domain struct {
	Type                   *string       `json:"type"`
	ID                     int64         `json:"id"`
	Name                   string        `json:"name"`
	Link                   string        `json:"link"`
	FirstSeen              *time.Time    `json:"firstseen,omitempty"`
	LastSeen               *time.Time    `json:"lastseen,omitempty"`
	NameServers            []*NameServer `json:"nameservers,omitempty"`
	ArchiveNameServers     []*NameServer `json:"archive_nameservers,omitempty"`
	NameServerCount        *int64        `json:"nameserver_count,omitempty"`
	ArchiveNameServerCount *int64        `json:"archive_nameserver_count,omitempty"`
	Zone                   *Zone         `json:"zone,omitempty"`
}

// GenerateMetaData generates metadata recursively of member models
func (d *Domain) GenerateMetaData() {
	d.Type = &domainType
	d.Link = fmt.Sprintf("/domains/%s", d.Name)
	for _, ns := range d.NameServers {
		if ns.Type == nil {
			ns.GenerateMetaData()
		}
	}
	for _, ns := range d.ArchiveNameServers {
		if ns.Type == nil {
			ns.GenerateMetaData()
		}
	}
}

type Feed struct {
	Type    *string   `json:"type"`
	Link    string    `json:"link"`
	Change  string    `json:"change,omitempty"`
	Date    time.Time `json:"date"`
	Domains []*Domain `json:"domains"`
}

func (f *Feed) GenerateMetaData() {
	f.Type = &feedType
	y, m, d := f.Date.Date()
	f.Link = fmt.Sprintf("/feeds/%s/%04d-%02d-%02d", f.Change, y, m, d)
	for _, d := range f.Domains {
		if d.Type == nil {
			d.GenerateMetaData()
		}
	}
}

type NSFeed struct {
	Type         *string       `json:"type"`
	Link         string        `json:"link"`
	Change       string        `json:"change,omitempty"`
	Date         time.Time     `json:"date"`
	Nameservers4 []*NameServer `json:"nameservers_4"`
	Nameservers6 []*NameServer `json:"nameservers_6"`
}

func (f *NSFeed) GenerateMetaData() {
	f.Type = &feedNsType
	y, m, d := f.Date.Date()
	f.Link = fmt.Sprintf("/feeds/ns/%s/%04d-%02d-%02d", f.Change, y, m, d)
	for _, n := range f.Nameservers4 {
		if n.Type == nil {
			n.GenerateMetaData()
		}
	}
	for _, n := range f.Nameservers6 {
		if n.Type == nil {
			n.GenerateMetaData()
		}
	}
}

// NameServer nameserver object
type NameServer struct {
	Type               *string    `json:"type"`
	ID                 int64      `json:"id"`
	Name               string     `json:"name"`
	Link               string     `json:"link"`
	FirstSeen          *time.Time `json:"firstseen,omitempty"`
	LastSeen           *time.Time `json:"lastseen,omitempty"`
	Domains            []*Domain  `json:"domains,omitempty"`
	ArchiveDomains     []*Domain  `json:"archive_domains,omitempty"`
	DomainCount        *int64     `json:"domain_count,omitempty"`
	ArchiveDomainCount *int64     `json:"archive_domain_count,omitempty"`
	IP4                []*IP4     `json:"ipv4,omitempty"`
	ArchiveIP4         []*IP4     `json:"archive_ipv4,omitempty"`
	IP4Count           *int64     `json:"ipv4_count,omitempty"`
	ArchiveIP4Count    *int64     `json:"archive_ipv4_count,omitempty"`
	IP6                []*IP6     `json:"ipv6,omitempty"`
	ArchiveIP6         []*IP6     `json:"archive_ipv6,omitempty"`
	IP6Count           *int64     `json:"ipv6_count,omitempty"`
	ArchiveIP6Count    *int64     `json:"archive_ipv6_count,omitempty"`
}

// GenerateMetaData generates metadata recursively of member models
func (ns *NameServer) GenerateMetaData() {
	ns.Type = &nameServerType
	ns.Link = fmt.Sprintf("/nameservers/%s", ns.Name)
	for _, d := range ns.Domains {
		if d.Type == nil {
			d.GenerateMetaData()
		}
	}
	for _, d := range ns.ArchiveDomains {
		if d.Type == nil {
			d.GenerateMetaData()
		}
	}
	for _, ip := range ns.IP4 {
		if ip.Type == nil {
			ip.GenerateMetaData()
		}
	}
	for _, ip := range ns.ArchiveIP4 {
		if ip.Type == nil {
			ip.GenerateMetaData()
		}
	}
	for _, ip := range ns.IP6 {
		if ip.Type == nil {
			ip.GenerateMetaData()
		}
	}
	for _, ip := range ns.ArchiveIP6 {
		if ip.Type == nil {
			ip.GenerateMetaData()
		}
	}
}

// IP holds information about an IP address
type IP struct {
	Type                   *string       `json:"type"`
	ID                     int64         `json:"id"`
	Name                   string        `json:"name"`
	Version                int           `json:"version"`
	Link                   string        `json:"link"`
	FirstSeen              *time.Time    `json:"firstseen,omitempty"`
	LastSeen               *time.Time    `json:"lastseen,omitempty"`
	NameServers            []*NameServer `json:"nameservers,omitempty"`
	ArchiveNameServers     []*NameServer `json:"archive_nameservers,omitempty"`
	NameServerCount        *int64        `json:"nameserver_count,omitempty"`
	ArchiveNameServerCount *int64        `json:"archive_nameserver_count,omitempty"`
}

// IP4 is an alias to the IP type
type IP4 struct {
	IP
}

// IP6 is an alias to the IP type
type IP6 struct {
	IP
}

// GenerateMetaData generates metadata recursively of member models
func (ip *IP) GenerateMetaData() {
	ip.Type = &ipType
	ip.Link = fmt.Sprintf("/ip/%s", ip.Name)
	for _, ns := range ip.NameServers {
		if ns.Type == nil {
			ns.GenerateMetaData()
		}
	}
	for _, ns := range ip.ArchiveNameServers {
		if ns.Type == nil {
			ns.GenerateMetaData()
		}
	}
}
