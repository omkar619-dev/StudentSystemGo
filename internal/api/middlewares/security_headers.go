package middlewares
import ("net/http" 
          "fmt")


func SecurityHeaders(next http.Handler) http.Handler {
		fmt.Println("SECURITY HEADERS MIDDLEWARE...")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Println("SECURITY HEADERS MIDDLEWARE BEING RETURNED...")
		w.Header().Set("X-DNS-Prefetch-Control","off")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Powered-By","Django")
		next.ServeHTTP(w, r)
		fmt.Println("SECURITY HEADERS MIDDLEWARE ENDED...")
	})
}

// basic example of how to use the middleware in your handlers
// func SecurityHeaders(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// next.ServeHTTP(w, r)
// 	})
// }