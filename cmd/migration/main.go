package main

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	dbURL := strings.TrimSpace(os.Getenv("DB_URL"))
	if dbURL == "" {
		log.Fatal("DB_URL is required")
	}

	dbURL = normalizeDBURL(dbURL)

	migrationsDir, err := resolveMigrationsDir()
	if err != nil {
		log.Fatalf("resolve migrations dir: %v", err)
	}

	sourceURL := "file://" + filepath.ToSlash(migrationsDir)
	m, err := migrate.New(sourceURL, dbURL)
	if err != nil {
		log.Fatalf("create migrator: %v", err)
	}
	defer closeMigrator(m)

	cmd := strings.ToLower(strings.TrimSpace(os.Args[1]))
	switch cmd {
	case "up":
		err = m.Up()
		handleMigrationErr(err)
		log.Printf("migrations applied (source=%s)", sourceURL)
	case "down":
		steps, parseErr := parseSteps(os.Args[2:])
		if parseErr != nil {
			log.Fatal(parseErr)
		}
		err = m.Steps(-steps)
		handleMigrationErr(err)
		log.Printf("rolled back %d migration(s)", steps)
	case "version":
		version, dirty, versionErr := m.Version()
		if errors.Is(versionErr, migrate.ErrNilVersion) {
			fmt.Println("version: none")
			fmt.Println("dirty: false")
			return
		}
		if versionErr != nil {
			log.Fatalf("read version: %v", versionErr)
		}
		fmt.Printf("version: %d\n", version)
		fmt.Printf("dirty: %t\n", dirty)
	case "force":
		if len(os.Args) < 3 {
			log.Fatal("force requires a version argument")
		}
		version, parseErr := parseVersion(os.Args[2])
		if parseErr != nil {
			log.Fatal(parseErr)
		}
		if err := m.Force(version); err != nil {
			log.Fatalf("force version %d: %v", version, err)
		}
		log.Printf("forced version to %d", version)
	case "goto", "migrate":
		if len(os.Args) < 3 {
			log.Fatal("goto requires a target version argument")
		}
		target, parseErr := parseTarget(os.Args[2])
		if parseErr != nil {
			log.Fatal(parseErr)
		}
		err = m.Migrate(target)
		handleMigrationErr(err)
		log.Printf("migrated to version %d", target)
	default:
		printUsage()
		os.Exit(2)
	}
}

func parseSteps(args []string) (int, error) {
	if len(args) == 0 {
		return 1, nil
	}

	steps, err := strconv.Atoi(strings.TrimSpace(args[0]))
	if err != nil {
		return 0, fmt.Errorf("invalid down steps %q: %w", args[0], err)
	}
	if steps <= 0 {
		return 0, fmt.Errorf("down steps must be > 0")
	}

	return steps, nil
}

func parseVersion(raw string) (int, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid version %q: %w", raw, err)
	}
	if value < 0 {
		return 0, fmt.Errorf("version must be >= 0")
	}
	if value > int64(^uint(0)>>1) {
		return 0, fmt.Errorf("version is too large for this platform")
	}

	return int(value), nil
}

func parseTarget(raw string) (uint, error) {
	value, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid target version %q: %w", raw, err)
	}
	return uint(value), nil
}

func handleMigrationErr(err error) {
	if err == nil {
		return
	}
	if errors.Is(err, migrate.ErrNoChange) {
		log.Printf("no migration changes")
		return
	}
	log.Fatal(err)
}

func closeMigrator(m *migrate.Migrate) {
	srcErr, dbErr := m.Close()
	if srcErr != nil {
		log.Printf("close migration source: %v", srcErr)
	}
	if dbErr != nil {
		log.Printf("close migration db: %v", dbErr)
	}
}

func resolveMigrationsDir() (string, error) {
	candidates := []string{
		strings.TrimSpace(os.Getenv("MIGRATIONS_DIR")),
		strings.TrimSpace(os.Getenv("MIGRATIONS_PATH")),
		"./db/migrations",
		"/app/db/migrations",
	}

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		abs, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		info, err := os.Stat(abs)
		if err != nil || !info.IsDir() {
			continue
		}
		return abs, nil
	}

	return "", fmt.Errorf("migration directory not found (checked MIGRATIONS_DIR, MIGRATIONS_PATH, ./db/migrations, /app/db/migrations)")
}

func normalizeDBURL(raw string) string {
	if !envBool("DB_DISABLE_PREPARED_BINARY_RESULT") {
		return raw
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed == nil {
		return raw
	}

	query := parsed.Query()
	if query.Get("disable_prepared_binary_result") == "" {
		query.Set("disable_prepared_binary_result", "yes")
		parsed.RawQuery = query.Encode()
	}

	return parsed.String()
}

func envBool(key string) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch value {
	case "1", "true", "t", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "usage: %s <up|down|version|force|goto> [args]\n", filepath.Base(os.Args[0]))
	fmt.Fprintln(os.Stderr, "examples:")
	fmt.Fprintf(os.Stderr, "  %s up\n", filepath.Base(os.Args[0]))
	fmt.Fprintf(os.Stderr, "  %s down 1\n", filepath.Base(os.Args[0]))
	fmt.Fprintf(os.Stderr, "  %s version\n", filepath.Base(os.Args[0]))
	fmt.Fprintf(os.Stderr, "  %s force 1771776034\n", filepath.Base(os.Args[0]))
	fmt.Fprintf(os.Stderr, "  %s goto 1771776034\n", filepath.Base(os.Args[0]))
}
