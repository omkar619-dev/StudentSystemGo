package sqlconnect

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// DB is the shared connection pool used by all handlers.
// Initialised once at app startup via InitDB.
var DB *sql.DB

// getEnv returns the value of an env var, or fallback if unset/empty.
// Lets us run locally with sensible defaults and override in Docker / prod.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// InitDB reads DB config from env vars, opens the shared pool, and
// configures sensible limits. Call this once at startup before serving traffic.
func InitDB() error {
	host := getEnv("HOST", "localhost")
	port := getEnv("DB_PORT", "3307")
	user := getEnv("DB_USER", "root")
	pass := getEnv("DB_PASSWORD", "")
	name := getEnv("DB_NAME", "schooldb")

	fmt.Printf("Connecting to database %s@%s:%s/%s...\n", user, host, port, name)

	connectionString := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?allowCleartextPasswords=true&tls=skip-verify&parseTime=true",
		user, pass, host, port, name,
	)

	db, err := sql.Open("mysql", connectionString)
	if err != nil {
		return err
	}

	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)

	if err := db.Ping(); err != nil {
		return err
	}

	DB = db
	fmt.Println("Connected to database successfully")
	return nil
}

// ConnectDb is kept for backward-compatibility with existing call sites.
// It returns the shared pool — DO NOT call db.Close() on the result,
// since that would close the pool for every other request.
func ConnectDb(dbname string) (*sql.DB, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialised; call sqlconnect.InitDB at startup")
	}
	return DB, nil
}
