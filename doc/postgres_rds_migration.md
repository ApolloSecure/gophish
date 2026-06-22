# PostgreSQL RDS Migration Runbook

This runbook describes the safest way to migrate an existing Gophish MySQL
database into a PostgreSQL RDS instance using the built-in
`migrate-mysql-to-postgres` tool.

This is intended for legitimate internal security-awareness and testing
deployments.

## Scope

This runbook assumes:

- the source database is MySQL-compatible and already contains a working
  Gophish deployment
- the target database is a PostgreSQL RDS database or cluster
- the application will use the PostgreSQL **writer** endpoint only

The PostgreSQL **reader** endpoint must not be used by the live application.
Gophish performs frequent writes for event tracking, campaign state, login
state, and mail processing.

## Prerequisites

Before starting, confirm:

1. You have a validated backup or snapshot of the source MySQL database.
2. You have a validated snapshot or rollback path for the target PostgreSQL
   database.
3. Security groups allow connectivity from the migration host to:
   - the MySQL source on port `3306`
   - the PostgreSQL writer on port `5432`
4. You have a PostgreSQL database created on the writer endpoint for Gophish.
5. You have the RDS CA bundle available locally if SSL verification is
   required.

## Endpoint Rules

Use:

- MySQL source: the primary/source instance you want to copy from
- PostgreSQL target: the **writer** endpoint

Do not use:

- PostgreSQL reader endpoint for the migration target
- PostgreSQL reader endpoint in Gophish `db_path`

The reader can still be used later for:

- read-only validation queries
- reporting
- replica-lag observation

## Build the Migrator

From the repo root:

```bash
go build -o ./migrate-mysql-to-postgres ./cmd/migrate-mysql-to-postgres
```

## Connection Strings

### MySQL DSN

Example without TLS:

```text
user:password@tcp(mysql-host:3306)/gophish?charset=utf8mb4&parseTime=true
```

Example with TLS already configured in the DSN:

```text
user:password@tcp(mysql-host:3306)/gophish?charset=utf8mb4&parseTime=true&tls=true
```

If your RDS MySQL instance requires a named TLS profile, configure that at the
client level before running the tool.

### PostgreSQL DSN

Example without certificate verification:

```text
host=writer-endpoint port=5432 user=gophish password=secret dbname=gophish sslmode=require
```

Example with CA verification:

```text
host=writer-endpoint port=5432 user=gophish password=secret dbname=gophish sslmode=verify-full sslrootcert=/path/to/rds-ca.pem
```

## Preflight Checks

Verify connectivity before migrating.

### MySQL

```bash
mysql \
  --host "$MYSQL_HOST" \
  --port "${MYSQL_PORT:-3306}" \
  --user "$MYSQL_USER" \
  --password \
  --database "$MYSQL_DB" \
  -e 'select count(*) as campaigns from campaigns;'
```

### PostgreSQL writer

```bash
PGPASSWORD="$PGPASSWORD" psql \
  -h "$PGHOST" \
  -p "${PGPORT:-5432}" \
  -U "$PGUSER" \
  -d "$PGDATABASE" \
  -c 'select current_database(), current_user, now();'
```

## First Migration Run

Recommended first pass:

1. Run PostgreSQL migrations on the target.
2. Truncate the target tables.
3. Import all data.
4. Reseed PostgreSQL sequences.
5. Compare row counts table-by-table.

Example:

```bash
./migrate-mysql-to-postgres \
  --mysql-dsn "$MYSQL_DSN" \
  --postgres-dsn "$POSTGRES_DSN" \
  --migrations-dir "db/db_postgres/migrations" \
  --init-target=true \
  --truncate-target=true
```

## Expected Output

Success should include:

- PostgreSQL migrations applied or confirmed current
- `migrated <table> rows=<n>` for each table
- `verified <table> row-count mysql=<n> postgres=<n>` for each table
- `mysql to postgres migration complete`

## Post-Migration Validation

After import:

1. Start Gophish against the PostgreSQL writer endpoint.
2. Confirm `models.Setup()` startup succeeds.
3. Confirm the admin login works.
4. Open the dashboard and compare counts with the MySQL-backed deployment.
5. Verify event writes by opening, clicking, and reporting a test campaign.

Useful checks:

```sql
select count(*) from campaigns;
select count(*) from results;
select count(*) from events;
select count(*) from mail_logs;
```

## Gophish Config Example

Point the app at the PostgreSQL writer endpoint only:

```json
{
  "db_name": "postgres",
  "db_path": "host=writer-endpoint port=5432 user=gophish password=secret dbname=gophish sslmode=verify-full",
  "db_sslca_path": "/opt/gophish/rds-ca.pem",
  "migrations_prefix": "db/db_"
}
```

The runtime already supports `db_sslca_path` for PostgreSQL keyword DSNs and
URL DSNs.

## Rollback Plan

If validation fails:

1. Stop the PostgreSQL-backed Gophish instance.
2. Leave the MySQL-backed deployment as the active dev instance.
3. Drop or restore the PostgreSQL target database from snapshot.
4. Fix the data issue and rerun the migration.

Do not attempt partial manual repair first unless you have already identified a
single isolated defect and know exactly how to validate it.

## Operational Notes

- The migration tool preserves primary keys and reseeds sequences after import.
- The migration is designed for a fresh PostgreSQL target, not a live merged
  environment.
- The PostgreSQL reader endpoint may lag behind writes; do not use it for
  correctness-sensitive verification immediately after import.
