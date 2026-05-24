package sqlite

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/obalunenko/getenv"
	_ "modernc.org/sqlite"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

const dbPathEnvKey = "FALLOUT_TRACKER_DB_PATH"

func ResolveDBPath() (string, error) {
	dbPath, err := getenv.Env[string](dbPathEnvKey)
	if err == nil {
		if err := ensureDBPathDir(dbPath); err != nil {
			return "", err
		}
		return dbPath, nil
	}
	if !errors.Is(err, getenv.ErrNotSet) {
		return "", fmt.Errorf("read %s: %w", dbPathEnvKey, err)
	}
	return DefaultDBPath()
}

func DefaultDBPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}

	appDir := filepath.Join(configDir, "fallout-tracker")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return "", fmt.Errorf("create app config dir: %w", err)
	}
	return filepath.Join(appDir, "tracker.db"), nil
}

func ensureDBPathDir(dbPath string) error {
	if strings.HasPrefix(dbPath, "file:") {
		return nil
	}

	dbDir := filepath.Dir(dbPath)
	if dbDir == "." {
		return nil
	}

	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		return fmt.Errorf("create db dir: %w", err)
	}
	return nil
}

func OpenAndMigrate(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", sqliteDSN(dbPath))
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	if _, err := db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set goose dialect: %w", err)
	}
	if err := goose.Up(db, "migrations"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return db, nil
}

func sqliteDSN(dbPath string) string {
	if strings.HasPrefix(dbPath, "file:") {
		separator := "?"
		if strings.Contains(dbPath, "?") {
			separator = "&"
		}
		return dbPath + separator + "_pragma=foreign_keys(1)"
	}

	u := url.URL{
		Scheme: "file",
		Path:   dbPath,
	}
	q := u.Query()
	q.Add("_pragma", "foreign_keys(1)")
	u.RawQuery = q.Encode()
	return u.String()
}
