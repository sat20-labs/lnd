package contractcourt

import (
	"github.com/sat20-labs/lnd/channeldb"
	"github.com/sat20-labs/lnd/channeldb/models"
)

type mockHTLCNotifier struct {
	HtlcNotifier
}

func (m *mockHTLCNotifier) NotifyFinalHtlcEvent(key models.CircuitKey,
	info channeldb.FinalHtlcInfo) { //nolint:whitespace
}
