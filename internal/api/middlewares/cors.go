package middlewares

import (
	"fmt"
	"net/http"
)

var allowedOrigins = []string{
	"http://localhost:3000",
	"http://example.com",
}


func Cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		fmt.Printf("Origin: %s\n", origin)
		if origin == "" || isOriginAllowed(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			http.Error(w, "Forbidden by CORS", http.StatusForbidden)
			return 
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Expose-Headers","Authorization")
		w.Header().Set("Access-Control-Max-Age", "3600")
		next.ServeHTTP(w, r)
	})
}
func isOriginAllowed(origin string) bool {
	for _, allowedOrigin := range allowedOrigins {
		if origin == allowedOrigin {
			return true
		}
	}
	return false
}