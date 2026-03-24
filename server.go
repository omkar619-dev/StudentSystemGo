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
			// fmt.Println(r.URL.Path)
			// path := strings.TrimPrefix(r.URL.Path, "/teachers/")
			// userID := strings.TrimPrefix(path, "/")
			// fmt.Printf("Split parts: %v\n", userID)
			// fmt.Println("Query params",r.URL.Query())
			// queryParams := r.URL.Query()
			// sortBy := queryParams.Get("sortby")
			// key := queryParams.Get("key")
			// sortOrder := queryParams.Get("sortorder")
			// fmt.Printf("Sort by: %s, Key: %s, Sort order: %s\n", sortBy, key, sortOrder)
			// for key, value := range queryParams {
			// 	fmt.Printf("Query param: %s = %s\n", key, value)
			// }
			w.Write([]byte("Hello GET Method on teachers route!"))
			fmt.Println("Hello GET Method on teachers route!")
			return
		case http.MethodPost:
			//parse form
			// err := r.ParseForm()
			// if err != nil {
			// 	http.Error(w,"Error parsing form",http.StatusBadRequest)
			// 	return
			// }
			// fmt.Printf("Form data: %v\n", r.Form)

			// // prepare response data
			// response := make(map[string]interface{})
			// for key, value := range r.Form {
			// 	if len(value) > 0 {
			// 		response[key] = value[0]
			// 	}
			// }
			// fmt.Printf("Response data: %v\n", response)

			// // RAW BODY 
			// body, err := io.ReadAll(r.Body)
			// if err != nil {
			// 	return
			// }
			// defer r.Body.Close()
			// fmt.Printf("Raw body: %s\n", string(body))

			// // if you expect json body, you can unmarshal it into a struct
			// var userInstance User
			// err = json.Unmarshal(body, &userInstance)
			// if err != nil {
			// 	return
			// }
			// fmt.Printf("User instance: %+v\n", userInstance)

			// // acccessing request details
			// fmt.Println("Body", r.Body)
			// fmt.Println("Form", r.Form)
			// fmt.Println("Header", r.Header)
			// fmt.Println("Context", r.Context())
			// fmt.Println("Method", r.Method)
			// fmt.Println("HOST", r.Host)
			// fmt.Println("Protocol", r.Proto)
			// fmt.Println("RemoteAddr", r.RemoteAddr)
			// fmt.Println("RequestURI", r.RequestURI)
			// fmt.Println("URL", r.URL)
			// fmt.Println("Port", r.URL.Port())

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
			//parse form
			// err := r.ParseForm()
			// if err != nil {
			// 	http.Error(w,"Error parsing form",http.StatusBadRequest)
			// 	return
			// }
			// fmt.Printf("Form data: %v\n", r.Form)

			// // prepare response data
			// response := make(map[string]interface{})
			// for key, value := range r.Form {
			// 	if len(value) > 0 {
			// 		response[key] = value[0]
			// 	}
			// }
			// fmt.Printf("Response data: %v\n", response)

			// // RAW BODY 
			// body, err := io.ReadAll(r.Body)
			// if err != nil {
			// 	return
			// }
			// defer r.Body.Close()
			// fmt.Printf("Raw body: %s\n", string(body))

			// // if you expect json body, you can unmarshal it into a struct
			// var userInstance User
			// err = json.Unmarshal(body, &userInstance)
			// if err != nil {
			// 	return
			// }
			// fmt.Printf("User instance: %+v\n", userInstance)

			// // acccessing request details
			// fmt.Println("Body", r.Body)
			// fmt.Println("Form", r.Form)
			// fmt.Println("Header", r.Header)
			// fmt.Println("Context", r.Context())
			// fmt.Println("Method", r.Method)
			// fmt.Println("HOST", r.Host)
			// fmt.Println("Protocol", r.Proto)
			// fmt.Println("RemoteAddr", r.RemoteAddr)
			// fmt.Println("RequestURI", r.RequestURI)
			// fmt.Println("URL", r.URL)
			// fmt.Println("Port", r.URL.Port())

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
			//parse form
			// err := r.ParseForm()
			// if err != nil {
			// 	http.Error(w,"Error parsing form",http.StatusBadRequest)
			// 	return
			// }
			// fmt.Printf("Form data: %v\n", r.Form)

			// // prepare response data
			// response := make(map[string]interface{})
			// for key, value := range r.Form {
			// 	if len(value) > 0 {
			// 		response[key] = value[0]
			// 	}
			// }
			// fmt.Printf("Response data: %v\n", response)

			// // RAW BODY 
			// body, err := io.ReadAll(r.Body)
			// if err != nil {
			// 	return
			// }
			// defer r.Body.Close()
			// fmt.Printf("Raw body: %s\n", string(body))

			// // if you expect json body, you can unmarshal it into a struct
			// var userInstance User
			// err = json.Unmarshal(body, &userInstance)
			// if err != nil {
			// 	return
			// }
			// fmt.Printf("User instance: %+v\n", userInstance)

			// // acccessing request details
			// fmt.Println("Body", r.Body)
			// fmt.Println("Form", r.Form)
			// fmt.Println("Header", r.Header)
			// fmt.Println("Context", r.Context())
			// fmt.Println("Method", r.Method)
			// fmt.Println("HOST", r.Host)
			// fmt.Println("Protocol", r.Proto)
			// fmt.Println("RemoteAddr", r.RemoteAddr)
			// fmt.Println("RequestURI", r.RequestURI)
			// fmt.Println("URL", r.URL)
			// fmt.Println("Port", r.URL.Port())

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