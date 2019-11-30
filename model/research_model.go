package model

var (
	IpNsZoneCountType = "ip_ns_zone_count"
)

type IpNsZoneCount struct {
	Type         *string          `json:"type"`
	Link         string           `json:"link"`
	IP           string           `json:"ip"`
	ZoneNSCounts map[string]int64 `json:"zone_ns_counts"`
}

// GenerateMetaData generates metadata recursively of member models
func (ipzc *IpNsZoneCount) GenerateMetaData() {
	ipzc.Type = &IpNsZoneCountType
	ipzc.Link = "/research/ipnszonecount/" + ipzc.IP
}
