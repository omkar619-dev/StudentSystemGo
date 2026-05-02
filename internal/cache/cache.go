// Package cache provides a thin generic cache-aside helper backed by Redis.
//
// Pattern:
//
//	GetOrFetch(ctx, "teachers:list", 5*time.Minute, func() (any, error) {
//	    return sqlconnect.GetTeachersDBHandler(...)
//	})
//
// On cache hit: returns value immediately, never calls the loader.
// On cache miss: calls loader, stores result with TTL, returns value.
// On Redis error: logs and falls through to loader (fail-open).
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	myredis "restapi/internal/redis"
)

// Default TTL for cached list endpoints.
// Tradeoff: longer = better hit rate but staler data.
// 5 min balances freshness with effectiveness for school-management workload.
const DefaultTTL = 5 * time.Minute

// Key prefixes — keep keys namespaced so cache management commands work.
// e.g., `redis-cli --scan --pattern 'cache:teachers:*'` lists all teacher caches.
const KeyPrefix = "cache:"

// GetOrFetch implements cache-aside.
//
//   - key: full Redis key (caller's responsibility to namespace)
//   - ttl: how long to cache after a fresh load
//   - loader: function that produces the fresh value when cache misses
//
// The loader's return value is JSON-serialised before storing.
// On cache hit, the cached JSON is unmarshalled into `dest`.
//
// `dest` MUST be a pointer to whatever type the loader returns — same shape
// as you'd pass to json.Unmarshal.
func GetOrFetch(
	ctx context.Context,
	key string,
	ttl time.Duration,
	dest any,
	loader func() (any, error),
) error {
	if myredis.Client == nil {
		// Cache layer not initialised. Fall through to loader.
		// (Defensive — should never happen if InitRedis was called at startup.)
		return loadAndAssign(dest, loader)
	}

	// ── 1. Try cache ──────────────────────────────────────
	cachedJSON, err := myredis.Client.Get(ctx, key).Result()
	switch {
	case err == redis.Nil:
		// Key doesn't exist — cache miss. Fall through to loader below.
	case err != nil:
		// Redis is sick. Log and fall through (fail-open).
		log.Printf("[cache] Redis GET err for %s: %v — falling through to DB", key, err)
	default:
		// Cache HIT — unmarshal JSON into dest.
		if err := json.Unmarshal([]byte(cachedJSON), dest); err == nil {
			return nil
		}
		// Bad cached value (corruption?) — log and continue to refresh.
		log.Printf("[cache] bad JSON in %s, refreshing: %v", key, err)
	}

	// ── 2. Cache miss — call loader ───────────────────────
	value, err := loader()
	if err != nil {
		return err
	}

	// Assign to dest via JSON round-trip so types match exactly.
	bytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache: marshal: %w", err)
	}
	if err := json.Unmarshal(bytes, dest); err != nil {
		return fmt.Errorf("cache: unmarshal into dest: %w", err)
	}

	// ── 3. Populate cache with TTL (best effort) ──────────
	// SET with EX (expiry). Don't fail the request if cache write fails.
	if err := myredis.Client.Set(ctx, key, bytes, ttl).Err(); err != nil {
		log.Printf("[cache] SET err for %s: %v (data still served)", key, err)
	}
	return nil
}

// Invalidate deletes one or more keys from cache. Used after writes.
// Best-effort — logs but doesn't fail if Redis is down.
func Invalidate(ctx context.Context, keys ...string) {
	if myredis.Client == nil || len(keys) == 0 {
		return
	}
	if err := myredis.Client.Del(ctx, keys...).Err(); err != nil {
		log.Printf("[cache] DEL err for %v: %v", keys, err)
	}
}

// InvalidatePattern deletes all keys matching a glob pattern (e.g., "cache:teachers:*").
// Used when a list cache + many individual caches all need to be flushed.
//
// ⚠️ Performance: uses SCAN, which iterates the keyspace. For 1000s of keys,
// can take seconds. For our use case (handful of keys), negligible.
func InvalidatePattern(ctx context.Context, pattern string) {
	if myredis.Client == nil {
		return
	}
	iter := myredis.Client.Scan(ctx, 0, pattern, 100).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		log.Printf("[cache] SCAN err for %s: %v", pattern, err)
		return
	}
	if len(keys) > 0 {
		Invalidate(ctx, keys...)
	}
}

// loadAndAssign is the fallback when cache is unavailable.
func loadAndAssign(dest any, loader func() (any, error)) error {
	value, err := loader()
	if err != nil {
		return err
	}
	bytes, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, dest)
}
