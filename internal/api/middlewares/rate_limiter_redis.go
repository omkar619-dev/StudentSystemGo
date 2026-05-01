package middlewares

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	myredis "restapi/internal/redis"
)

// RedisRateLimit returns middleware that enforces a per-IP request quota
// using Redis as shared state across all app instances.
//
// Algorithm: fixed window. Each IP gets `limit` requests per `window` duration.
// Window is keyed by floor(now / window) — so requests bucket cleanly each minute
// (or whatever window is). Auto-expires via Redis TTL — no manual cleanup.
//
// Behaviour on Redis failure: FAIL OPEN. Logs the error, allows the request.
// Reasoning: a rate limiter should not take the whole API down. Availability > strictness.
//
// Usage:
//
//	rl := middlewares.RedisRateLimit("global", 100, time.Minute)
//	mux.Use(rl)
//
// The `name` param namespaces keys so multiple limiters can coexist
// (e.g., "global" 100/min + "login" 10/min on different routes).
func RedisRateLimit(name string, limit int, window time.Duration) func(http.Handler) http.Handler {
	if limit <= 0 || window <= 0 {
		log.Fatalf("RedisRateLimit: limit and window must be positive (got limit=%d window=%v)", limit, window)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Resolve the client IP. Behind Nginx, the real client IP comes from
			// the X-Forwarded-For / X-Real-IP headers Nginx sets — NOT r.RemoteAddr
			// (which would be Nginx's container IP).
			ip := clientIP(r)

			// Bucket key: window of the current time (e.g., minute 28293743).
			// All requests from the same IP within the same window share this counter.
			bucket := time.Now().Unix() / int64(window.Seconds())
			key := fmt.Sprintf("rl:%s:%s:%d", name, ip, bucket)

			// Bounded Redis call. If Redis hangs, we don't block the request forever.
			ctx, cancel := context.WithTimeout(r.Context(), 200*time.Millisecond)
			defer cancel()

			// Atomic INCR: returns new count after increment.
			// Two goroutines calling INCR at the same instant get sequential values
			// (1, 2 — never both 1). Redis is single-threaded internally.
			count, err := myredis.Client.Incr(ctx, key).Result()
			if err != nil {
				// Fail open: log and pass through. Don't punish users for our infra problem.
				log.Printf("[ratelimit:%s] redis err: %v — failing open", name, err)
				next.ServeHTTP(w, r)
				return
			}

			// On the first request of a window, set the TTL so the key auto-expires.
			// Subsequent requests in the same window don't reset TTL (idempotent).
			// `EXPIRE` is fire-and-forget — we don't care if it fails (worst case:
			// the key lives slightly longer, eaten by Redis maxmemory eviction later).
			if count == 1 {
				myredis.Client.Expire(ctx, key, window)
			}

			// Send rate-limit headers so clients can self-throttle.
			// These follow the IETF "RateLimit Header Fields" draft.
			remaining := int64(limit) - count
			if remaining < 0 {
				remaining = 0
			}
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", (bucket+1)*int64(window.Seconds())))

			if count > int64(limit) {
				// Tell the client when to retry (start of next window)
				retryAfter := int64(window.Seconds()) - (time.Now().Unix() % int64(window.Seconds()))
				w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// PathOnly wraps a middleware so it only fires when the request path matches
// one of `paths`. Other paths pass through the wrapper untouched.
//
// Used to apply different rate limit policies to specific endpoints — e.g.,
// stricter limit on /execs/login (anti-brute-force) while leaving other routes
// to the global limiter.
//
// Example:
//
//	loginLimit := PathOnly(
//	    []string{"/execs/login"},
//	    RedisRateLimit("login", 10, time.Minute),
//	)
func PathOnly(paths []string, mw func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	// Build a set for O(1) lookup. Using map[string]struct{} (the empty-struct
	// pattern) takes 0 bytes per entry — Go's idiom for "set of strings".
	pathSet := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		pathSet[p] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		// Pre-wrap once at setup time (not per request) — we want to share the
		// inner middleware's state (e.g., rate limit counters) across all requests.
		wrapped := mw(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, matches := pathSet[r.URL.Path]; matches {
				wrapped.ServeHTTP(w, r) // apply the wrapped middleware
				return
			}
			next.ServeHTTP(w, r) // skip the wrapped middleware
		})
	}
}

// clientIP returns the real client IP, honoring X-Forwarded-For (set by Nginx
// in front of the app). Falls back to RemoteAddr.
//
// SECURITY NOTE: blindly trusting X-Forwarded-For lets clients spoof their IP
// by setting the header themselves. Only safe when there's a known proxy in
// front (Nginx) that overwrites the header. Our Nginx config sets this; direct
// requests to the app port (not exposed publicly) shouldn't happen.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// XFF can be a comma-separated list: "client, proxy1, proxy2".
		// First entry is the original client.
		if idx := strings.IndexByte(xff, ','); idx >= 0 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	// Fall back to RemoteAddr — strip port if present
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
