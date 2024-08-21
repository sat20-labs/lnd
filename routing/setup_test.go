package routing

import (
	"testing"

	"github.com/sat20-labs/lnd/kvdb"
)

func TestMain(m *testing.M) {
	kvdb.RunTests(m)
}
