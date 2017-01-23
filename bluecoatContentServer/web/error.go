package web

import (
	"encoding/json"
	"net/http"
)

// Errors is a list of errors
type Errors struct {
	Errors []*Error `json:"errors"`
}

// Error holds the info about a web error
type Error struct {
	ID     string `json:"id"`
	Status int    `json:"status"`
	Title  string `json:"title,omitempty"`
	Detail string `json:"detail,omitempty"`
}

func (e *Error) Error() string {
	return e.Title + ":" + e.Detail
}

// writeOK writes ok to reply
func writeOK(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ErrOK)
}

// writeError writes an error to the reply
func writeError(w http.ResponseWriter, err *Error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.Status)
	json.NewEncoder(w).Encode(err)
}

var (
	// ErrOK is returned for successful operations
	ErrOK = &Error{ID: "success", Status: 200}
	// ErrBadRequest is a generic bad request
	ErrBadRequest = &Error{"bad_request", 400, "Bad request", "Request body is not well-formed. It must be JSON."}
	// ErrAuth if not authenticated
	ErrAuth = &Error{"unauthorized", 401, "Unauthorized", "The request requires authorization"}
	// ErrCredentials if there are missing / wrong credentials
	ErrCredentials = &Error{"invalid_credentials", 401, "Invalid credentials", "Invalid username or password"}
	// ErrNotAcceptable wrong accept header
	ErrNotAcceptable = &Error{"not_acceptable", 406, "Not Acceptable", "Accept header must be set to 'application/json'."}
	// ErrUnsupportedMediaType wrong media type
	ErrUnsupportedMediaType = &Error{"unsupported_media_type", 415, "Unsupported Media Type", "Content-Type header must be set to: 'application/json'."}
	// ErrInternalServer if things go wrong on our side
	ErrInternalServer = &Error{"internal_server_error", 500, "Internal Server Error", "Something went wrong."}
)
