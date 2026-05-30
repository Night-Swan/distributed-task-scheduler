package db

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool is the shared database connection pool.
// A pool manages multiple connections automatically — you don't
// open and close individual connections yourself.
var Pool *pgxpool.Pool

// Connect reads the database URL from an environment variable and
// opens a connection pool. Call this once at startup.
func Connect() error {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://taskflow:taskflow@localhost:5432/taskflow"
	}

	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		return fmt.Errorf("could not connect to database: %w", err)
	}

	// Ping to confirm the connection actually works
	if err := pool.Ping(context.Background()); err != nil {
		return fmt.Errorf("database unreachable: %w", err)
	}

	Pool = pool
	return nil
}