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
	"regexp"

	"github.com/PuerkitoBio/throttled"
	"github.com/PuerkitoBio/throttled/store"
	"github.com/gorilla/context"
	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
)

// handler to ensure correct request accept headers
// also want to be able to accept text/plain
/*
func acceptHandler(next http.Handler) http.Handler {
  fn := func(w http.ResponseWriter, r *http.Request) {
    // We send a JSON-API error if the Accept header does not have a valid value.
    if r.Header.Get("Accept") != "application/vnd.api+json" {
      jsonErr := &Error{"not_acceptable", 406, "Not Acceptable", "Accept header must be set to 'application/vnd.api+json'."}
      w.Header().Set("Content-Type", "application/vnd.api+json")
      w.WriteHeader(jsonErr.Status)
      json.NewEncoder(w).Encode(Errors{[]*Error{jsonErr}})
      return
    }

    next.ServeHTTP(w, r)
  }

  return http.HandlerFunc(fn)
}*/

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
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		log.Printf("[%s] %s %q %v\n", ip, r.Method, r.RequestURI, t2.Sub(t1))
	}

	return http.HandlerFunc(fn)
}

// 404 not found handler
func notFoundJSON(w http.ResponseWriter, r *http.Request) {
	WriteJSONError(w, ErrNotFound)
}

// creates a TimeoutHandler using the provided sec timeout
func makeTimeoutHandler(sec int) func(http.Handler) http.Handler {
	timeout_error_json, err := json.Marshal(ErrTimeout)
	if err != nil {
		log.Fatal(err)
	}
	return func(h http.Handler) http.Handler {
		return http.TimeoutHandler(h, time.Duration(sec)*time.Second, string(timeout_error_json))
	}
}

// creates a throttled handler using the perMin limit on requests
func makeThrottleHandler(perMin int) func(http.Handler) http.Handler {
	th := throttled.RateLimit(throttled.PerMin(perMin), &throttled.VaryBy{RemoteAddr: true}, store.NewMemStore(1000))
	th.DeniedHandler = http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WriteJSONError(w, ErrLimitExceeded)
	}))
	return th.Throttle
}

// variables to hold common json errors
var (
	//ErrBadRequest           = &JSONError{"bad_request", 400, "Bad request", "Request body is not well-formed. It must be JSON."}
	//ErrUnauthorized         = &JSONError{"unauthorized", 401, "Unauthorized", "Access token is invalid."}
	ErrNotFound             = &JSONError{"not_found", 404, "Not found", "Route not found."}
	//ErrNotAcceptable        = &JSONError{"not_acceptable", 406, "Not acceptable", "Accept HTTP header must be \"application/vnd.api+json\"."}
	//ErrUnsupportedMediaType = &JSONError{"unsupported_media_type", 415, "Unsupported Media Type", "Content-Type header must be \"application/vnd.api+json\"."}
	ErrLimitExceeded  = &JSONError{"limit_exceeded", 429, "Too Many Requests", "To many requests, please wait and submit again."}
	ErrInternalServer = &JSONError{"internal_server_error", 500, "Internal Server Error", "Something went wrong."}
	ErrTimeout        = &JSONError{"timeout", 503, "Service Unavailable", "The request took longer than expected to process."}
)

func WriteJSONError(w http.ResponseWriter, err *JSONError) {
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(err.Status)
	json.NewEncoder(w).Encode(JSONErrors{[]*JSONError{err}})
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
		makeTimeoutHandler(server.config.API.Timeout),
		loggingHandler,
		recoverHandler,
		makeThrottleHandler(server.config.API.Requests_Per_Minute),
	)
	server.router.NotFound = notFoundJSON
	return server
}

// Adds a method to the router's GET handler but also adds it to the API index map
// description is the API function description
func (s *server) Get_Index(path, description string, fn http.HandlerFunc) {
	re := regexp.MustCompile(":[a-zA-Z0-9_]*")
	param_path := re.ReplaceAllStringFunc(path, func(s string) string { return fmt.Sprintf("{%s}", s[1:]) })
	s.index[description] = param_path
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
