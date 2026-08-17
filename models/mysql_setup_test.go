package models

import (
	"database/sql"
	"os"
	"strings"
	"testing"

	"bitbucket.org/liamstask/goose/lib/goose"
	"github.com/gophish/gophish/testutil"
)

func TestMySQLTenantMigrationPreservesExistingCampaigns(t *testing.T) {
	if strings.ToLower(strings.TrimSpace(os.Getenv("GOPHISH_TEST_DB"))) != "mysql" {
		t.Skip("MySQL migration test")
	}
	conf, cleanup, err := testutil.NewTestConfig("tenant_migration")
	if err != nil {
		t.Fatalf("create MySQL test database: %v", err)
	}
	defer func() {
		if err := cleanup(); err != nil {
			t.Errorf("clean up MySQL test database: %v", err)
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

	rawDB, err := sql.Open("mysql", conf.DBPath)
	if err != nil {
		t.Fatalf("open MySQL test database: %v", err)
	}
	if _, err := rawDB.Exec("INSERT INTO campaigns (user_id, name, status) VALUES (1, 'Existing campaign', 'Completed')"); err != nil {
		rawDB.Close()
		t.Fatalf("insert existing campaign: %v", err)
	}
	if err := rawDB.Close(); err != nil {
		t.Fatalf("close MySQL test database: %v", err)
	}

	if err := Setup(conf); err != nil {
		t.Fatalf("apply tenant migrations: %v", err)
	}
	defer func() {
		if err := Close(); err != nil {
			t.Errorf("close model database: %v", err)
		}
	}()

	var tenantID *string
	if err := db.Raw("SELECT tenant_id FROM campaigns WHERE name = 'Existing campaign'").Row().Scan(&tenantID); err != nil {
		t.Fatalf("read migrated campaign: %v", err)
	}
	if tenantID != nil {
		t.Fatalf("existing campaign tenant_id = %q, want NULL", *tenantID)
	}
	var indexCount int
	if err := db.Raw("SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'targets' AND index_name = 'targets_tenant_email_idx'").Row().Scan(&indexCount); err != nil {
		t.Fatalf("inspect tenant target index: %v", err)
	}
	if indexCount != 2 {
		t.Fatalf("tenant target index columns = %d, want 2", indexCount)
	}
	var constraintCount int
	if err := db.Raw("SELECT COUNT(*) FROM information_schema.referential_constraints WHERE constraint_schema = DATABASE() AND constraint_name = 'campaigns_tenant_id_fk' AND delete_rule = 'CASCADE'").Row().Scan(&constraintCount); err != nil {
		t.Fatalf("inspect campaign tenant foreign key: %v", err)
	}
	if constraintCount != 1 {
		t.Fatalf("campaign tenant foreign key count = %d, want 1", constraintCount)
	}
}
