package datastore

import (
	"context"
	"dnscoffee/model"
	"strings"
	"time"
)

// GetActiveIPs returns the active IP addresses (IPv4 and IPv6) for a given date
func (ds *DataStore) GetActiveIPs(ctx context.Context, date time.Time) (*model.ActiveIPs, error) {
	var err error
	var aip model.ActiveIPs
	aip.Date = date

	query := "select distinct a.ip from a_nameservers, a where a_nameservers.a_id = a.id and first_seen <= $1 and (last_seen >= $1 or last_seen is NULL)"
	rows, err := ds.db.QueryContext(ctx, query, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	aip.IPv4IPs = make([]string, 0)
	for rows.Next() {
		var ipv4 string
		err = rows.Scan(&ipv4)
		if err != nil {
			return nil, err
		}
		aip.IPv4IPs = append(aip.IPv4IPs, ipv4)
	}

	query = "select distinct aaaa.ip from aaaa_nameservers, aaaa where aaaa_nameservers.aaaa_id = aaaa.id and first_seen <= $1 and (last_seen >= $1 or last_seen is NULL)"
	rows, err = ds.db.QueryContext(ctx, query, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	aip.IPv6IPs = make([]string, 0)
	for rows.Next() {
		var ipv6 string
		err = rows.Scan(&ipv6)
		if err != nil {
			return nil, err
		}
		aip.IPv6IPs = append(aip.IPv6IPs, ipv6)
	}

	return &aip, nil
}

// GetIPNsZoneCount returns the count of nameservers pointing to an IP grouped
// by the zone
func (ds *DataStore) GetIPNsZoneCount(ctx context.Context, ip string) (*model.ResearchIpNsZoneCount, error) {
	var ipzc model.ResearchIpNsZoneCount
	var err error
	ipzc.IP = ip

	query := "select zone, count(*) from zones, a_nameservers, a where a.id = a_nameservers.a_id and zones.id = a_nameservers.zone_id and a.ip = $1 group by zone order by count desc"
	if strings.Contains(ip, ":") {
		query = "select zone, count(*) from zones, aaaa_nameservers, aaaa where aaaa.id = aaaa_nameservers.aaaa_id and zones.id = aaaa_nameservers.zone_id and aaaa.ip = $1 group by zone order by count desc"
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
