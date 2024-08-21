//go:build kvdb_sqlite && !(windows && (arm || 386)) && !(linux && (ppc64 || mips || mipsle || mips64))

package sqlite

import (
	"testing"
	"time"

	"github.com/sat20-labs/lnd/kvdb/sqlbase"
	"github.com/stretchr/testify/require"
	"github.com/tinyverse-web3/btcwallet/walletdb/walletdbtest"
	"golang.org/x/net/context"
)

// TestInterface performs all interfaces tests for this database driver.
func TestInterface(t *testing.T) {
	// dbType is the database type name for this driver.
	dir := t.TempDir()
	ctx := context.Background()

	sqlbase.Init(0)

	cfg := &Config{
		BusyTimeout: time.Second * 5,
	}

	sqlDB, err := NewSqliteBackend(ctx, cfg, dir, "tmp.db", "table")
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	walletdbtest.TestInterface(t, dbType, ctx, cfg, dir, "tmp.db", "temp")
}
