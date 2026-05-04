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
	"restapi/internal/dbmigrate"
	myredis "restapi/internal/redis"
	"restapi/internal/repository/sqlconnect"
	"restapi/internal/storage/photo"
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

	// Run pending schema migrations against PRIMARY only.
	// Replica picks up DDL via binlog replication.
	// Idempotent — no-op when schema is already current.
	if err := dbmigrate.Run(dbmigrate.Config{
		Host:     getEnv("DB_PRIMARY_HOST", "mysql_primary"),
		Port:     getEnv("DB_PORT", "3306"),
		User:     getEnv("DB_USER", "root"),
		Password: getEnv("DB_PASSWORD", ""),
		DBName:   getEnv("DB_NAME", "schooldb"),
	}); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	if err := myredis.InitRedis(); err != nil {
		log.Fatalf("failed to initialise redis: %v", err)
	}
	defer myredis.Close()

	// Initialise the S3 + CloudFront photo storage layer.
	// Fails fast if env vars are missing or the CF private key file is
	// unreadable. See docs/photo-flow.md for env var reference.
	if err := photo.Init(photo.Config{
		Bucket:          getEnv("S3_BUCKET", ""),
		Region:          getEnv("S3_REGION", "ap-south-1"),
		CFDomain:        getEnv("CF_DOMAIN", ""),
		CFKeyPairID:     getEnv("CF_KEY_PAIR_ID", ""),
		CFPrivateKeyPth: getEnv("CF_PRIVATE_KEY_PATH", "/etc/secrets/cloudfront_private_key.pem"),
	}); err != nil {
		log.Fatalf("failed to initialise photo storage: %v", err)
	}

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

