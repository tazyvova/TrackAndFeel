package db

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	Host string
	Port string
	User string
	Pass string
	Name string
}

func FromEnv() Config {
	return Config{
		Host: getenv("DB_HOST", "localhost"),
		Port: getenv("DB_PORT", "5432"),
		User: getenv("DB_USER", "postgres"),
		Pass: getenv("DB_PASSWORD", "postgres"),
		Name: getenv("DB_NAME", "training"),
	}
}

func Connect(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.User, cfg.Pass, cfg.Host, cfg.Port, cfg.Name)
	pcfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	pcfg.MaxConns = 5
	db, err := pgxpool.NewWithConfig(ctx, pcfg)
	if err != nil {
		return nil, err
	}
	ctxPing, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.Ping(ctxPing); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
