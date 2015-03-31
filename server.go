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
	"time"

	"github.com/PuerkitoBio/throttled"
	"github.com/PuerkitoBio/throttled/store"
	"github.com/gorilla/context"
	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
)

// handler for catching a panic
// returns an HTTP code 500
func recoverHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic: %+v", err)
				http.Error(w, http.StatusText(500), 500)
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
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		log.Printf("[%s] %s %q %v\n", ip, r.Method, r.RequestURI, t2.Sub(t1))
	}

	return http.HandlerFunc(fn)
}

// creates a TimeoutHandler using the provided sec timeout
func makeTimeoutHandler(sec int) func(http.Handler) http.Handler {
	// TODO make json error
	return func(h http.Handler) http.Handler {
		return http.TimeoutHandler(h, time.Duration(sec)*time.Second, "timed out")
	}
}

// creates a throttled handler using the perMin limit on requests
func makeThrottleHandler(perMin int) func(http.Handler) http.Handler {
	th := throttled.RateLimit(throttled.PerMin(perMin), &throttled.VaryBy{RemoteAddr: true}, store.NewMemStore(1000))
	// TODO make json error
	th.DeniedHandler = http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "limit exceeded", 429)
	}))
	return th.Throttle
}

// struct for holding server resources
type server struct {
	// basic router which is extended with many functions in server
	// should not be used by external functions
	// all communication with the server's router should be done with server methods
	router *httprouter.Router

	// used for creating the API index
	index map[string]string

	// Alice chain of http handlers
	handlers alice.Chain

	// Configuration
	config *Config
}

// creates a new server object with the default (included) handlers
func NewServer(config *Config) *server {
	server := &server{}
	server.index = make(map[string]string)
	server.router = httprouter.New()
	server.config = config
	server.handlers = alice.New(
		context.ClearHandler,
		loggingHandler,
		recoverHandler,
		makeTimeoutHandler(server.config.API.API_Timeout),
		makeThrottleHandler(server.config.API.Requests_PerMin),
	)
	return server
}

// Adds a method to the router's GET handler but also adds it to the API index map
// description is the API function description
func (s *server) Get_Index(path, description string, fn http.HandlerFunc) {
	s.index[description] = path
	s.Get(path, s.handlers.ThenFunc(fn))
}

// add a method to the router's GET handler
func (s *server) Get(path string, handler http.Handler) {
	s.router.GET(path,
		func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
			context.Set(r, "params", ps)
			handler.ServeHTTP(w, r)
		})
}

// Index handler
// Displays the map of the API methods available
func (s *server) Index(w http.ResponseWriter, req *http.Request) {
	err := json.NewEncoder(w).Encode(s.index)
	if err != nil && err != http.ErrHandlerTimeout {
		panic(err)
	}
}

// Starts the server
// blocking function
func (s *server) Start() error {
	s.Get("/", s.handlers.ThenFunc(s.Index))
	return http.ListenAndServe(fmt.Sprintf("%s:%d", s.config.Http.IP, s.config.Http.Port), s.router)
}
