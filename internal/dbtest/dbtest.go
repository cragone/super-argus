// Package dbtest provides a migrated test database: SQLite in a temp
// directory by default, or a real Postgres (truncated between tests) when
// ARGUS_TEST_POSTGRES_DSN is set, so the same test code exercises both
// dialects without duplication.
package dbtest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cragone/Super-Argus/internal/db"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// tablesInDeleteOrder lists every table a migration creates, children before
// parents, so truncateAll can clear them without relying on cascade config.
var tablesInDeleteOrder = []string{
	"alert_events", "alert_states", "alert_rules", "notification_channels",
	"panels", "dashboards", "saved_queries",
	"chunks", "stream_labels", "streams",
	"api_keys", "users", "settings",
}

// OpenForTest returns a migrated database for the duration of the test.
func OpenForTest(t *testing.T) *gorm.DB {
	t.Helper()
	if dsn := os.Getenv("ARGUS_TEST_POSTGRES_DSN"); dsn != "" {
		return openPostgres(t, dsn)
	}
	g, err := db.Open("sqlite:" + filepath.Join(t.TempDir(), "t.db"))
	require.NoError(t, err)
	sqlDB, err := g.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	require.NoError(t, db.Migrate(g))
	return g
}

// openPostgres migrates (idempotent across the whole test run; goose no-ops
// once a migration is applied) and truncates every table so the test starts
// from a clean slate on a Postgres instance shared across the test binary.
func openPostgres(t *testing.T, dsn string) *gorm.DB {
	t.Helper()
	g, err := db.Open(dsn)
	require.NoError(t, err)
	require.NoError(t, db.Migrate(g))
	truncateAll(t, g)
	t.Cleanup(func() { truncateAll(t, g) })
	return g
}

func truncateAll(t *testing.T, g *gorm.DB) {
	t.Helper()
	for _, tbl := range tablesInDeleteOrder {
		require.NoError(t, g.Exec("DELETE FROM "+tbl).Error, tbl) //nolint:gosec // table names are a fixed internal list, not user input
	}
}
