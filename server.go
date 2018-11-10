package main

/*
* Much of the design was models after this blog post
* http://nicolasmerouze.com/how-to-render-json-api-golang-mongodb/
 */

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
	"gopkg.in/throttled/throttled.v2"
	"gopkg.in/throttled/throttled.v2/store/memstore"
)

type serverContextType string

var serverContext serverContextType = "params"

// handler for catching a panic
// returns an HTTP code 500
func recoverHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic: %+v", err)
				WriteJSONError(w, ErrInternalServer)
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

// prints requests using the log package
func loggingHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		t1 := time.Now()
		next.ServeHTTP(w, r)
		t2 := time.Now()
		ip := getIPAddress(r)
		log.Printf("[%s] %s %q %v\n", ip, r.Method, r.RequestURI, t2.Sub(t1))
	}

	return http.HandlerFunc(fn)
}

// prints requests using the log package
/*func addContextHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		//TODO
		ctx := context.WithValue(r.Context(), serverContext, *spanID)
		r = r.WithContext(ctx)
	}

	return http.HandlerFunc(fn)
}*/

// 404 not found handler
/*func notFoundJSON(w http.ResponseWriter, r *http.Request) {
	WriteJSONError(w, ErrNotFound)
}*/

// creates a TimeoutHandler using the provided sec timeout
func makeTimeoutHandler(sec int) func(http.Handler) http.Handler {
	timeoutErrorJSON, err := json.Marshal(ErrTimeout)
	if err != nil {
		log.Fatal(err)
	}
	return func(h http.Handler) http.Handler {
		return http.TimeoutHandler(h, time.Duration(sec)*time.Second, string(timeoutErrorJSON))
	}
}

//custom vary by to use real remote IP without port
type myVaryBy struct{}

func (m myVaryBy) Key(r *http.Request) string {
	return getIPAddress(r)
}

// creates a throttled handler using the perMin limit on requests
func makeThrottleHandler(perMin, burst, storeSize int) func(http.Handler) http.Handler {
	store, err := memstore.New(storeSize)
	if err != nil {
		log.Fatal(err)
	}
	quota := throttled.RateQuota{
		MaxRate:  throttled.PerMin(perMin),
		MaxBurst: 5,
	}
	rateLimiter, err := throttled.NewGCRARateLimiter(store, quota)
	if err != nil {
		log.Fatal(err)
	}

	httpRateLimiter := throttled.HTTPRateLimiter{
		RateLimiter: rateLimiter,
		VaryBy:      new(myVaryBy),
		DeniedHandler: http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			WriteJSONError(w, ErrLimitExceeded)
		})),
	}

	return httpRateLimiter.RateLimit
}

// variables to hold common json errors
var (
	//ErrBadRequest           = &JSONError{"bad_request", 400, "Bad request", "Request body is not well-formed. It must be JSON."}
	//ErrUnauthorized         = &JSONError{"unauthorized", 401, "Unauthorized", "Access token is invalid."}
	ErrNotFound         = &JSONError{"not_found", 404, "Not found", "Route not found."}
	ErrResourceNotFound = &JSONError{"resource_not_found", 404, "Not found", "Resource not found."}
	ErrLimitExceeded    = &JSONError{"limit_exceeded", 429, "Too Many Requests", "To many requests, please wait and submit again."}
	ErrInternalServer   = &JSONError{"internal_server_error", 500, "Internal Server Error", "Something went wrong."}
	ErrNotImplemented   = &JSONError{"not_implemented", 501, "Not Implemented", "The server does not support the functionality required to fulfill the request. It may not have been implemented yet"}
	ErrTimeout          = &JSONError{"timeout", 503, "Service Unavailable", "The request took longer than expected to process."}
)

func HandlerNotImplemented(w http.ResponseWriter, r *http.Request) {
	WriteJSONError(w, ErrNotImplemented)
}

// TODO make not all errors JSON
func WriteJSONError(w http.ResponseWriter, err *JSONError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.Status)
	json.NewEncoder(w).Encode(JSONErrors{[]*JSONError{err}})
}

func WriteJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(JSONResponse{data})
	if err != nil && err != http.ErrHandlerTimeout {
		panic(err)
	}
}

// struct for holding server resources
type server struct {
	// basic router which is extended with many functions in server
	// should not be used by external functions
	// all communication with the server's router should be done with server methods
	router *httprouter.Router

	// Alice chain of http handlers
	handlers alice.Chain

	// Configuration
	config *Config
}

// creates a new server object with the default (included) handlers
func NewServer(config *Config) *server {
	server := &server{}
	server.router = httprouter.New()
	server.config = config
	server.handlers = alice.New(
		//context.ClearHandler,
		//addContextHandler,
		makeTimeoutHandler(server.config.API.Timeout),
		loggingHandler,
		recoverHandler,
		makeThrottleHandler(server.config.API.RequestsPerMinute, server.config.API.RequestsBurst, server.config.API.RequestsMaxHistory),
	)
	//server.router.NotFound = notFoundJSON
	return server
}

// add a method to the router's GET handler
func (s *server) Get(path string, fn http.HandlerFunc) {
	handler := s.handlers.ThenFunc(fn)
	s.router.Handler(http.MethodGet, path, handler)
	/*s.router.GET(path,
	func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		//context.Set(r, serverContext, ps)
		handler.ServeHTTP(w, r)
	})*/
}

// TODO gorilla mux is broken https://github.com/gorilla/context/issues/32
// move to http.Request.WithContext()

func (s *server) Post(path string, fn http.HandlerFunc) {
	handler := s.handlers.ThenFunc(fn)
	s.router.Handler(http.MethodPost, path, handler)
	/*s.router.POST(path,
	func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		// TODO VIAL this breaks!!!
		//context.Set(r, serverContext, ps)
		handler.ServeHTTP(w, r)
	})*/
}

// Starts the server
// blocking function
func (s *server) Start() error {
	return http.ListenAndServe(fmt.Sprintf("%s:%d", s.config.HTTP.IP, s.config.HTTP.Port), s.router)
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
