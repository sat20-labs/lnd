package postgres

import "github.com/tinyverse-web3/btcwallet/walletdb"

type Fixture interface {
	DB() walletdb.DB
	Dump() (map[string]interface{}, error)
}
