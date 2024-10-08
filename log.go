package lnd

import (
	"github.com/btcsuite/btcd/connmgr"
	"github.com/btcsuite/btclog"
	"github.com/lightninglabs/neutrino"
	sphinx "github.com/lightningnetwork/lightning-onion"
	"github.com/sat20-labs/lnd/autopilot"
	"github.com/sat20-labs/lnd/build"
	"github.com/sat20-labs/lnd/chainntnfs"
	"github.com/sat20-labs/lnd/chainreg"
	"github.com/sat20-labs/lnd/chanacceptor"
	"github.com/sat20-labs/lnd/chanbackup"
	"github.com/sat20-labs/lnd/chanfitness"
	"github.com/sat20-labs/lnd/channeldb"
	"github.com/sat20-labs/lnd/channelnotifier"
	"github.com/sat20-labs/lnd/cluster"
	"github.com/sat20-labs/lnd/contractcourt"
	"github.com/sat20-labs/lnd/discovery"
	"github.com/sat20-labs/lnd/funding"
	"github.com/sat20-labs/lnd/graph"
	"github.com/sat20-labs/lnd/healthcheck"
	"github.com/sat20-labs/lnd/htlcswitch"
	"github.com/sat20-labs/lnd/invoices"
	"github.com/sat20-labs/lnd/lncfg"
	"github.com/sat20-labs/lnd/lnrpc/autopilotrpc"
	"github.com/sat20-labs/lnd/lnrpc/chainrpc"
	"github.com/sat20-labs/lnd/lnrpc/devrpc"
	"github.com/sat20-labs/lnd/lnrpc/invoicesrpc"
	"github.com/sat20-labs/lnd/lnrpc/neutrinorpc"
	"github.com/sat20-labs/lnd/lnrpc/peersrpc"
	"github.com/sat20-labs/lnd/lnrpc/routerrpc"
	"github.com/sat20-labs/lnd/lnrpc/signrpc"
	"github.com/sat20-labs/lnd/lnrpc/verrpc"
	"github.com/sat20-labs/lnd/lnrpc/walletrpc"
	"github.com/sat20-labs/lnd/lnwallet"
	"github.com/sat20-labs/lnd/lnwallet/btcwallet"
	"github.com/sat20-labs/lnd/lnwallet/chancloser"
	"github.com/sat20-labs/lnd/lnwallet/chanfunding"
	"github.com/sat20-labs/lnd/lnwallet/rpcwallet"
	"github.com/sat20-labs/lnd/monitoring"
	"github.com/sat20-labs/lnd/netann"
	"github.com/sat20-labs/lnd/peer"
	"github.com/sat20-labs/lnd/peernotifier"
	"github.com/sat20-labs/lnd/routing"
	"github.com/sat20-labs/lnd/routing/blindedpath"
	"github.com/sat20-labs/lnd/rpcperms"
	"github.com/sat20-labs/lnd/signal"
	"github.com/sat20-labs/lnd/sweep"
	"github.com/sat20-labs/lnd/tor"
	"github.com/sat20-labs/lnd/watchtower"
	"github.com/sat20-labs/lnd/watchtower/wtclient"
)

// replaceableLogger is a thin wrapper around a logger that is used so the
// logger can be replaced easily without some black pointer magic.
type replaceableLogger struct {
	btclog.Logger
	subsystem string
}

