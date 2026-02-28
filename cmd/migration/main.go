package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/riskibarqy/fantasy-league/internal/platform/logging"
)

var logger = logging.NewJSON(logging.LevelInfo).With(
	"service", "fantasy-league-migrate",
	"env", migrationEnv(),
	"component", "migration",
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	dbURL := strings.TrimSpace(os.Getenv("DB_URL"))
	if dbURL == "" {
		fatal("DB_URL is required")
	}

	dbURL = normalizeDBURL(dbURL)

	migrationsDir, err := resolveMigrationsDir()
	if err != nil {
		fatal("resolve migrations dir failed", "error", err)
	}

	sourceURL := "file://" + filepath.ToSlash(migrationsDir)
	m, err := migrate.New(sourceURL, dbURL)
	if err != nil {
		fatal("create migrator failed", "error", err)
	}
	defer closeMigrator(m)

	cmd := strings.ToLower(strings.TrimSpace(os.Args[1]))
	switch cmd {
	case "up":
		err = m.Up()
		handleMigrationErr(err)
		logger.Info("migrations applied", "source", sourceURL)
	case "down":
		steps, parseErr := parseSteps(os.Args[2:])
		if parseErr != nil {
			fatal("parse down steps failed", "error", parseErr)
		}
		err = m.Steps(-steps)
		handleMigrationErr(err)
		logger.Info("migrations rolled back", "steps", steps)
	case "version":
		version, dirty, versionErr := m.Version()
		if errors.Is(versionErr, migrate.ErrNilVersion) {
			fmt.Println("version: none")
			fmt.Println("dirty: false")
			return
		}
		if versionErr != nil {
			fatal("read migration version failed", "error", versionErr)
		}
		fmt.Printf("version: %d\n", version)
		fmt.Printf("dirty: %t\n", dirty)
	case "force":
		if len(os.Args) < 3 {
			fatal("force requires a version argument")
		}
		version, parseErr := parseVersion(os.Args[2])
		if parseErr != nil {
			fatal("parse force version failed", "error", parseErr)
		}
		if err := m.Force(version); err != nil {
			fatal("force migration version failed", "version", version, "error", err)
		}
		logger.Info("migration version forced", "version", version)
	case "goto", "migrate":
		if len(os.Args) < 3 {
			fatal("goto requires a target version argument")
		}
		target, parseErr := parseTarget(os.Args[2])
		if parseErr != nil {
			fatal("parse goto target failed", "error", parseErr)
		}
		err = m.Migrate(target)
		handleMigrationErr(err)
		logger.Info("migrated to version", "target", target)
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
		logger.Info("no migration changes")
		return
	}
	fatal("migration failed", "error", err)
}

func closeMigrator(m *migrate.Migrate) {
	srcErr, dbErr := m.Close()
	if srcErr != nil {
		logger.Warn("close migration source failed", "error", srcErr)
	}
	if dbErr != nil {
		logger.Warn("close migration db failed", "error", dbErr)
	}
}

func fatal(msg string, args ...any) {
	logger.Error(msg, args...)
	os.Exit(1)
}

func migrationEnv() string {
	env := strings.TrimSpace(os.Getenv("APP_ENV"))
	if env == "" {
		return "dev"
	}
	return env
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
