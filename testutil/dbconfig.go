package testutil

import (
	"database/sql"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gophish/gophish/config"
	_ "github.com/lib/pq"
)

const (
	defaultSQLiteDriver = "sqlite3"
	postgresDriver      = "postgres"
)

func NewTestConfig(prefix string) (*config.Config, func() error, error) {
	driver := strings.ToLower(strings.TrimSpace(os.Getenv("GOPHISH_TEST_DB")))
	if driver == "" {
		driver = defaultSQLiteDriver
	}
	switch driver {
	case postgresDriver:
		return newPostgresConfig(prefix)
	case defaultSQLiteDriver:
		return newSQLiteConfig()
	default:
		return nil, nil, fmt.Errorf("unsupported GOPHISH_TEST_DB %q", driver)
	}
}

func newSQLiteConfig() (*config.Config, func() error, error) {
	dir, err := os.MkdirTemp("", "gophish-sqlite-*")
	if err != nil {
		return nil, nil, err
	}
	conf := &config.Config{
		DBName:         defaultSQLiteDriver,
		DBPath:         filepath.Join(dir, "gophish.db"),
		MigrationsPath: migrationsPath(defaultSQLiteDriver),
	}
	cleanup := func() error {
		return os.RemoveAll(dir)
	}
	return conf, cleanup, nil
}

func newPostgresConfig(prefix string) (*config.Config, func() error, error) {
	adminDSN := strings.TrimSpace(os.Getenv("GOPHISH_TEST_POSTGRES_ADMIN_DSN"))
	if adminDSN == "" {
		return nil, nil, fmt.Errorf("GOPHISH_TEST_POSTGRES_ADMIN_DSN must be set when GOPHISH_TEST_DB=postgres")
	}

	dbName := fmt.Sprintf("gophish_%s_%d_%d", sanitizeIdentifier(prefix), time.Now().UnixNano(), rand.Intn(100000))
	adminDB, err := sql.Open(postgresDriver, adminDSN)
	if err != nil {
		return nil, nil, err
	}
	defer adminDB.Close()

	if _, err := adminDB.Exec(fmt.Sprintf("CREATE DATABASE %s", pqQuoteIdentifier(dbName))); err != nil {
		return nil, nil, err
	}

	targetDSN, err := withPostgresDBName(adminDSN, dbName)
	if err != nil {
		_, _ = adminDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", pqQuoteIdentifier(dbName)))
		return nil, nil, err
	}

	conf := &config.Config{
		DBName:         postgresDriver,
		DBPath:         targetDSN,
		MigrationsPath: migrationsPath(postgresDriver),
	}
	cleanup := func() error {
		cleanupDB, err := sql.Open(postgresDriver, adminDSN)
		if err != nil {
			return err
		}
		defer cleanupDB.Close()

		if _, err := cleanupDB.Exec(
			`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()`,
			dbName,
		); err != nil {
			return err
		}
		_, err = cleanupDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", pqQuoteIdentifier(dbName)))
		return err
	}
	return conf, cleanup, nil
}

func migrationsPath(dbName string) string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("unable to resolve runtime caller for test migrations path")
	}
	root := filepath.Dir(filepath.Dir(file))
	return filepath.Join(root, "db", "db_"+dbName, "migrations")
}

func withPostgresDBName(dsn string, dbName string) (string, error) {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		u, err := url.Parse(dsn)
		if err != nil {
			return "", err
		}
		u.Path = "/" + dbName
		return u.String(), nil
	}

	fields := strings.Fields(dsn)
	rewritten := make([]string, 0, len(fields)+1)
	replaced := false
	for _, field := range fields {
		if strings.HasPrefix(field, "dbname=") {
			rewritten = append(rewritten, "dbname="+dbName)
			replaced = true
			continue
		}
		rewritten = append(rewritten, field)
	}
	if !replaced {
		rewritten = append(rewritten, "dbname="+dbName)
	}
	return strings.Join(rewritten, " "), nil
}

func pqQuoteIdentifier(v string) string {
	return `"` + strings.ReplaceAll(v, `"`, `""`) + `"`
}

func sanitizeIdentifier(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return "test"
	}
	var b strings.Builder
	for _, r := range v {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return strings.Trim(b.String(), "_")
}
