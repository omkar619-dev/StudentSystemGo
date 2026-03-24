package middlewares

import (
	"fmt"
	"net/http"
	"sync"
	"time"
	"strings"
)

type rateLimiter struct {
	mu sync.Mutex
	visitors map[string] int
	limit int
	resetTime time.Duration
}

func NewRateLimiter(limit int, resetTime time.Duration) *rateLimiter {
	rl := &rateLimiter{
		visitors: make(map[string] int),
		limit:    limit,
		resetTime: resetTime,
	}
	go rl.resetVisitorCount()
	return rl
}

func (rl *rateLimiter) resetVisitorCount() {
	for {
		time.Sleep(rl.resetTime)	
		rl.mu.Lock()
		rl.visitors = make(map[string] int) 
		rl.mu.Unlock()
	} 
}

func (rl *rateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rl.mu.Lock()
		defer rl.mu.Unlock()

		visitorIP := r.RemoteAddr
		if colon := strings.LastIndex(visitorIP, ":"); colon != -1 {
            visitorIP = visitorIP[:colon]
        }
		rl.visitors[visitorIP]++
		fmt.Printf("Visitor count from %v is %v\n", visitorIP, rl.visitors[visitorIP])

		if rl.visitors[visitorIP] > rl.limit {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	}) 
}