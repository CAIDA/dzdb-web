// Package server handles the http server for the frontend
package server

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"dnscoffee/model"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"gopkg.in/throttled/throttled.v2"
	"gopkg.in/throttled/throttled.v2/store/memstore"
)

type APIConfig struct {
	APITimeout           int
	APIRequestsPerMinute int
	APIMaxRequestHistory int
	APIRequestsBurst     int
}

var DefaultAPIConfig = APIConfig{
	APITimeout:           60,
	APIRequestsPerMinute: 5 * 60,
	APIMaxRequestHistory: 16384,
	APIRequestsBurst:     10,
}

// 404 not found handler
//func notFoundJSON(w http.ResponseWriter, r *http.Request) {
//	WriteJSONError(w, ErrNotFound)
//}

// creates a throttled handler using the perMin limit on requests
func makeThrottleHandler(perMin, burst, storeSize int) func(http.Handler) http.Handler {
	store, err := memstore.New(storeSize)
	if err != nil {
		log.Fatal(err)
	}
	quota := throttled.RateQuota{
		MaxRate:  throttled.PerMin(perMin),
		MaxBurst: burst,
	}
	rateLimiter, err := throttled.NewGCRARateLimiter(store, quota)
	if err != nil {
		log.Fatal(err)
	}

	httpRateLimiter := throttled.HTTPRateLimiter{
		RateLimiter: rateLimiter,
		VaryBy:      new(ipVaryBy),
		DeniedHandler: http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			WriteJSONError(w, ErrLimitExceeded)
		})),
	}

	return httpRateLimiter.RateLimit
}

// // HandlerNotImplemented returns ErrNotImplemented as JSON
// func HandlerNotImplemented(w http.ResponseWriter, r *http.Request) {
// 	WriteJSONError(w, ErrNotImplemented)
// }

// WriteJSONError returns an error as JSON
// TODO make not all errors JSON
func WriteJSONError(w http.ResponseWriter, jsonErr *model.JSONError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(jsonErr.Status)
	err := json.NewEncoder(w).Encode(model.JSONErrors{Errors: []*model.JSONError{jsonErr}})
	if err != nil {
		panic(err)
	}
}

// WriteJSON writes JSON from data to the response
func WriteJSON(w http.ResponseWriter, data model.APIData) {
	data.GenerateMetaData()
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(model.JSONResponse{Data: data})
	if err != nil && err != http.ErrHandlerTimeout {
		panic(err)
	}
}

// Server struct for holding server resources
type Server struct {
	// basic router which is extended with many functions in server
	// should not be used by external functions
	// all communication with the server's router should be done with server methods
	router *mux.Router

	listenAddr string

	apiConfig APIConfig
}

// New creates a new server object with the default (included) handlers
func New(listenAddr string, apiConfig APIConfig) (*Server, error) {
	server := &Server{
		listenAddr: listenAddr,
		apiConfig:  apiConfig,
		router:     mux.NewRouter(), //.StrictSlash(true),
	}

	// serve static content
	static := http.StripPrefix("/static/", http.FileServer(http.Dir("static")))
	server.router.PathPrefix("/static/").Methods(http.MethodGet).Handler(neuterDirectoryListing(static))

	// setup robots.txt
	server.router.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/robots.txt")
	}).Methods(http.MethodGet)
	// favicon
	server.router.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/favicon.ico")
	}).Methods(http.MethodGet)

	return server, nil

	// // add rate limiting after static handler
	// server.handlers = handlers.Append(makeThrottleHandler(
	// 	server.apiConfig.APIRequestsPerMinute,
	// 	server.apiConfig.APIRequestsBurst,
	// 	server.apiConfig.APIMaxRequestHistory,
	// ))

}

// Get registers a HTTP GET to the router & handler
func (s *Server) Get(path string, fn http.HandlerFunc) {
	s.router.Handle(path, fn).Methods(http.MethodGet)
}

// Post registers a HTTP POST to the router & handler
func (s *Server) Post(path string, fn http.HandlerFunc) {
	s.router.Handle(path, fn).Methods(http.MethodPost)
}

// Start Starts the server, blocking function
func (s *Server) Start() error {
	timeoutDuration := time.Duration(s.apiConfig.APITimeout) * time.Second
	// prep proxy handler
	h := handlers.ProxyHeaders(s.router)
	// setup logging
	h = handlers.LoggingHandler(os.Stdout, h)
	// add recovery
	h = handlers.RecoveryHandler(handlers.PrintRecoveryStack(true))(h)
	// timeouts
	h = http.TimeoutHandler(h, timeoutDuration, ErrTimeout.Error())
	// cors
	h = handlers.CORS(handlers.AllowedOrigins([]string{"http://127.0.0.1:5353"}))(h)
	// run server
	srv := &http.Server{
		Handler:      h,
		Addr:         s.listenAddr,
		WriteTimeout: timeoutDuration,
		ReadTimeout:  timeoutDuration,
	}
	return srv.ListenAndServe()
}

func neuterDirectoryListing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}
