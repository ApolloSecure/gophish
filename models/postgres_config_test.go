package models

import "testing"

func TestApplyPostgresSSLCAKeywordDSN(t *testing.T) {
	dsn := "host=127.0.0.1 port=5432 user=postgres dbname=gophish"

	got := applyPostgresSSLCA(dsn, "/tmp/postgres root.pem")

	expected := "host=127.0.0.1 port=5432 user=postgres dbname=gophish sslrootcert='/tmp/postgres root.pem' sslmode=verify-ca"
	if got != expected {
		t.Fatalf("unexpected postgres keyword DSN. expected %q got %q", expected, got)
	}
}

func TestApplyPostgresSSLCAPreservesExistingSettings(t *testing.T) {
	dsn := "host=127.0.0.1 port=5432 user=postgres dbname=gophish sslmode=disable sslrootcert=/existing.pem"

	got := applyPostgresSSLCA(dsn, "/tmp/new.pem")

	if got != dsn {
		t.Fatalf("expected existing postgres SSL settings to be preserved, got %q", got)
	}
}

func TestApplyPostgresSSLCAURLDSN(t *testing.T) {
	dsn := "postgres://postgres:postgres@127.0.0.1:5432/gophish"

	got := applyPostgresSSLCA(dsn, "/tmp/postgres root.pem")

	expected := "postgres://postgres:postgres@127.0.0.1:5432/gophish?sslmode=verify-ca&sslrootcert=%2Ftmp%2Fpostgres+root.pem"
	if got != expected {
		t.Fatalf("unexpected postgres URL DSN. expected %q got %q", expected, got)
	}
}
