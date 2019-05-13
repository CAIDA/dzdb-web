package server

// Much of the design was models after this blog post
// http://nicolasmerouze.com/how-to-render-json-api-golang-mongodb/

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"vdz-web/model"

	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
	"gopkg.in/throttled/throttled.v2"
	"gopkg.in/throttled/throttled.v2/store/memstore"
)

const (
	// Default environment variables
	defaultServerAddress        = "127.0.0.1:8080" // HTTP_LISTEN_ADDR
	defaultAPITimeout           = 60               // API_TIMEOUT
	defaultAPIRequestsPerMinute = 30               // API_REQUESTS_PER_MINUTE
	defaultAPIMaxRequestHistory = 16384            // API_MAX_HISTORY
	defaultAPIRequestsBurst     = 5                // API_REQUEST_BURST
)

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

// 404 not found handler
//func notFoundJSON(w http.ResponseWriter, r *http.Request) {
//	WriteJSONError(w, ErrNotFound)
//}

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
		VaryBy:      new(ipVaryBy),
		DeniedHandler: http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			WriteJSONError(w, ErrLimitExceeded)
		})),
	}

	return httpRateLimiter.RateLimit
}

// HandlerNotImplemented returns ErrNotImplemented as JSON
func HandlerNotImplemented(w http.ResponseWriter, r *http.Request) {
	WriteJSONError(w, ErrNotImplemented)
}

// WriteJSONError returns an error as JSON
// TODO make not all errors JSON
func WriteJSONError(w http.ResponseWriter, err *model.JSONError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.Status)
	json.NewEncoder(w).Encode(model.JSONErrors{Errors: []*model.JSONError{err}})
}

// WriteJSON writes JSON from data to the response
func WriteJSON(w http.ResponseWriter, data interface{}) {
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
	router *httprouter.Router

	// Alice chain of http handlers
	handlers alice.Chain

	// ListenAddr address to bind http server too
	ListenAddr string
}

// New creates a new server object with the default (included) handlers
func New() (*Server, error) {
	server := &Server{}

	// get settings
	server.ListenAddr = model.GetStringEnv("HTTP_LISTEN_ADDR", defaultServerAddress)
	apiTimeout, err := model.GetIntEnv("API_TIMEOUT", defaultAPITimeout)
	if err != nil {
		return nil, err
	}
	apiRequestsPerMinute, err := model.GetIntEnv("API_REQUESTS_PER_MINUTE", defaultAPIRequestsPerMinute)
	if err != nil {
		return nil, err
	}
	apiMaxHistory, err := model.GetIntEnv("API_MAX_HISTORY", defaultAPIMaxRequestHistory)
	if err != nil {
		return nil, err
	}
	apiRequestsBurst, err := model.GetIntEnv("API_REQUEST_BURST", defaultAPIRequestsBurst)
	if err != nil {
		return nil, err
	}

	// setup server
	server.router = httprouter.New()
	server.handlers = alice.New(
		//context.ClearHandler,
		//addContextHandler,
		makeTimeoutHandler(apiTimeout),
		loggingHandler,
		recoverHandler,
		makeThrottleHandler(apiRequestsPerMinute, apiRequestsBurst, apiMaxHistory),
	)
	//server.router.NotFound = notFoundJSON
	return server, nil
}

// Get registers a HTTP GET to the router & handler
func (s *Server) Get(path string, fn http.HandlerFunc) {
	handler := s.handlers.ThenFunc(fn)
	s.router.Handler(http.MethodGet, path, handler)
}

// Post registers a HTTP POST to the router & handler
func (s *Server) Post(path string, fn http.HandlerFunc) {
	handler := s.handlers.ThenFunc(fn)
	s.router.Handler(http.MethodPost, path, handler)
}

// Start Starts the server, blocking function
func (s *Server) Start() error {
	return http.ListenAndServe(s.ListenAddr, s.router)
}
