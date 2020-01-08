package datastore

import (
	"context"
	"strings"
	"vdz-web/model"
)

// GetIPNsZoneCount returns the count of nameservers pointing to an IP grouped
// by the zone
func (ds *DataStore) GetIPNsZoneCount(ctx context.Context, ip string) (*model.ResearchIpNsZoneCount, error) {
	var ipzc model.ResearchIpNsZoneCount
	var err error
	ipzc.IP = ip

	query := "select zone, count(*) from zones, a_nameservers, a where a.id = a_nameservers.a_id and zones.id = a_nameservers.zone_id and a_nameservers.last_seen is null and a.ip = $1 group by zone order by count desc"
	if strings.Contains(ip, ":") {
		query = "select zone, count(*) from zones, aaaa_nameservers, aaaa where aaaa.id = aaaa_nameservers.aaaa_id and zones.id = aaaa_nameservers.zone_id and aaaa_nameservers.last_seen is null and aaaa.ip = $1 group by zone order by count desc"
	}

	rows, err := ds.db.QueryContext(ctx, query, ip)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	total := float64(0)

	ipzc.ZoneNSCounts = make([]model.ResearchZoneCount, 0)
	for rows.Next() {
		var zc model.ResearchZoneCount
		err = rows.Scan(&zc.Zone, &zc.Count)
		if err != nil {
			return nil, err
		}
		total = total + float64(zc.Count)
		ipzc.ZoneNSCounts = append(ipzc.ZoneNSCounts, zc)
	}
	// set percents
	for i := range ipzc.ZoneNSCounts {
		ipzc.ZoneNSCounts[i].Percent = 100 * float64(ipzc.ZoneNSCounts[i].Count) / total
	}

	return &ipzc, nil
}
