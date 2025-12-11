// Package middleware provides HTTP middleware for the API.
package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime/debug"
)

// ErrorResponse represents a standardized API error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// WriteError writes a JSON error response with the given status code.
func WriteError(w http.ResponseWriter, status int, errCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   errCode,
		Message: message,
	})
}

// WriteErrorWithDetails writes a JSON error response with additional details.
func WriteErrorWithDetails(w http.ResponseWriter, status int, errCode, message string, details any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   errCode,
		Message: message,
		Details: details,
	})
}

// ErrorRecovery is middleware that recovers from panics and returns a 500 error.
func ErrorRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v\n%s", err, debug.Stack())
				WriteError(w, http.StatusInternalServerError, "internal_error", "An unexpected error occurred")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Common error codes
const (
	ErrNotFound       = "not_found"
	ErrBadRequest     = "bad_request"
	ErrConflict       = "conflict"
	ErrInternalError  = "internal_error"
	ErrValidation     = "validation_error"
	ErrUnauthorized   = "unauthorized"
)



