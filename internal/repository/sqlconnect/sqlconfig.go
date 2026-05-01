package sqlconnect

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// Write is the connection pool to the PRIMARY database.
// Use for: INSERT, UPDATE, DELETE, transactions, anything that mutates.
// Also use for reads where staleness is unacceptable (login, anti-fraud, etc.).
var Write *sql.DB

// Read is the connection pool to the REPLICA database.
// Use for: SELECT queries where milliseconds-of-staleness is acceptable.
//
// ⚠️ IMPORTANT — Replication is asynchronous. Reads here can be a few
// hundred milliseconds (or seconds, under load) behind primary. If you
// JUST wrote something and need to read it back IMMEDIATELY, use Write.
//
// Replica is read_only=ON; writes will fail with an error.
var Read *sql.DB

// InitDB reads connection config from env vars, opens TWO pools (primary +
// replica), pings both, and stores them in package globals Write and Read.
// Call once at startup before serving traffic.
func InitDB() error {
	primaryHost := getEnv("DB_PRIMARY_HOST", "mysql_primary")
	replicaHost := getEnv("DB_REPLICA_HOST", "mysql_replica")
	port := getEnv("DB_PORT", "3306")
	user := getEnv("DB_USER", "root")
	pass := getEnv("DB_PASSWORD", "")
	name := getEnv("DB_NAME", "schooldb")

	// ── Open the PRIMARY pool ────────────────────────────────
	primaryDSN := buildDSN(user, pass, primaryHost, port, name)
	pdb, err := openAndTune(primaryDSN)
	if err != nil {
		return fmt.Errorf("primary open: %w", err)
	}
	Write = pdb
	fmt.Printf("Connected to PRIMARY at %s:%s/%s\n", primaryHost, port, name)

	// ── Open the REPLICA pool ────────────────────────────────
	replicaDSN := buildDSN(user, pass, replicaHost, port, name)
	rdb, err := openAndTune(replicaDSN)
	if err != nil {
		return fmt.Errorf("replica open: %w", err)
	}
	Read = rdb
	fmt.Printf("Connected to REPLICA at %s:%s/%s\n", replicaHost, port, name)

	return nil
}

// buildDSN constructs a MariaDB/MySQL connection string from parts.
func buildDSN(user, pass, host, port, name string) string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?allowCleartextPasswords=true&tls=skip-verify&parseTime=true",
		user, pass, host, port, name,
	)
}

// openAndTune opens a sql.DB pool, applies sensible limits, and verifies
// reachability with Ping().
func openAndTune(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	return db, nil
}

// ConnectDb returns the WRITE pool (primary).
// Kept for backward compatibility — existing code that doesn't care about
// read/write split gets the write pool, which is always safe.
//
// New code that wants the replica should use ConnectReadDb() instead.
//
// DO NOT call db.Close() on the returned pool — it's shared across all
// goroutines.
func ConnectDb(dbname string) (*sql.DB, error) {
	if Write == nil {
		return nil, fmt.Errorf("database not initialised; call sqlconnect.InitDB at startup")
	}
	return Write, nil
}

// ConnectReadDb returns the REPLICA pool (read-only, slightly stale).
// Use for SELECT queries where minor staleness (~ms-seconds) is acceptable.
//
// Examples of GOOD use:
//   - GET /teachers — listing teachers, no urgency
//   - GET /students/{id} — fetching a student profile
//   - dashboards, reports, search
//
// Examples of BAD use (use Write/ConnectDb instead):
//   - Login auth (GetUserByUsername) — fresh signups must work immediately
//   - Read-after-write flows (just inserted X, need to confirm it's there)
//   - Transactions (replicas can't do real transactions across writes)
//
// DO NOT call db.Close() on the returned pool.
func ConnectReadDb(dbname string) (*sql.DB, error) {
	if Read == nil {
		return nil, fmt.Errorf("database not initialised; call sqlconnect.InitDB at startup")
	}
	return Read, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
