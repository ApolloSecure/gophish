package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"bitbucket.org/liamstask/goose/lib/goose"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

var tableOrder = []string{
	"roles",
	"permissions",
	"users",
	"role_permissions",
	"templates",
	"pages",
	"smtp",
	"headers",
	"webhooks",
	"targets",
	"groups",
	"group_targets",
	"campaigns",
	"results",
	"events",
	"mail_logs",
	"attachments",
	"email_requests",
	"imap",
	"goose_db_version",
}

var sequenceTables = []string{
	"attachments",
	"campaigns",
	"email_requests",
	"events",
	"goose_db_version",
	"groups",
	"headers",
	"mail_logs",
	"pages",
	"permissions",
	"results",
	"roles",
	"smtp",
	"targets",
	"templates",
	"users",
	"webhooks",
}

type sourceColumn struct {
	Name     string
	Type     string
	Nullable bool
}

func main() {
	mysqlDSN := flag.String("mysql-dsn", "", "MySQL DSN, for example root:root@tcp(127.0.0.1:3307)/gophish?charset=utf8mb4")
	postgresDSN := flag.String("postgres-dsn", "", "Postgres DSN, for example host=127.0.0.1 port=5432 user=postgres password=postgres dbname=gophish_postgres_clone sslmode=disable")
	migrationsDir := flag.String("migrations-dir", "db/db_postgres/migrations", "Path to the Postgres migrations directory")
	initTarget := flag.Bool("init-target", true, "Run the Postgres migration set before copying data")
	truncateTarget := flag.Bool("truncate-target", true, "Truncate target tables before importing data")
	flag.Parse()

	if *mysqlDSN == "" || *postgresDSN == "" {
		flag.Usage()
		os.Exit(2)
	}

	mysqlDB, err := sql.Open("mysql", *mysqlDSN)
	if err != nil {
		log.Fatalf("open mysql connection: %v", err)
	}
	defer mysqlDB.Close()

	postgresDB, err := sql.Open("postgres", *postgresDSN)
	if err != nil {
		log.Fatalf("open postgres connection: %v", err)
	}
	defer postgresDB.Close()

	if err := mysqlDB.Ping(); err != nil {
		log.Fatalf("ping mysql: %v", err)
	}
	if err := postgresDB.Ping(); err != nil {
		log.Fatalf("ping postgres: %v", err)
	}

	if *initTarget {
		if err := runPostgresMigrations(postgresDB, *postgresDSN, *migrationsDir); err != nil {
			log.Fatalf("initialize postgres schema: %v", err)
		}
	}

	if *truncateTarget {
		if err := truncatePostgresTables(postgresDB); err != nil {
			log.Fatalf("truncate postgres tables: %v", err)
		}
	}

	for _, table := range tableOrder {
		count, err := migrateTable(mysqlDB, postgresDB, table)
		if err != nil {
			log.Fatalf("migrate table %s: %v", table, err)
		}
		log.Printf("migrated %s rows=%d", table, count)
	}

	if err := reseedPostgresSequences(postgresDB); err != nil {
		log.Fatalf("reseed postgres sequences: %v", err)
	}

	if err := compareCounts(mysqlDB, postgresDB); err != nil {
		log.Fatalf("compare row counts: %v", err)
	}

	log.Printf("mysql to postgres migration complete")
}

func runPostgresMigrations(db *sql.DB, dsn, migrationsDir string) error {
	absMigrationsDir, err := filepath.Abs(migrationsDir)
	if err != nil {
		return err
	}
	conf := &goose.DBConf{
		MigrationsDir: absMigrationsDir,
		Env:           "production",
		Driver: goose.DBDriver{
			Name:    "postgres",
			OpenStr: dsn,
			Import:  "github.com/lib/pq",
			Dialect: &goose.PostgresDialect{},
		},
	}
	latest, err := goose.GetMostRecentDBVersion(absMigrationsDir)
	if err != nil {
		return err
	}
	return goose.RunMigrationsOnDb(conf, absMigrationsDir, latest, db)
}

func truncatePostgresTables(db *sql.DB) error {
	for i := len(tableOrder) - 1; i >= 0; i-- {
		table := tableOrder[i]
		query := fmt.Sprintf(`TRUNCATE TABLE "%s" RESTART IDENTITY CASCADE`, table)
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("truncate %s: %w", table, err)
		}
	}
	return nil
}

