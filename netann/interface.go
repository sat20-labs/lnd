package netann

import (
	"github.com/sat20-labs/lnd/channeldb"
	"github.com/sat20-labs/lnd/channeldb/models"
	"github.com/tinyverse-web3/btcd/wire"
)

// DB abstracts the required database functionality needed by the
// ChanStatusManager.
type DB interface {
	// FetchAllOpenChannels returns a slice of all open channels known to
	// the daemon. This may include private or pending channels.
	FetchAllOpenChannels() ([]*channeldb.OpenChannel, error)
}

// ChannelGraph abstracts the required channel graph queries used by the
// ChanStatusManager.
type ChannelGraph interface {
	// FetchChannelEdgesByOutpoint returns the channel edge info and most
	// recent channel edge policies for a given outpoint.
	FetchChannelEdgesByOutpoint(*wire.OutPoint) (*models.ChannelEdgeInfo,
		*models.ChannelEdgePolicy, *models.ChannelEdgePolicy, error)
}
