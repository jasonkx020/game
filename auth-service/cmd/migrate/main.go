package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/example/auth-service/internal/config"
)

func main() {
	dir := flag.String("path", "migrations", "migrations directory")
	down := flag.Bool("down", false, "roll back one migration")
	force := flag.Int("force", -1, "force schema version and clear dirty (-1 = disabled)")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	dbURL, err := withAuthMigrationsTable(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database url: %v", err)
	}

	m, err := migrate.New("file://"+*dir, dbURL)
	if err != nil {
		log.Fatalf("migrate new: %v", err)
	}
	defer m.Close()

	if *force >= 0 {
		if err := m.Force(*force); err != nil {
			log.Fatalf("force: %v", err)
		}
		fmt.Printf("forced auth_schema_migrations to version %d (dirty=false)\n", *force)
		return
	}

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
	v, dirty, verr := m.Version()
	if verr != nil && !errors.Is(verr, migrate.ErrNilVersion) {
		log.Fatalf("version: %v", verr)
	}
	if errors.Is(verr, migrate.ErrNilVersion) {
		fmt.Println("auth migrations applied (version none)")
	} else {
		fmt.Printf("auth migrations applied (version=%d dirty=%v)\n", v, dirty)
	}
	_ = os.Stdout
}

// Prefer a dedicated migrations table so auth-service can share the Postgres
// instance with GameS without colliding on schema_migrations version numbers.
func withAuthMigrationsTable(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	q := u.Query()
	if q.Get("x-migrations-table") == "" {
		q.Set("x-migrations-table", "auth_schema_migrations")
	}
	u.RawQuery = q.Encode()
	// url.Values.Encode escapes, but postgres DSN accepts it; keep sslmode etc.
	out := u.String()
	// Defend against Force misuse logs.
	if strings.Contains(out, "x-migrations-table") {
		return out, nil
	}
	return raw + sep(raw) + "x-migrations-table=auth_schema_migrations", nil
}

func sep(raw string) string {
	if strings.Contains(raw, "?") {
		return "&"
	}
	return "?"
}
