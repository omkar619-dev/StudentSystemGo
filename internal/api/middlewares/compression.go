package middlewares

import (
	"compress/gzip"
	"fmt"
	"net/http"
	"strings"
)

func Compression(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the client supports gzip encoding by looking at the "Accept-Encoding" header
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			// If the client does not support gzip, simply call the next handler
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") { 
					next.ServeHTTP(w, r)			
				}
			
			 w.Header().Set("Content-Encoding", "gzip")
			 gz := gzip.NewWriter(w)
			 defer gz.Close()
			 w = &gzipResponseWriter{ResponseWriter: w, Writer: gz}
			
		}
			 next.ServeHTTP(w, r)	
			 fmt.Printf("Finished processing request in compression: %s %s\n", r.Method, r.URL.Path)
		} )
	}

	type gzipResponseWriter struct {
		http.ResponseWriter
		Writer *gzip.Writer
	}

	func (w *gzipResponseWriter) Write(b []byte) (int, error) {
		return w.Writer.Write(b)
	}