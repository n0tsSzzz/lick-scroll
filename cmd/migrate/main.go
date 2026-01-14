package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"

	"lick-scroll/pkg/config"

	"github.com/pressly/goose/v3"
	_ "github.com/lib/pq"
)

func main() {
	var (
		dir     = flag.String("dir", "migrations", "directory with migration files")
		command = flag.String("command", "up", "migration command (up, down, status, create)")
		name    = flag.String("name", "", "name for new migration (used with create command)")
	)
	flag.Parse()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Build DSN
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cfg.DBHost,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
		cfg.DBPort,
		cfg.DBSSLMode,
	)

	// Open database connection
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Set goose dialect
	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("Failed to set dialect: %v", err)
	}

	// Execute command
	switch *command {
	case "create":
		if *name == "" {
			log.Fatal("Name is required for create command")
		}
		if err := goose.Create(db, *dir, *name, "sql"); err != nil {
			log.Fatalf("Failed to create migration: %v", err)
		}
		fmt.Printf("Created migration: %s\n", *name)
	case "up":
		if err := goose.Up(db, *dir); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		fmt.Println("Migrations applied successfully")
	case "down":
		if err := goose.Down(db, *dir); err != nil {
			log.Fatalf("Failed to rollback migrations: %v", err)
		}
		fmt.Println("Migrations rolled back successfully")
	case "status":
		if err := goose.Status(db, *dir); err != nil {
			log.Fatalf("Failed to get migration status: %v", err)
		}
	default:
		log.Fatalf("Unknown command: %s", *command)
	}
}
