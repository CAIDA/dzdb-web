package model

var (
	IpNsZoneCountType = "ip_ns_zone_count"
)

type ResearchIpNsZoneCount struct {
	Type         *string             `json:"type"`
	Link         string              `json:"link"`
	IP           string              `json:"ip"`
	ZoneNSCounts []ResearchZoneCount `json:"zone_counts"`
}

type ResearchZoneCount struct {
	Zone    string  `json:"zone"`
	Count   int64   `json:"count"`
	Percent float64 `json:"percent"`
}

// GenerateMetaData generates metadata recursively of member models
func (ipzc *ResearchIpNsZoneCount) GenerateMetaData() {
	ipzc.Type = &IpNsZoneCountType
	ipzc.Link = "/research/ipnszonecount/" + ipzc.IP
}
