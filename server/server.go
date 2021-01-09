// Package server handles the http server for the frontend
package server

import (
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

type APIConfig struct {
	APITimeout           int
	APIRequestsPerMinute int
	APIMaxRequestHistory int
	APIRequestsBurst     int
}

var DefaultAPIConfig = APIConfig{
	APITimeout:           60,
	APIRequestsPerMinute: 60,
	APIMaxRequestHistory: 16384,
	APIRequestsBurst:     10,
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
		router:     mux.NewRouter().StrictSlash(true),
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
}

// Get registers a HTTP GET to the router & handler
func (s *Server) Get(path string, fn http.HandlerFunc) {
	s.router.Handle(path, fn).Methods(http.MethodGet)
	s.router.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}).Methods(http.MethodHead)
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
	h = setProxyURLHost(h)
	// setup logging
	h = handlers.LoggingHandler(os.Stdout, h)
	// add recovery
	h = handlers.RecoveryHandler(handlers.PrintRecoveryStack(true))(h)
	// timeouts
	h = http.TimeoutHandler(h, timeoutDuration, ErrTimeout.Error())
	// cors
	h = handlers.CORS(handlers.AllowedOrigins([]string{"http://127.0.0.1:5353"}))(h)

	// rate limiting
	// TODO add rate limiting after static handler and possible the main page
	throttleHandler := makeThrottleHandler(
		s.apiConfig.APIRequestsPerMinute,
		s.apiConfig.APIRequestsBurst,
		s.apiConfig.APIMaxRequestHistory,
	)
	h = throttleHandler(h)

	// run server
	srv := &http.Server{
		Handler:      h,
		Addr:         s.listenAddr,
		WriteTimeout: timeoutDuration,
		ReadTimeout:  timeoutDuration,
	}
	return srv.ListenAndServe()
}
