// Package server handles the http server for the frontend
package server

// Much of the design was models after this blog post
// http://nicolasmerouze.com/how-to-render-json-api-golang-mongodb/

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"

	"dnscoffee/model"

	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
	"github.com/rs/cors"
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
	APIRequestsPerMinute: 4 * 60,
	APIMaxRequestHistory: 16384,
	APIRequestsBurst:     5,
}

// handler for catching a panic
// returns an HTTP code 500
func recoverHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				//log.Printf("panic: %+v", err)
				// notice that we're using 1, so it will actually log the where
				// the error happened, 0 = this function, we don't want that.
				pc, fn, line, _ := runtime.Caller(2)
				log.Printf("[panic] in %s[%s:%d] %v", runtime.FuncForPC(pc).Name(), fn, line, err)
				//log.Println("stacktrace from panic: \n" + string(debug.Stack()))
				//debug.PrintStack()
				// postgresql error check
				// pqErr, ok := err.(*pq.Error)
				// if ok {
				// 	log.Printf("%+v\n", pqErr)
				// }
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
	router *httprouter.Router

	// Alice chain of http handlers
	handlers alice.Chain

	listenAddr string

	apiConfig APIConfig
}

// New creates a new server object with the default (included) handlers
func New(listenAddr string, apiConfig APIConfig) (*Server, error) {
	server := &Server{
		listenAddr: listenAddr,
		apiConfig:  apiConfig,
	}

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://127.0.0.1:5353"},
	})

	// setup server
	server.router = httprouter.New()
	handlers := alice.New(
		c.Handler,
		//context.ClearHandler,
		//addContextHandler,
		makeTimeoutHandler(server.apiConfig.APITimeout),
		loggingHandler,
		recoverHandler,
	)
	// serve static content
	static := http.StripPrefix("/static/", http.FileServer(http.Dir("static")))
	server.router.Handler(http.MethodGet, "/static/*path", handlers.Then(neuterDirectoryListing(static)))

	// setup robots.txt
	server.router.Handler(http.MethodGet, "/robots.txt", handlers.ThenFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/robots.txt")
	}))

	// add rate limiting after static handler
	server.handlers = handlers.Append(makeThrottleHandler(
		server.apiConfig.APIRequestsPerMinute,
		server.apiConfig.APIRequestsBurst,
		server.apiConfig.APIMaxRequestHistory,
	))

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
	return http.ListenAndServe(s.listenAddr, s.router)
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
