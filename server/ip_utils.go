package server

import (
	"net/http"
)

//custom vary by to use real remote IP without port
type ipVaryBy struct{}

func (ip ipVaryBy) Key(r *http.Request) string {
	return r.RemoteAddr
}