func migrateTable(mysqlDB, postgresDB *sql.DB, table string) (int64, error) {
	columns, err := loadSourceColumns(mysqlDB, table)
	if err != nil {
		return 0, err
	}

	selectColumns := make([]string, 0, len(columns))
	insertColumns := make([]string, 0, len(columns))
	placeholders := make([]string, 0, len(columns))
	for i, column := range columns {
		selectColumns = append(selectColumns, fmt.Sprintf("`%s`", column.Name))
		insertColumns = append(insertColumns, fmt.Sprintf(`"%s"`, column.Name))
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
	}

	query := fmt.Sprintf("SELECT %s FROM `%s`", strings.Join(selectColumns, ", "), table)
	rows, err := mysqlDB.Query(query)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	tx, err := postgresDB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	insertSQL := fmt.Sprintf(
		`INSERT INTO "%s" (%s) VALUES (%s)`,
		table,
		strings.Join(insertColumns, ", "),
		strings.Join(placeholders, ", "),
	)
	stmt, err := tx.Prepare(insertSQL)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	var migrated int64
	for rows.Next() {
		dest := make([]interface{}, len(columns))
		rawValues := make([]sql.RawBytes, len(columns))
		for i := range rawValues {
			dest[i] = &rawValues[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return 0, err
		}

		values := make([]interface{}, len(columns))
		for i, raw := range rawValues {
			value, err := convertValue(columns[i].Type, raw)
			if err != nil {
				return 0, fmt.Errorf("%s.%s: %w", table, columns[i].Name, err)
			}
			values[i] = value
		}

		if _, err := stmt.Exec(values...); err != nil {
			return 0, fmt.Errorf("insert into %s: %w", table, err)
		}
		migrated++
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return migrated, nil
}

func loadSourceColumns(mysqlDB *sql.DB, table string) ([]sourceColumn, error) {
	rows, err := mysqlDB.Query(`
		SELECT column_name, column_type, is_nullable
		FROM information_schema.columns
		WHERE table_schema = DATABASE() AND table_name = ?
		ORDER BY ordinal_position
	`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []sourceColumn
	for rows.Next() {
		var column sourceColumn
		var nullable string
		if err := rows.Scan(&column.Name, &column.Type, &nullable); err != nil {
			return nil, err
		}
		column.Nullable = nullable == "YES"
		columns = append(columns, column)
	}
	return columns, rows.Err()
}

func convertValue(columnType string, raw sql.RawBytes) (interface{}, error) {
	if raw == nil {
		return nil, nil
	}

	value := string(raw)
	lowerType := strings.ToLower(columnType)

	switch {
	case strings.HasPrefix(lowerType, "tinyint(1)"):
		return value == "1", nil
	case strings.HasPrefix(lowerType, "bigint unsigned"):
		parsed, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return nil, err
		}
		return int64(parsed), nil
	case strings.HasPrefix(lowerType, "bigint"),
		strings.HasPrefix(lowerType, "int"),
		strings.HasPrefix(lowerType, "smallint"):
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, err
		}
		return parsed, nil
	case strings.HasPrefix(lowerType, "double"),
		strings.HasPrefix(lowerType, "float"),
		strings.HasPrefix(lowerType, "real"),
		strings.HasPrefix(lowerType, "decimal"):
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, err
		}
		return parsed, nil
	case strings.HasPrefix(lowerType, "datetime"),
		strings.HasPrefix(lowerType, "timestamp"):
		return parseMySQLTime(value)
	default:
		return value, nil
	}
}

func parseMySQLTime(value string) (interface{}, error) {
	if value == "" || strings.HasPrefix(value, "0000-00-00") {
		return nil, nil
	}
	layouts := []string{
		"2006-01-02 15:04:05.999999",
		"2006-01-02 15:04:05",
		time.RFC3339Nano,
		time.RFC3339,
	}
	for _, layout := range layouts {
		if parsed, err := time.ParseInLocation(layout, value, time.UTC); err == nil {
			return parsed.UTC(), nil
		}
	}
	return nil, fmt.Errorf("unsupported time value %q", value)
}

func reseedPostgresSequences(db *sql.DB) error {
	for _, table := range sequenceTables {
		var sequence sql.NullString
		err := db.QueryRow(`SELECT pg_get_serial_sequence($1, $2)`, fmt.Sprintf(`"%s"`, table), "id").Scan(&sequence)
		if err != nil {
			return fmt.Errorf("lookup sequence for %s: %w", table, err)
		}
		if !sequence.Valid {
			continue
		}
		var maxID sql.NullInt64
		query := fmt.Sprintf(`SELECT MAX("id") FROM "%s"`, table)
		if err := db.QueryRow(query).Scan(&maxID); err != nil {
			return fmt.Errorf("max id for %s: %w", table, err)
		}
		if maxID.Valid {
			if _, err := db.Exec(`SELECT setval($1, $2, true)`, sequence.String, maxID.Int64); err != nil {
				return fmt.Errorf("setval for %s: %w", table, err)
			}
			continue
		}
		if _, err := db.Exec(`SELECT setval($1, 1, false)`, sequence.String); err != nil {
			return fmt.Errorf("reset empty sequence for %s: %w", table, err)
		}
	}
	return nil
}

func compareCounts(mysqlDB, postgresDB *sql.DB) error {
	for _, table := range tableOrder {
		var mysqlCount int64
		var postgresCount int64
		mysqlQuery := fmt.Sprintf("SELECT COUNT(*) FROM `%s`", table)
		if err := mysqlDB.QueryRow(mysqlQuery).Scan(&mysqlCount); err != nil {
			return fmt.Errorf("mysql count %s: %w", table, err)
		}
		postgresQuery := fmt.Sprintf(`SELECT COUNT(*) FROM "%s"`, table)
		if err := postgresDB.QueryRow(postgresQuery).Scan(&postgresCount); err != nil {
			return fmt.Errorf("postgres count %s: %w", table, err)
		}
		if mysqlCount != postgresCount {
			return fmt.Errorf("row count mismatch for %s: mysql=%d postgres=%d", table, mysqlCount, postgresCount)
		}
		log.Printf("verified %s row-count mysql=%d postgres=%d", table, mysqlCount, postgresCount)
	}
	return nil
}
