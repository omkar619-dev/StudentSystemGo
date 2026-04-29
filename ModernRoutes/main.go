package main

import (
	"fmt"
	"net/http"
)

func main() {

	mux := http.NewServeMux()

	// Method based routing
	mux.HandleFunc("POST /teachers/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w,"Teacher Created")
	})
	mux.HandleFunc("GET /teachers/{id}", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w,"Teacher Retrieved: %s", r.PathValue("id"))
	})
// while you are configuring your routes be aware of the order of your routes, if you have a route like /teachers/{id} and /teachers/ it will match the first one and not the second one, so you need to be careful while configuring your routes
	http.ListenAndServe(":7373",mux)
}
