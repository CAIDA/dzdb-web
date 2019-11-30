package datastore

import (
	"context"
	"strings"
	"vdz-web/model"
)

// GetIpNsZoneCount returns the count of nameservers pointing to an IP grouped
// by the zone
func (ds *DataStore) GetIPNsZoneCount(ctx context.Context, ip string) (*model.IpNsZoneCount, error) {
	var ipzc model.IpNsZoneCount
	var err error
	ipzc.IP = ip

	query := "select zone, count(*) from zones, a_nameservers, a where a.id = a_nameservers.a_id and zones.id = a_nameservers.zone_id and a_nameservers.last_seen is null and a.ip = $1 group by zone"
	if !strings.Contains(ip, ".") {
		query = "select zone, count(*) from zones, aaaa_nameservers, aaaa where aaaa.id = aaaa_nameservers.aaaa_id and zones.id = aaaa_nameservers.zone_id and aaaa_nameservers.last_seen is null and aaaa.ip = $1 group by zone"
	}

	rows, err := ds.db.QueryContext(ctx, query, ip)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	ipzc.ZoneNSCounts = make(map[string]int64)

	for rows.Next() {
		var zone string
		var count int64
		err = rows.Scan(&zone, &count)
		if err != nil {
			return nil, err
		}
		ipzc.ZoneNSCounts[zone] = count
	}

	return &ipzc, nil
}
