package models

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bitbucket.org/liamstask/goose/lib/goose"
	"github.com/gophish/gophish/config"
	"github.com/gophish/gophish/testutil"
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

func TestPostgresTenantMigrationPreservesExistingCampaigns(t *testing.T) {
	if strings.ToLower(strings.TrimSpace(os.Getenv("GOPHISH_TEST_DB"))) != "postgres" {
		t.Skip("PostgreSQL migration test")
	}
	conf, cleanup, err := testutil.NewTestConfig("tenant_migration")
	if err != nil {
		t.Fatalf("create PostgreSQL test database: %v", err)
	}
	defer func() {
		if err := cleanup(); err != nil {
			t.Errorf("clean up PostgreSQL test database: %v", err)
		}
	}()

	migrateConf := &goose.DBConf{
		MigrationsDir: conf.MigrationsPath,
		Env:           "production",
		Driver:        chooseDBDriver(conf.DBName, conf.DBPath),
	}
	const previousMigration = int64(20220321133237)
	if err := goose.RunMigrations(migrateConf, conf.MigrationsPath, previousMigration); err != nil {
		t.Fatalf("apply pre-tenant migrations: %v", err)
	}

	rawDB, err := sql.Open("postgres", conf.DBPath)
	if err != nil {
		t.Fatalf("open PostgreSQL test database: %v", err)
	}
	if _, err := rawDB.Exec(`INSERT INTO campaigns (user_id, name, status) VALUES (1, 'Existing campaign', 'Completed')`); err != nil {
		rawDB.Close()
		t.Fatalf("insert existing campaign: %v", err)
	}
	if err := rawDB.Close(); err != nil {
		t.Fatalf("close PostgreSQL test database: %v", err)
	}

	if err := Setup(conf); err != nil {
		t.Fatalf("apply tenant migration: %v", err)
	}
	defer func() {
		if err := Close(); err != nil {
			t.Errorf("close model database: %v", err)
		}
	}()

	var tenantID *string
	if err := db.Raw(`SELECT tenant_id FROM campaigns WHERE name = 'Existing campaign'`).Row().Scan(&tenantID); err != nil {
		t.Fatalf("read migrated campaign: %v", err)
	}
	if tenantID != nil {
		t.Fatalf("existing campaign tenant_id = %q, want NULL", *tenantID)
	}
	var indexCount int
	if err := db.Raw(`SELECT COUNT(*) FROM pg_indexes WHERE tablename = 'campaigns' AND indexname = 'campaigns_user_tenant_id_idx'`).Row().Scan(&indexCount); err != nil {
		t.Fatalf("inspect tenant index: %v", err)
	}
	if indexCount != 1 {
		t.Fatalf("tenant index count = %d, want 1", indexCount)
	}
}
