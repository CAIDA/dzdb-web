package server

import "vdz-web/model"

// variables to hold common json errors
var (
	//ErrBadRequest           = &JSONError{"bad_request", 400, "Bad request", "Request body is not well-formed. It must be JSON."}
	//ErrUnauthorized         = &JSONError{"unauthorized", 401, "Unauthorized", "Access token is invalid."}
	ErrNotFound         = model.NewJSONError("not_found", 404, "Not found", "Route not found.")
	ErrResourceNotFound = model.NewJSONError("resource_not_found", 404, "Not found", "Resource not found.")
	ErrLimitExceeded    = model.NewJSONError("limit_exceeded", 429, "Too Many Requests", "To many requests, please wait and submit again.")
	ErrInternalServer   = model.NewJSONError("internal_server_error", 500, "Internal Server Error", "Something went wrong.")
	ErrNotImplemented   = model.NewJSONError("not_implemented", 501, "Not Implemented", "The server does not support the functionality required to fulfill the request. It may not have been implemented yet")
	ErrTimeout          = model.NewJSONError("timeout", 503, "Service Unavailable", "The request took longer than expected to process.")
)
