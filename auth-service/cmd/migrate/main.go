package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/example/auth-service/internal/config"
)

func main() {
	dir := flag.String("path", "migrations", "migrations directory")
	down := flag.Bool("down", false, "roll back one migration")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	m, err := migrate.New("file://"+*dir, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("migrate new: %v", err)
	}
	defer m.Close()

	if *down {
		if err := m.Steps(-1); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			log.Fatalf("migrate down: %v", err)
		}
		fmt.Println("migrated down 1")
		return
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("migrate up: %v", err)
	}
	fmt.Println("auth migrations applied")
	_ = os.Stdout
}
