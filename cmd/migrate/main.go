package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/samirwankhede/lewly-pgpyewj/internal/config"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	conn, err := pgxpool.New(context.Background(), cfg.PostgresURL)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer conn.Close()

	// For now, just verify connectivity. We will add golang-migrate CLI usage via scripts.
	var one int
	if err := conn.QueryRow(context.Background(), "SELECT 1").Scan(&one); err != nil {
		log.Fatalf("db ping: %v", err)
	}
	fmt.Fprintln(os.Stdout, "DB connection OK")
}
