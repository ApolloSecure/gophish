package models

import (
	"database/sql"
	"os"
	"strings"
	"testing"

	"bitbucket.org/liamstask/goose/lib/goose"
	"github.com/gophish/gophish/testutil"
)

const (
	preApolloMigration       = int64(20220321133237)
	customFieldsMigration    = int64(20260810000000)
	tenantCampaignsMigration = int64(20260814000000)
)

func TestSQLiteCustomFieldsAndTenantCampaignMigrationsRoundTrip(t *testing.T) {
	if driver := strings.ToLower(strings.TrimSpace(os.Getenv("GOPHISH_TEST_DB"))); driver != "" && driver != "sqlite3" {
		t.Skip("SQLite migration test")
	}

	conf, cleanup, err := testutil.NewTestConfig("legacy_migration_round_trip")
	if err != nil {
		t.Fatalf("create SQLite test database: %v", err)
	}
	defer func() {
		if err := cleanup(); err != nil {
			t.Errorf("clean up SQLite test database: %v", err)
		}
	}()

	migrateConf := &goose.DBConf{
		MigrationsDir: conf.MigrationsPath,
		Env:           "production",
		Driver:        chooseDBDriver(conf.DBName, conf.DBPath),
	}
	legacyRunSQLiteMigrations(t, migrateConf, conf.MigrationsPath, customFieldsMigration)
	legacyExecSQLite(t, conf.DBPath, `INSERT INTO group_targets (group_id, target_id, custom_fields) VALUES (11, 12, '{"department":"QA"}')`)
	legacyExecSQLite(t, conf.DBPath, `INSERT INTO results (campaign_id, status, email, custom_fields) VALUES (21, 'Queued', 'result@example.com', '{"department":"QA"}')`)
	legacyExecSQLite(t, conf.DBPath, `INSERT INTO email_requests (user_id, email, custom_fields) VALUES (1, 'request@example.com', '{"department":"QA"}')`)

	legacyRunSQLiteMigrations(t, migrateConf, conf.MigrationsPath, preApolloMigration)
	for _, table := range []string{"group_targets", "results", "email_requests"} {
		legacyAssertSQLiteColumn(t, conf.DBPath, table, "custom_fields", false)
	}
	legacyAssertSQLiteRowCount(t, conf.DBPath, "group_targets", "group_id = 11 AND target_id = 12", 1)
	legacyAssertSQLiteRowCount(t, conf.DBPath, "results", "campaign_id = 21 AND email = 'result@example.com'", 1)
	legacyAssertSQLiteRowCount(t, conf.DBPath, "email_requests", "email = 'request@example.com'", 1)

	legacyRunSQLiteMigrations(t, migrateConf, conf.MigrationsPath, customFieldsMigration)
	for _, table := range []string{"group_targets", "results", "email_requests"} {
		legacyAssertSQLiteColumn(t, conf.DBPath, table, "custom_fields", true)
	}

	legacyRunSQLiteMigrations(t, migrateConf, conf.MigrationsPath, tenantCampaignsMigration)
	legacyExecSQLite(t, conf.DBPath, `INSERT INTO campaigns (user_id, name, status, tenant_id) VALUES (1, 'Preserved campaign', 'Completed', 'tenant-a')`)
	legacyRunSQLiteMigrations(t, migrateConf, conf.MigrationsPath, customFieldsMigration)
	legacyAssertSQLiteColumn(t, conf.DBPath, "campaigns", "tenant_id", false)
	legacyAssertSQLiteRowCount(t, conf.DBPath, "campaigns", "name = 'Preserved campaign'", 1)

	legacyRunSQLiteMigrations(t, migrateConf, conf.MigrationsPath, tenantCampaignsMigration)
	legacyAssertSQLiteColumn(t, conf.DBPath, "campaigns", "tenant_id", true)
	legacyAssertSQLiteRowCount(t, conf.DBPath, "campaigns", "name = 'Preserved campaign'", 1)
}

func legacyRunSQLiteMigrations(t *testing.T, conf *goose.DBConf, migrationsPath string, target int64) {
	t.Helper()
	if err := goose.RunMigrations(conf, migrationsPath, target); err != nil {
		t.Fatalf("migrate SQLite database to %d: %v", target, err)
	}
}

func legacyExecSQLite(t *testing.T, dbPath string, query string) {
	t.Helper()
	database, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open SQLite database: %v", err)
	}
	defer database.Close()
	if _, err := database.Exec(query); err != nil {
		t.Fatalf("execute SQLite fixture: %v", err)
	}
}

func legacyAssertSQLiteColumn(t *testing.T, dbPath string, table string, column string, want bool) {
	t.Helper()
	database, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open SQLite database: %v", err)
	}
	defer database.Close()

	rows, err := database.Query(`PRAGMA table_info("` + table + `")`)
	if err != nil {
		t.Fatalf("inspect %s columns: %v", table, err)
	}
	defer rows.Close()
	found := false
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, primaryKey int
		var defaultValue interface{}
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &primaryKey); err != nil {
			t.Fatalf("scan %s columns: %v", table, err)
		}
		if name == column {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate %s columns: %v", table, err)
	}
	if found != want {
		t.Fatalf("%s.%s presence = %t, want %t", table, column, found, want)
	}
}

func legacyAssertSQLiteRowCount(t *testing.T, dbPath string, table string, condition string, want int) {
	t.Helper()
	database, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open SQLite database: %v", err)
	}
	defer database.Close()

	var got int
	if err := database.QueryRow("SELECT COUNT(*) FROM " + table + " WHERE " + condition).Scan(&got); err != nil {
		t.Fatalf("count %s rows: %v", table, err)
	}
	if got != want {
		t.Fatalf("%s row count = %d, want %d", table, got, want)
	}
}
