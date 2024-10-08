package neutrino_test

import (
	"testing"

	lnwallettest "github.com/sat20-labs/lnd/lnwallet/test"
)

// TestLightningWallet tests LightningWallet powered by neutrino against our
// suite of interface tests.
func TestLightningWallet(t *testing.T) {
	lnwallettest.TestLightningWallet(t, "neutrino")
}
