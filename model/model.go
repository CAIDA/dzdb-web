// Package model defined the types used by the API
package model

import (
	"fmt"
	"net"
	"time"
)

var (
	domainType            = "domain"
	zoneType              = "zone"
	feedType              = "feed"
	feedNsType            = "feed_ns"
	nameServerType        = "nameserver"
	ipType                = "ip"
	ipListType            = "ips"
	importProgressType    = "import_progress"
	zoneImportResultType  = "zone_import_result"
	zoneImportResultsType = "zone_import_results"
	zoneCountsType        = "zone_counts"
	zoneAllCountsType     = "zone_all_counts"
)

// APIData interface forces the use of GenerateMetaData on response data
type APIData interface {
	GenerateMetaData()
}

// Metadata defines the object's type and Link to self for API responses
type Metadata struct {
	Type *string `json:"type,omitempty"`
	Link string  `json:"link,omitempty"`
}

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
	ID     string `json:"-"`
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

// Error implements the error interface.
func (err JSONError) Error() string {
	return err.Detail
}

// Dataset holds information about entire dataset
type Dataset struct {
	TopNameServers []*NameServer `json:"topnameservers,omitempty"`
}

// ImportDate import date data
// TODO go2: time.Duration does not marshal into JSON correctly https://github.com/golang/go/issues/10275
type ImportDate struct {
	Date           *time.Time    `json:"date"`
	DiffDuration   time.Duration `json:"diff_duration"`
	ImportDuration time.Duration `json:"import_duration"`
	Count          uint64        `json:"count"`
}

// ZoneImportResults results for imports
type ZoneImportResults struct {
	Metadata
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
	Metadata
	FirstImportID   int64      `json:"-"`
	LastImportID    int64      `json:"-"`
	FirstImportDate *time.Time `json:"first_date"`
	LastImportDate  *time.Time `json:"last_date"`
	Zone            string     `json:"zone"`
	Records         int64      `json:"records"`
	Domains         int64      `json:"domains"`
	Count           int64      `json:"count"`
}

// GenerateMetaData generates metadata recursively of member models
func (zir *ZoneImportResult) GenerateMetaData() {
	zir.Type = &zoneImportResultType
	zir.Link = fmt.Sprintf("/zones/%s", zir.Zone)
}

// ZoneCount struct that contains a history of a single zone's sizes over time
type ZoneCount struct {
	Metadata
	Zone    string        `json:"zone"`
	History []*ZoneCounts `json:"history"`
}

// GenerateMetaData generates metadata recursively of member models
func (zc *ZoneCount) GenerateMetaData() {
	zc.Type = &zoneCountsType
	zc.Link = fmt.Sprintf("/counts/zones/%s", zc.Zone)
}

// ZoneCounts contains stats and counts for a single zone on a single day
type ZoneCounts struct {
	Date    time.Time `json:"date"`
	Domains int64     `json:"domains"`
	Old     int64     `json:"old"`
	Moved   int64     `json:"moved"`
	New     int64     `json:"new"`
}

// AllZoneCounts contains zone counts for all zones in a Map
type AllZoneCounts struct {
	Metadata
	Counts map[string]*ZoneCount `json:"counts"`
}

// GenerateMetaData generates metadata recursively of member models
func (zc *AllZoneCounts) GenerateMetaData() {
	zc.Type = &zoneAllCountsType
	zc.Link = "/counts/all"
}

// Zone holds information about a zone
type Zone struct {
	Metadata
	ID                     int64             `json:"-"`
	Name                   string            `json:"name"`
	FirstSeen              *time.Time        `json:"firstseen,omitempty"`
	LastSeen               *time.Time        `json:"lastseen,omitempty"`
	NameServers            []*NameServer     `json:"nameservers,omitempty"`
	ArchiveNameServers     []*NameServer     `json:"archive_nameservers,omitempty"`
	NameServerCount        *int64            `json:"nameserver_count,omitempty"`
	ArchiveNameServerCount *int64            `json:"archive_nameserver_count,omitempty"`
	ImportData             *ZoneImportResult `json:"import_data,omitempty"`
	Domains                *[]Domain         `json:"domains,omitempty"`
	RootImport             *RootZone         `json:"root,omitempty"`
}

// RootZone adds root metadata to the zone types
type RootZone struct {
	FirstImport *time.Time `json:"first_import"`
	LastImport  *time.Time `json:"last_import"`
}

// GenerateMetaData generates metadata recursively of member models
func (z *Zone) GenerateMetaData() {
	z.Type = &zoneType
	z.Link = fmt.Sprintf("/zones/%s", z.Name)
}

// Domain domain object
type Domain struct {
	Metadata
	ID                     int64         `json:"-"`
	Name                   string        `json:"name"`
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
	Metadata
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
	Metadata
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
	Metadata
	ID                 int64      `json:"-"`
	Name               string     `json:"name"`
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
	Zone               *Zone      `json:"zone,omitempty"`
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
	Metadata
	ID                     int64         `json:"-"`
	Name                   string        `json:"name"`
	IP                     *net.IP       `json:"-"`
	Version                int           `json:"version"`
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

// IPString returns the IP as a string
// if it is an AAAA records with an IPv4 address returns the value in IPv6 notation
// must be called after version and IP are set
func (ip *IP) IPString() string {
	ipStr := ip.IP.String()
	if ip.Version == 6 && ip.IP.To4() != nil {
		ipStr = fmt.Sprintf("::FFFF:%s", ipStr)
	}
	return ipStr
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

// IPList is a list of IPs
type IPList struct {
	Metadata
	IPs            []*IP `json:"ips,omitempty"`
}

// GenerateMetaData generates metadata recursively of member models
func (ipl *IPList) GenerateMetaData() {
	ipl.Type = &ipListType
	ipl.Link = fmt.Sprintf("/ip")
	for _, ip := range ipl.IPs {
		if ip.Type == nil {
			ip.GenerateMetaData()
		}
	}
}

// Search has the metadata and results for a search operation
type Search struct {
	Query   string
	Type    string
	Results []SearchResult
}

// SearchResult has the name and type of search results
type SearchResult struct {
	Name string
	Link string
	Type string
}

// PrefixResult stores the result of an individual prefix search result
type PrefixResult struct {
	Domain    string     `json:"domain"`
	FirstSeen *time.Time `json:"firstseen,omitempty"`
	LastSeen  *time.Time `json:"lastseen,omitempty"`
}

// PrefixList holds information about an IP address
type PrefixList struct {
	Metadata
	Active  bool           `json:"active"`
	Prefix  string         `json:"prefix"`
	Domains []PrefixResult `json:"domains"`
}

// TLDLife holds TLD age information for the TLD graveyard page
type TLDLife struct {
	Metadata
	Zone    string     `json:"zone"`
	Created *time.Time `json:"created"`
	Removed *time.Time `json:"removed"`
	Domains *int64     `json:"domains"`
	Age     *string    `json:"age"`
}

type FeedCountList struct {
	Search string      `json:"search"`
	Type   string      `json:"type"`
	Counts []FeedCount `json:"counts"`
}

// GenerateMetaData generates metadata recursively of member models
func (fc *FeedCountList) GenerateMetaData() {
	// TODO ?
}

type FeedCount struct {
	Date  *time.Time `json:"date"`
	Count int64      `json:"count"`
}
