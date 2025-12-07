package middleware

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture the status code.
// It also implements http.Hijacker to support WebSocket connections.
type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// Hijack implements http.Hijacker interface to support WebSocket upgrades.
// This delegates to the underlying ResponseWriter if it supports hijacking.
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("response writer does not implement http.Hijacker")
	}
	return h.Hijack()
}

// Flush implements http.Flusher interface for streaming responses.
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Logging is middleware that logs HTTP requests.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status
		wrapped := &responseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		// Process request
		next.ServeHTTP(wrapped, r)

		// Log request details
		duration := time.Since(start)
		log.Printf(
			"%s %s %d %d %s",
			r.Method,
			r.URL.Path,
			wrapped.status,
			wrapped.size,
			duration,
		)
	})
}

