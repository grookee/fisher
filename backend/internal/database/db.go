package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

func Connect() error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://fisher:fisher@localhost:5432/fisher?sslmode=disable"
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	cfg.MaxConns = 25
	cfg.MinConns = 5

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	Pool, err = pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return fmt.Errorf("new pool: %w", err)
	}

	if err := Pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping: %w", err)
	}

	log.Println("connected to database")
	return nil
}

func Close() {
	if Pool != nil {
		Pool.Close()
	}
}
