package main

import (
	"time"
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
