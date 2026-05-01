// Package redis provides a singleton Redis client used across the app.
// Same pattern as internal/repository/sqlconnect/sqlconfig.go for the DB pool.
//
// Why singleton: one *redis.Client → one connection pool, shared across all
// goroutines. *redis.Client is thread-safe and pools TCP connections internally.
package redis

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client is the shared Redis client used by middleware/handlers.
// Initialised once at app startup via InitRedis.
var Client *redis.Client

// InitRedis reads REDIS_URL from env, opens a pooled Redis connection,
// and verifies connectivity with a PING. Call once at startup.
//
// REDIS_URL format examples:
//
//	redis://localhost:6379          (no auth, default DB 0)
//	redis://:password@redis:6379/0  (auth, db 0)
//	redis://redis:6379/2            (db 2)
//
// If REDIS_URL is empty, defaults to redis://localhost:6379.
func InitRedis() error {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		url = "redis://localhost:6379"
	}

	opts, err := redis.ParseURL(url)
	if err != nil {
		return fmt.Errorf("invalid REDIS_URL: %w", err)
	}

	// Tune the pool. Defaults are usually fine but we set explicit values
	// so behaviour doesn't drift if go-redis defaults change.
	opts.PoolSize = 10                          // max active connections
	opts.MinIdleConns = 2                       // always keep 2 idle (avoid cold-start penalty)
	opts.PoolTimeout = 4 * time.Second          // wait up to 4s if pool is exhausted
	opts.ReadTimeout = 200 * time.Millisecond   // strict — rate limiter must be fast
	opts.WriteTimeout = 200 * time.Millisecond
	opts.DialTimeout = 2 * time.Second          // first connect

	Client = redis.NewClient(opts)

	// Verify reachable. PING is the cheapest possible health check.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}

	log.Printf("Connected to Redis at %s (pool size %d)", opts.Addr, opts.PoolSize)
	return nil
}

// Close shuts down the connection pool. Call from main()'s defer.
func Close() error {
	if Client == nil {
		return nil
	}
	return Client.Close()
}
