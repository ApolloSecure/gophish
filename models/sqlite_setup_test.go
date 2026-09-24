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
	tenantOwnershipMigration = int64(20260817000000)
	previousTenantMigration  = int64(20260814000000)
)

func TestSQLiteTenantMigrationCanBeRolledBackAndReapplied(t *testing.T) {
	if driver := strings.ToLower(strings.TrimSpace(os.Getenv("GOPHISH_TEST_DB"))); driver != "" && driver != "sqlite3" {
		t.Skip("SQLite migration test")
	}

	conf, cleanup, err := testutil.NewTestConfig("tenant_migration_round_trip")
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
	if err := goose.RunMigrations(migrateConf, conf.MigrationsPath, tenantOwnershipMigration); err != nil {
		t.Fatalf("apply tenant ownership migration: %v", err)
	}

	rawDB, err := sql.Open("sqlite3", conf.DBPath)
	if err != nil {
		t.Fatalf("open SQLite test database: %v", err)
	}
	defer rawDB.Close()
	if _, err := rawDB.Exec(`INSERT INTO groups (user_id, name, tenant_id) VALUES (1, 'Preserved group', NULL)`); err != nil {
		t.Fatalf("insert migrated group: %v", err)
	}
	if _, err := rawDB.Exec(`INSERT INTO targets (first_name, last_name, email, position, tenant_id) VALUES ('Test', 'User', 'preserved@example.com', 'Tester', NULL)`); err != nil {
		t.Fatalf("insert migrated target: %v", err)
	}
	if _, err := rawDB.Exec(`INSERT INTO email_requests (user_id, email, custom_fields, tenant_id) VALUES (1, 'preserved@example.com', '{"department":"QA"}', NULL)`); err != nil {
		t.Fatalf("insert migrated email request: %v", err)
	}
	if err := rawDB.Close(); err != nil {
		t.Fatalf("close SQLite test database before rollback: %v", err)
	}

	if err := goose.RunMigrations(migrateConf, conf.MigrationsPath, previousTenantMigration); err != nil {
		t.Fatalf("roll back tenant ownership migration: %v", err)
	}
	assertSQLiteColumn(t, conf.DBPath, "groups", "tenant_id", false)
	assertSQLiteColumn(t, conf.DBPath, "targets", "tenant_id", false)
	assertSQLiteColumn(t, conf.DBPath, "email_requests", "tenant_id", false)
	assertSQLiteRowCount(t, conf.DBPath, "groups", "name = 'Preserved group'", 1)
	assertSQLiteRowCount(t, conf.DBPath, "targets", "email = 'preserved@example.com'", 1)
	assertSQLiteRowCount(t, conf.DBPath, "email_requests", "custom_fields = '{\"department\":\"QA\"}'", 1)

	if err := goose.RunMigrations(migrateConf, conf.MigrationsPath, tenantOwnershipMigration); err != nil {
		t.Fatalf("reapply tenant ownership migration: %v", err)
	}
	assertSQLiteColumn(t, conf.DBPath, "groups", "tenant_id", true)
	assertSQLiteColumn(t, conf.DBPath, "targets", "tenant_id", true)
	assertSQLiteColumn(t, conf.DBPath, "email_requests", "tenant_id", true)
	assertSQLiteRowCount(t, conf.DBPath, "groups", "name = 'Preserved group'", 1)
}

func assertSQLiteColumn(t *testing.T, dbPath string, table string, column string, want bool) {
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

func assertSQLiteRowCount(t *testing.T, dbPath string, table string, condition string, want int) {
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
