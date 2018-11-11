package server

import (
	"net"
	"net/http"
	"strings"
)

//custom vary by to use real remote IP without port
type ipVaryBy struct{}

func (ip ipVaryBy) Key(r *http.Request) string {
	return getIPAddress(r)
}

func getIPAddress(r *http.Request) string {
	hdr := r.Header
	hdrRealIP := hdr.Get("X-Real-Ip")
	hdrForwardedFor := hdr.Get("X-Forwarded-For")
	if hdrRealIP == "" && hdrForwardedFor == "" {
		hdrRealIP, _, _ := net.SplitHostPort(r.RemoteAddr)
		return hdrRealIP
	}
	if hdrForwardedFor != "" {
		// X-Forwarded-For is potentially a list of addresses separated with ","
		parts := strings.Split(hdrForwardedFor, ",")
		for i, p := range parts {
			parts[i] = strings.TrimSpace(p)
		}
		// TODO: should return first non-local address
		return parts[0]
	}
	return hdrRealIP
}
