package model

var (
	IpNsZoneCountType = "ip_ns_zone_count"
)

type IpNsZoneCount struct {
	Type         *string     `json:"type"`
	Link         string      `json:"link"`
	IP           string      `json:"ip"`
	ZoneNSCounts []ZoneCount `json:"zone_counts"`
}

type ZoneCount struct {
	Zone    string  `json:"zone"`
	Count   int64   `json:"count"`
	Percent float64 `json:"percent"`
}

// GenerateMetaData generates metadata recursively of member models
func (ipzc *IpNsZoneCount) GenerateMetaData() {
	ipzc.Type = &IpNsZoneCountType
	ipzc.Link = "/research/ipnszonecount/" + ipzc.IP
}
