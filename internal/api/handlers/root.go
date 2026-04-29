package handlers

import (
	"fmt"
	"net/http"
)
func RootHandler(w http.ResponseWriter, r *http.Request) {
		// fmt.Fprintf(w, "Hello, Root route!")
		w.Write([]byte("Welcome to the School API!"))
		fmt.Println("Hello root route")

	}