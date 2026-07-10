package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/example/game/internal/config"
)

type migrationEntry struct {
	Version uint
	Name    string
}

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	dbURL := os.Getenv("MIGRATE_DATABASE_URL")
	if dbURL == "" {
		dbURL = cfg.DatabaseURL
	}
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	migrationsDir, err := findMigrationsDir()
	if err != nil {
		log.Fatal(err)
	}
	available, err := listMigrations(migrationsDir)
	if err != nil {
		log.Fatal(err)
	}

	m, err := migrate.New(toFileURL(migrationsDir), dbURL)
	if err != nil {
		log.Fatalf("migrate init: %v", err)
	}
	defer func() { _, _ = m.Close() }()

	fmt.Printf("migrate database: %s\n", dbURL)

	switch args[0] {
	case "status", "version":
		if err := runStatus(m, available); err != nil {
			log.Fatal(err)
		}
	case "up":
		if err := runUp(m, available); err != nil {
			log.Fatal(err)
		}
	case "down":
		n := 1
		if len(args) >= 2 {
			n, err = strconv.Atoi(args[1])
			if err != nil || n < 1 {
				log.Fatalf("invalid down steps %q", args[1])
			}
		}
		if err := runDown(m, n); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("unknown command %q", args[0])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`usage: migrate <command> [args]

commands:
  status          查看当前版本、是否有脏状态、与最新版本差距
  up              升级到最新版本
  down [N]        回滚 N 步（默认 1）`)
}

func runStatus(m *migrate.Migrate, available []migrationEntry) error {
	current, dirty, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return fmt.Errorf("read version: %w", err)
	}

	latest := latestVersion(available)
	fmt.Println()
	if errors.Is(err, migrate.ErrNilVersion) {
		fmt.Println("current version: (none)")
	} else {
		fmt.Printf("current version: %d (%s)\n", current, migrationName(available, current))
	}
	fmt.Printf("latest version:  %d (%s)\n", latest, migrationName(available, latest))
	fmt.Printf("dirty:           %v\n", dirty)

	pending := pendingCount(available, current, err)
	switch {
	case dirty:
		fmt.Println("status:          dirty — 需人工修复后再迁移")
	case errors.Is(err, migrate.ErrNilVersion):
		fmt.Printf("status:          pending %d migration(s)\n", len(available))
	case pending > 0:
		fmt.Printf("status:          pending %d migration(s)\n", pending)
	case pending == 0:
		fmt.Println("status:          up to date")
	}

	if len(available) > 0 {
		fmt.Println("\napplied migrations:")
		for _, entry := range available {
			mark := " "
			if !errors.Is(err, migrate.ErrNilVersion) && entry.Version <= current {
				mark = "*"
			}
			fmt.Printf("  %s %03d  %s\n", mark, entry.Version, entry.Name)
		}
	}
	return nil
}

func runUp(m *migrate.Migrate, available []migrationEntry) error {
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			fmt.Println("no change")
			return nil
		}
		return fmt.Errorf("migrate up: %w", err)
	}
	version, dirty, err := m.Version()
	if err != nil {
		return fmt.Errorf("read version after up: %w", err)
	}
	fmt.Printf("upgraded to version %d (%s), dirty=%v\n", version, migrationName(available, version), dirty)
	return nil
}

func runDown(m *migrate.Migrate, steps int) error {
	if err := m.Steps(-steps); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			fmt.Println("no change")
			return nil
		}
		return fmt.Errorf("migrate down: %w", err)
	}
	version, dirty, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return fmt.Errorf("read version after down: %w", err)
	}
	if errors.Is(err, migrate.ErrNilVersion) {
		fmt.Printf("rolled back %d step(s), now at version (none)\n", steps)
		return nil
	}
	fmt.Printf("rolled back %d step(s), now at version %d, dirty=%v\n", steps, version, dirty)
	return nil
}

func listMigrations(dir string) ([]migrationEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	seen := make(map[uint]string)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".up.sql") {
			continue
		}
		base := strings.TrimSuffix(e.Name(), ".up.sql")
		parts := strings.SplitN(base, "_", 2)
		if len(parts) == 0 {
			continue
		}
		v, err := strconv.ParseUint(parts[0], 10, 32)
		if err != nil {
			continue
		}
		name := ""
		if len(parts) == 2 {
			name = parts[1]
		}
		seen[uint(v)] = name
	}
	out := make([]migrationEntry, 0, len(seen))
	for v, name := range seen {
		out = append(out, migrationEntry{Version: v, Name: name})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out, nil
}

func latestVersion(available []migrationEntry) uint {
	if len(available) == 0 {
		return 0
	}
	return available[len(available)-1].Version
}

func migrationName(available []migrationEntry, version uint) string {
	for _, e := range available {
		if e.Version == version {
			if e.Name != "" {
				return e.Name
			}
			return "-"
		}
	}
	return "-"
}

func pendingCount(available []migrationEntry, current uint, versionErr error) int {
	if errors.Is(versionErr, migrate.ErrNilVersion) {
		return len(available)
	}
	pending := 0
	for _, e := range available {
		if e.Version > current {
			pending++
		}
	}
	return pending
}

func findMigrationsDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			migrations := filepath.Join(dir, "migrations")
			if _, err := os.Stat(migrations); err != nil {
				return "", fmt.Errorf("migrations directory not found: %s", migrations)
			}
			return migrations, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("go.mod not found (run from repository root)")
		}
		dir = parent
	}
}

func toFileURL(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "file://" + filepath.ToSlash(dir)
	}
	return "file://" + filepath.ToSlash(abs)
}
