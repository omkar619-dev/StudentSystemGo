package middlewares

import (
	"fmt"
	"net/http"
	"time"
)

func ResponseTimeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Received request in response Time: %s %s\n", r.Method, r.URL.Path)
		// Middleware logic to measure response time can be added here
		start := time.Now()

		wrappedWriter := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		duration := time.Since(start)
		wrappedWriter.Header().Set("X-Response-Time", duration.String())

		next.ServeHTTP(wrappedWriter, r)
		// log the request details
		fmt.Printf("Method: %s, URL: %s, status: %d, Duration: %v\n ",r.Method,r.URL,wrappedWriter.statusCode,duration.String())
		fmt.Printf("Finished processing request in response Time: %s %s\n", r.Method, r.URL.Path)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}