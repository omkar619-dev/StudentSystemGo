// Package dbmigrate runs versioned schema migrations at app startup.
//
// Backed by golang-migrate/migrate v4. Migration files live in this package's
// migrations/ directory and are EMBEDDED into the binary at compile time —
// no need to ship them separately or mount into the container.
//
// Usage (from cmd/api/server.go):
//
//	if err := dbmigrate.Run(dbmigrate.Config{
//	    Host: "mysql_primary", Port: "3306", User: "root",
//	    Password: pwd, DBName: "schooldb",
//	}); err != nil { log.Fatal(err) }
//
// Runs ONLY against the primary. The replica picks up the resulting DDL
// (CREATE TABLE, ALTER TABLE) via binlog replication automatically — do
// NOT call this against the replica (it's read_only=ON, ALTERs would fail).
package dbmigrate

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// migrationsFS embeds every .sql file under migrations/ into the binary.
// The //go:embed directive runs at compile time; the resulting embed.FS
// behaves like a read-only filesystem.
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

// Config holds primary-DB connection parameters.
//
// We open a *separate* connection here (not reusing sqlconnect.Write) because
// migrations need ?multiStatements=true on the DSN — that flag lets a single
// query contain multiple SQL statements separated by `;`, which our v1
// migration relies on (3 CREATE TABLEs in one file).
//
// We don't want multiStatements=true on the main app pool because it slightly
// widens the SQL-injection blast radius if anyone ever builds a query via
// string concat. Migrations only run at startup, with our own SQL — safe.
type Config struct {
	Host, Port, User, Password, DBName string
}

// Run applies all pending up-migrations against the primary DB.
//
// Idempotent — safe to call on every app startup. If the schema is already
// at the latest version, this is a no-op (logs "no change" and returns nil).
//
// On a fresh DB: creates schema_migrations table, runs every up file in order,
// records each version.
//
// On an existing DB: reads schema_migrations to find the current version,
// runs only files newer than that.
//
// On error: returns the error WITHOUT swallowing. Caller should log.Fatal —
// running the app against a half-migrated DB is worse than failing to start.
func Run(c Config) error {
	// ── 1. Build a one-shot connection with multiStatements=true ──
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?multiStatements=true&parseTime=true&tls=skip-verify",
		c.User, c.Password, c.Host, c.Port, c.DBName,
	)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("dbmigrate: open: %w", err)
	}
	defer db.Close() // one-shot — close as soon as Run returns

	// Ping verifies the DSN is correct and the DB is reachable.
	// Without this, sql.Open's lazy nature would defer the failure to the
	// first migrate operation, with a more confusing error.
	if err := db.Ping(); err != nil {
		return fmt.Errorf("dbmigrate: ping: %w", err)
	}

	// ── 2. Wire up the source (embedded migration files) ────────
	// iofs.New takes our embed.FS and the subdirectory containing the .sql
	// files. It hands back a Source that golang-migrate can read from.
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("dbmigrate: source: %w", err)
	}

	// ── 3. Wire up the database driver ──────────────────────────
	// mysql.WithInstance lets us reuse our already-opened *sql.DB instead
	// of having golang-migrate parse the DSN itself. The empty mysql.Config
	// is fine — defaults work for MariaDB.
	drv, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		return fmt.Errorf("dbmigrate: driver: %w", err)
	}

	// ── 4. Build the migrator ───────────────────────────────────
	// "iofs" and "mysql" are the source/driver type names — they're labels
	// matching what we wired above, not magic strings.
	m, err := migrate.NewWithInstance("iofs", src, "mysql", drv)
	if err != nil {
		return fmt.Errorf("dbmigrate: migrator: %w", err)
	}

	// ── 5. Apply pending migrations ─────────────────────────────
	// m.Up() walks from current version to the highest available.
	// If already at the top, it returns ErrNoChange — that's NOT a real
	// error, just a signal that there was nothing to do.
	err = m.Up()
	switch {
	case errors.Is(err, migrate.ErrNoChange):
		log.Println("[migrate] schema up to date — no migrations applied")
	case err != nil:
		return fmt.Errorf("dbmigrate: up: %w", err)
	default:
		log.Println("[migrate] migrations applied successfully")
	}

	// Final state log — useful when debugging "did the migration actually run?"
	v, dirty, verr := m.Version()
	if verr == nil {
		log.Printf("[migrate] schema version=%d dirty=%v", v, dirty)
	}
	return nil
}
