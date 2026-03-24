package main

import (
	
	"fmt"
	"crypto/tls"
	"log"
	"net/http"
	"restapi/internal/api/middlewares"
	"time"
	
)

type User struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
	City string `json:"city"`
}
func rootHandler(w http.ResponseWriter, r *http.Request) {
		// fmt.Fprintf(w, "Hello, Root route!")
		w.Write([]byte("Hello, Root route!"))
		fmt.Println("Hello root route")

	}

	func teachersHandler(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Request method: %s\n", r.Method)
		switch r.Method {
		case http.MethodGet:
			w.Write([]byte("Hello GET Method on teachers route!"))
			fmt.Println("Hello GET Method on teachers route!")
			return
		case http.MethodPost:
			w.Write([]byte("Hello POST Method on teachers route!"))
			fmt.Println("Hello POST Method on teachers route!")
			return
		case http.MethodPut:
			w.Write([]byte("Hello PUT Method on teachers route!"))
			fmt.Println("Hello PUT Method on teachers route!")
			return
		case http.MethodDelete:
			w.Write([]byte("Hello DELETE Method on teachers route!"))
			fmt.Println("Hello DELETE Method on teachers route!")
			return
		case http.MethodPatch:
			w.Write([]byte("Hello PATCH Method on teachers route!"))
			fmt.Println("Hello PATCH Method on teachers route!")
			return
		}
		w.Write([]byte("Hello, Teachers route!"))
		fmt.Println("Hello teachers route")

	}

	func studentsHandler(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Write([]byte("Hello GET Method on students route!"))
			fmt.Println("Hello GET Method on students route!")
			return
		case http.MethodPost:
			w.Write([]byte("Hello POST Method on students route!"))
			fmt.Println("Hello POST Method on students route!")
			return
		case http.MethodPut:
			w.Write([]byte("Hello PUT Method on students route!"))
			fmt.Println("Hello PUT Method on students route!")
			return
		case http.MethodDelete:
			w.Write([]byte("Hello DELETE Method on students route!"))
			fmt.Println("Hello DELETE Method on students route!")
			return
		case http.MethodPatch:
			w.Write([]byte("Hello PATCH Method on students route!"))
			fmt.Println("Hello PATCH Method on students route!")
			return
		}
		w.Write([]byte("Hello, Students route!"))
		fmt.Println("Hello students route")

	}

	func execsHandler(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Write([]byte("Hello GET Method on execs route!"))
			fmt.Println("Hello GET Method on execs route!")
			return
		case http.MethodPost:
			w.Write([]byte("Hello POST Method on execs route!"))
			fmt.Println("Hello POST Method on execs route!")
			return
		case http.MethodPut:
			w.Write([]byte("Hello PUT Method on execs route!"))
			fmt.Println("Hello PUT Method on execs route!")
			return
		case http.MethodDelete:
			w.Write([]byte("Hello DELETE Method on execs route!"))
			fmt.Println("Hello DELETE Method on execs route!")
			return
		case http.MethodPatch:
			w.Write([]byte("Hello PATCH Method on execs route!"))
			fmt.Println("Hello PATCH Method on execs route!")
			return
		}
		w.Write([]byte("Hello, Execs route!"))
		fmt.Println("Hello execs route")

	}
func main() {

	port := ":3000"
	cert := "cert.pem"
	key := "key.pem"

	mux := http.NewServeMux()


	mux.HandleFunc("/", rootHandler)

	mux.HandleFunc("/teachers/", teachersHandler)

	mux.HandleFunc("/students/", studentsHandler)

	mux.HandleFunc("/execs/", execsHandler)

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	rl := middlewares.NewRateLimiter(5, time.Minute)
	//create custom server
	server := &http.Server{
		Addr:      port,
		Handler: rl.Middleware(middlewares.Compression(middlewares.ResponseTimeMiddleware(middlewares.Cors(mux)))),
		TLSConfig: tlsConfig,
	}

	fmt.Printf("Starting server on port %s\n", port)
	err := server.ListenAndServeTLS(cert, key)
	if err != nil {
		log.Fatal("error starting the server")
	}
}