// Loggers can not be used before the log rotator has been initialized with a
// log file. This must be performed early during application startup by
// calling InitLogRotator() on the main log writer instance in the config.
var (
	// lndPkgLoggers is a list of all lnd package level loggers that are
	// registered. They are tracked here so they can be replaced once the
	// SetupLoggers function is called with the final root logger.
	lndPkgLoggers []*replaceableLogger

	// addLndPkgLogger is a helper function that creates a new replaceable
	// main lnd package level logger and adds it to the list of loggers that
	// are replaced again later, once the final root logger is ready.
	addLndPkgLogger = func(subsystem string) *replaceableLogger {
		l := &replaceableLogger{
			Logger:    build.NewSubLogger(subsystem, nil),
			subsystem: subsystem,
		}
		lndPkgLoggers = append(lndPkgLoggers, l)
		return l
	}

	// Loggers that need to be accessible from the lnd package can be placed
	// here. Loggers that are only used in sub modules can be added directly
	// by using the addSubLogger method. We declare all loggers so we never
	// run into a nil reference if they are used early. But the SetupLoggers
	// function should always be called as soon as possible to finish
	// setting them up properly with a root logger.
	ltndLog = addLndPkgLogger("LTND")
	rpcsLog = addLndPkgLogger("RPCS")
	srvrLog = addLndPkgLogger("SRVR")
	atplLog = addLndPkgLogger("ATPL")
)

// genSubLogger creates a logger for a subsystem. We provide an instance of
// a signal.Interceptor to be able to shutdown in the case of a critical error.
func genSubLogger(root *build.RotatingLogWriter,
	interceptor signal.Interceptor) func(string) btclog.Logger {

	// Create a shutdown function which will request shutdown from our
	// interceptor if it is listening.
	shutdown := func() {
		if !interceptor.Listening() {
			return
		}

		interceptor.RequestShutdown()
	}

	// Return a function which will create a sublogger from our root
	// logger without shutdown fn.
	return func(tag string) btclog.Logger {
		return root.GenSubLogger(tag, shutdown)
	}
}

