package middlewares
import (
	"fmt"
	"net/http"
	"strings"
)

type HPPOptions struct {
	// Add any options you want to configure for the HPP middleware
	CheckQuery bool
	CheckBody  bool
	CheckBodyOnlyForContentType string
	Whitelist []string
}

func Hpp(options HPPOptions) func(http.Handler) http.Handler {
		fmt.Println("HPP MIDDLEWARE ...")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { 
					fmt.Println("HPP MIDDLEWARE BEING RETURNED...")

			// Implement HPP logic here based on the options provided
			if options.CheckBody && r.Method == http.MethodPost &&  isCorrectContentType(r,options.CheckBodyOnlyForContentType){
				// Check for duplicate parameters in the request body
				filterBodyParams(r,options.Whitelist)
			}
			if options.CheckQuery && r.URL.Query() != nil{
				// Check for duplicate parameters in the query string
				filterQueryParams(r,options.Whitelist)
			}
			next.ServeHTTP(w, r)
			fmt.Println("HPP MIDDLEWARE ENDED...")

		})
	}
}

func isCorrectContentType(r *http.Request, contentType string) bool {
	return strings.Contains(r.Header.Get("Content-Type"),contentType)
}

func filterBodyParams(r *http.Request, whitelist []string) {
	err := r.ParseForm()
	if err != nil {
		fmt.Printf("Error parsing form: %v\n", err)
		return
	}
	for k,v := range r.Form {
		if len(v) > 1 {
			r.Form.Set(k, v[0]) // first value
		}
			if !isWhiteListedParam(k, whitelist) {
				delete(r.Form, k)
			}
		}
	}

func filterQueryParams(r *http.Request, whitelist []string) {
	query := r.URL.Query()
	for k,v := range query {
		if len(v) > 1 {
			query.Set(k, v[0]) // first value
		}
			if !isWhiteListedParam(k, whitelist) {
				query.Del(k)
			}
		}
		r.URL.RawQuery = query.Encode()
	}


func isWhiteListedParam(param string, whitelist []string) bool {
	for _, v := range whitelist { 
		if v == param {
			return true
		} 
	}
	return false
}