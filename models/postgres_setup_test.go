package models

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gophish/gophish/config"
)

func TestSetupPostgresSmoke(t *testing.T) {
	dsn := os.Getenv("GOPHISH_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set GOPHISH_TEST_POSTGRES_DSN to run the Postgres setup smoke test")
	}

	migrationsPath, err := filepath.Abs("../db/db_postgres/migrations")
	if err != nil {
		t.Fatalf("resolve migrations path: %v", err)
	}

	conf := &config.Config{
		DBName:         "postgres",
		DBPath:         dsn,
		MigrationsPath: migrationsPath,
	}

	if err := Setup(conf); err != nil {
		t.Fatalf("setup postgres: %v", err)
	}
	defer func() {
		_ = Close()
	}()

	admin, err := GetUserByUsername(DefaultAdminUsername)
	if err != nil {
		t.Fatalf("load admin user: %v", err)
	}
	if admin.Username != DefaultAdminUsername {
		t.Fatalf("unexpected admin username: %q", admin.Username)
	}
	if admin.ApiKey == "" {
		t.Fatal("expected admin API key to be bootstrapped")
	}

	role, err := GetRoleBySlug(RoleAdmin)
	if err != nil {
		t.Fatalf("load admin role: %v", err)
	}
	if admin.RoleID != role.ID {
		t.Fatalf("expected admin role id %d, got %d", role.ID, admin.RoleID)
	}
}
