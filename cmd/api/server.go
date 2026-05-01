package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"restapi/internal/api/middlewares"
	"restapi/internal/api/router"
	myredis "restapi/internal/redis"
	"restapi/internal/repository/sqlconnect"
	"restapi/pkg/utils"

	"github.com/joho/godotenv"
)

func getEnv(key, fallback string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return fallback
}

func main() {
	// .env is optional — container env vars override anyway

	_ = godotenv.Load(); 
	

	if err := sqlconnect.InitDB(); err != nil {
		log.Fatalf("failed to initialise database: %v", err)
	}

	if err := myredis.InitRedis(); err != nil {
		log.Fatalf("failed to initialise redis: %v", err)
	}
	defer myredis.Close()

	port := ":3000"

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	// Redis-backed rate limiters (replaces old in-memory rl).
	// Two policies:
	//   - global: 100 req/min per IP, applies to ALL routes
	//   - login:  10 req/min per IP, applies ONLY to /execs/login (brute-force defense)
	// State is in Redis → all 3 app instances share counters → no bypass via LB.
	globalRateLimit := middlewares.RedisRateLimit("global", 100, time.Minute)
	loginRateLimit := middlewares.PathOnly(
		[]string{"/execs/login"},
		middlewares.RedisRateLimit("login", 10, time.Minute),
	)

	hppOptions := middlewares.HPPOptions{
		CheckBody: true,
		CheckQuery: true,
		CheckBodyOnlyForContentType: "application/x-www-form-urlencoded",
		Whitelist: []string{"sortby","sortOrder","name","age","class", "first_name", "last_name","subject"},
	}

	router := router.MainRouter()
	// secureMux := middlewares.Cors(rl.Middleware(middlewares.ResponseTimeMiddleware(middlewares.Compression(middlewares.Hpp(hppOptions)(middlewares.SecurityHeaders(mux)))))),
	jwtMiddleware := middlewares.MiddlewaresExcludePaths(middlewares.JWTMiddleware,"/execs/login","/execs/forgotpassword","/execs/resetpassword/reset","/healthz")
	// secureMux := utils.ApplyMiddlewares(
	// 	router,
	// 	middlewares.SecurityHeaders,
	// 	middlewares.Compression,
	// 	middlewares.Hpp(hppOptions),
	// 	middlewares.XSSMiddleware,
	// 	jwtMiddleware,
	// 	middlewares.ResponseTimeMiddleware,
	// 	rl.Middleware,
	// 	middlewares.Cors,
	// )
	// Middleware order matters! Outermost first, innermost last.
	// Request flow: Cors → ResponseTime → globalRateLimit → loginRateLimit
	//             → jwtMiddleware → XSS → HPP → Compression → SecurityHeaders → router
	// Rate limiters run EARLY (before expensive auth/db work) so we reject
	// over-limit requests fast.
	secureMux := utils.ApplyMiddlewares(
		router,
		middlewares.SecurityHeaders,
		middlewares.Compression,
		middlewares.Hpp(hppOptions),
		middlewares.XSSMiddleware,
		jwtMiddleware,
		loginRateLimit,    // stricter limit on /execs/login only
		globalRateLimit,   // 100/min on everything
		middlewares.ResponseTimeMiddleware,
		middlewares.Cors,
	)
	
// secureMux := middlewares.XSSMiddleware(router)
useTLS := getEnv("USE_TLS", "false") == "true"
server := &http.Server{
		Addr:      port,
		Handler:   secureMux, 
		// TLSConfig: tlsConfig,
	}
	if useTLS {
		server.TLSConfig = tlsConfig
		cert := getEnv("CERT_FILE", "cert.pem")
		key := getEnv("KEY_FILE", "key.pem")
		fmt.Printf("Starting HTTPS server on %s\n", port)
		log.Fatal(server.ListenAndServeTLS(cert, key))
	} else {
		fmt.Printf("Starting HTTP server on %s (TLS off)\n", port)
		log.Fatal(server.ListenAndServe())
	}
}

