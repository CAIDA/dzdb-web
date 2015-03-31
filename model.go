package main

import (
	"time"
)

// domain object
type Domain struct {
	Domain      string       `json:"domain"`
	NameServers []NameServer `json:"nameservers,omitempty"`
	FirstSeen   *time.Time   `json:"firstseen,omitempty"`
	LastSeen    *time.Time   `json:"lastseen,omitempty"`
}

// nameserver object
type NameServer struct {
	NameServer string     `json:"nameserver"`
	Domains    []Domain   `json:"domains,omitempty"`
	FirstSeen  *time.Time `json:"firstseen,omitempty"`
	LastSeen   *time.Time `json:"lastseen,omitempty"`
}
