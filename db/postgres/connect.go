package db

import (
	"database/sql"
	"fmt"
	"os"
	"sync"

	_ "github.com/lib/pq" // the db driver supports database/sql
)

// Connection settings come from the environment, falling back to the
// historical local development defaults.
const (
	defaultHost     = "localhost"
	defaultPort     = "5432"
	defaultUser     = "iequser"
	defaultPassword = "iequser"
	defaultDbname   = "ieqdb"
)

var (
	pool     *sql.DB
	poolErr  error
	poolOnce sync.Once
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getDB returns the shared connection pool, creating it on first use.
// sql.DB is a long-lived pool: it is opened once and shared by all callers,
// never closed per query.
func getDB() (*sql.DB, error) {
	poolOnce.Do(func() {
		psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			envOr("IEQ_DB_HOST", defaultHost),
			envOr("IEQ_DB_PORT", defaultPort),
			envOr("IEQ_DB_USER", defaultUser),
			envOr("IEQ_DB_PASSWORD", defaultPassword),
			envOr("IEQ_DB_NAME", defaultDbname))

		pool, poolErr = sql.Open("postgres", psqlInfo)
		if poolErr != nil {
			poolErr = fmt.Errorf("open database: %w", poolErr)
		}
	})
	return pool, poolErr
}
