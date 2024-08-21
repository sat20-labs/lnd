//go:build integration || lowscrypt

package btcwallet

import (
	"github.com/tinyverse-web3/btcwallet/snacl"
	"github.com/tinyverse-web3/btcwallet/waddrmgr"
)

func init() {
	// Instruct waddrmgr to use the cranked down scrypt parameters when
	// creating new wallet encryption keys. This will speed up the itests
	// considerably.
	fastScrypt := waddrmgr.FastScryptOptions
	keyGen := func(passphrase *[]byte, config *waddrmgr.ScryptOptions) (
		*snacl.SecretKey, error) {

		return snacl.NewSecretKey(
			passphrase, fastScrypt.N, fastScrypt.R, fastScrypt.P,
		)
	}
	waddrmgr.SetSecretKeyGen(keyGen)
}
