package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"restapi/internal/api/middlewares"
	"restapi/internal/api/router"
	"restapi/internal/repository/sqlconnect"
	"restapi/pkg/utils"
	"os"
	// "time"

	"github.com/joho/godotenv"
	// "golang.org/x/text/secure"
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

	port := ":3000"

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	// rl := middlewares.NewRateLimiter(5, time.Minute)

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
	secureMux := utils.ApplyMiddlewares(
		router,
		middlewares.SecurityHeaders,
		middlewares.Compression,
		middlewares.Hpp(hppOptions),
		middlewares.XSSMiddleware,
		jwtMiddleware,
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

