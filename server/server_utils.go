package server

import (
	"dnscoffee/model"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"gopkg.in/throttled/throttled.v2"
	"gopkg.in/throttled/throttled.v2/store/memstore"
)

func neuterDirectoryListing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

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

// 404 not found handler
//func notFoundJSON(w http.ResponseWriter, r *http.Request) {
//	WriteJSONError(w, ErrNotFound)
//}

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
