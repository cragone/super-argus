package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenAndMigrate(t *testing.T) {
	g, err := Open("sqlite:" + filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := g.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := Migrate(g); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := g.Raw("SELECT count(*) FROM settings").Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
}

// TestOpenAndMigrate_Postgres exercises the postgres branch of Open and the
// "postgres" goose dialect in migrateSQL against a real server, to catch
// dialect surprises the SQLite-only test above can't (CHECK constraints,
// JSON columns, boolean defaults). Skipped unless ARGUS_TEST_POSTGRES_DSN is
// set (CI's postgres service container sets it; locally: docker compose up
// -d postgres && ARGUS_TEST_POSTGRES_DSN=postgres://argus:argus@localhost:5432/argus?sslmode=disable).
func TestOpenAndMigrate_Postgres(t *testing.T) {
	dsn := os.Getenv("ARGUS_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set ARGUS_TEST_POSTGRES_DSN to run")
	}
	g, err := Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err := Migrate(g); err != nil {
		t.Fatal(err)
	}
	// Migrate again: goose must no-op on an already-migrated database
	// (this test's DSN is a shared server, so other tests may run first).
	if err := Migrate(g); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
	var n int
	if err := g.Raw("SELECT count(*) FROM settings").Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
}