// SetupLoggers initializes all package-global logger variables.
func SetupLoggers(root *build.RotatingLogWriter, interceptor signal.Interceptor) {
	genLogger := genSubLogger(root, interceptor)

	// Now that we have the proper root logger, we can replace the
	// placeholder lnd package loggers.
	for _, l := range lndPkgLoggers {
		l.Logger = build.NewSubLogger(l.subsystem, genLogger)
		SetSubLogger(root, l.subsystem, l.Logger)
	}

	// Initialize loggers from packages outside of `lnd` first. The
	// packages below will overwrite the names of the loggers they import.
	// For instance, the logger in `neutrino.query` is overwritten by
	// `btcwallet.chain`, which is overwritten by `lnwallet`. To ensure the
	// overwriting works, we need to initialize the loggers here so they
	// can be overwritten later.
	AddSubLogger(root, "BTCN", interceptor, neutrino.UseLogger)
	AddSubLogger(root, "CMGR", interceptor, connmgr.UseLogger)

	// Some of the loggers declared in the main lnd package are also used
	// in sub packages.
	signal.UseLogger(ltndLog)
	autopilot.UseLogger(atplLog)

	AddSubLogger(root, "LNWL", interceptor, lnwallet.UseLogger)
	AddSubLogger(root, "DISC", interceptor, discovery.UseLogger)
	AddSubLogger(root, "NTFN", interceptor, chainntnfs.UseLogger)
	AddSubLogger(root, "CHDB", interceptor, channeldb.UseLogger)
	AddSubLogger(root, "HSWC", interceptor, htlcswitch.UseLogger)
	AddSubLogger(root, "CNCT", interceptor, contractcourt.UseLogger)
	AddSubLogger(root, "UTXN", interceptor, contractcourt.UseNurseryLogger)
	AddSubLogger(root, "BRAR", interceptor, contractcourt.UseBreachLogger)
	AddSubLogger(root, "SPHX", interceptor, sphinx.UseLogger)
	AddSubLogger(root, "SWPR", interceptor, sweep.UseLogger)
	AddSubLogger(root, "SGNR", interceptor, signrpc.UseLogger)
	AddSubLogger(root, "WLKT", interceptor, walletrpc.UseLogger)
	AddSubLogger(root, "ARPC", interceptor, autopilotrpc.UseLogger)
	AddSubLogger(root, "NRPC", interceptor, neutrinorpc.UseLogger)
	AddSubLogger(root, "DRPC", interceptor, devrpc.UseLogger)
	AddSubLogger(root, "INVC", interceptor, invoices.UseLogger)
	AddSubLogger(root, "NANN", interceptor, netann.UseLogger)
	AddSubLogger(root, "WTWR", interceptor, watchtower.UseLogger)
	AddSubLogger(root, "NTFR", interceptor, chainrpc.UseLogger)
	AddSubLogger(root, "IRPC", interceptor, invoicesrpc.UseLogger)
	AddSubLogger(root, "CHNF", interceptor, channelnotifier.UseLogger)
	AddSubLogger(root, "CHBU", interceptor, chanbackup.UseLogger)
	AddSubLogger(root, "PROM", interceptor, monitoring.UseLogger)
	AddSubLogger(root, "WTCL", interceptor, wtclient.UseLogger)
	AddSubLogger(root, "PRNF", interceptor, peernotifier.UseLogger)
	AddSubLogger(root, "CHFD", interceptor, chanfunding.UseLogger)
	AddSubLogger(root, "PEER", interceptor, peer.UseLogger)
	AddSubLogger(root, "CHCL", interceptor, chancloser.UseLogger)

	AddSubLogger(root, routing.Subsystem, interceptor, routing.UseLogger)
	AddSubLogger(root, routerrpc.Subsystem, interceptor, routerrpc.UseLogger)
	AddSubLogger(root, chanfitness.Subsystem, interceptor, chanfitness.UseLogger)
	AddSubLogger(root, verrpc.Subsystem, interceptor, verrpc.UseLogger)
	AddSubLogger(root, healthcheck.Subsystem, interceptor, healthcheck.UseLogger)
	AddSubLogger(root, chainreg.Subsystem, interceptor, chainreg.UseLogger)
	AddSubLogger(root, chanacceptor.Subsystem, interceptor, chanacceptor.UseLogger)
	AddSubLogger(root, funding.Subsystem, interceptor, funding.UseLogger)
	AddSubLogger(root, cluster.Subsystem, interceptor, cluster.UseLogger)
	AddSubLogger(root, rpcperms.Subsystem, interceptor, rpcperms.UseLogger)
	AddSubLogger(root, tor.Subsystem, interceptor, tor.UseLogger)
	AddSubLogger(root, btcwallet.Subsystem, interceptor, btcwallet.UseLogger)
	AddSubLogger(root, rpcwallet.Subsystem, interceptor, rpcwallet.UseLogger)
	AddSubLogger(root, peersrpc.Subsystem, interceptor, peersrpc.UseLogger)
	AddSubLogger(root, graph.Subsystem, interceptor, graph.UseLogger)
	AddSubLogger(root, lncfg.Subsystem, interceptor, lncfg.UseLogger)
	AddSubLogger(
		root, blindedpath.Subsystem, interceptor, blindedpath.UseLogger,
	)
}

// AddSubLogger is a helper method to conveniently create and register the
// logger of one or more sub systems.
func AddSubLogger(root *build.RotatingLogWriter, subsystem string,
	interceptor signal.Interceptor, useLoggers ...func(btclog.Logger)) {

	// genSubLogger will return a callback for creating a logger instance,
	// which we will give to the root logger.
	genLogger := genSubLogger(root, interceptor)

	// Create and register just a single logger to prevent them from
	// overwriting each other internally.
	logger := build.NewSubLogger(subsystem, genLogger)
	SetSubLogger(root, subsystem, logger, useLoggers...)
}

// SetSubLogger is a helper method to conveniently register the logger of a sub
// system.
func SetSubLogger(root *build.RotatingLogWriter, subsystem string,
	logger btclog.Logger, useLoggers ...func(btclog.Logger)) {

	root.RegisterSubLogger(subsystem, logger)
	for _, useLogger := range useLoggers {
		useLogger(logger)
	}
}
