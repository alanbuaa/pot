package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/zzz136454872/upgradeable-consensus/config"
	chain "github.com/zzz136454872/upgradeable-consensus/internal/node"
	"github.com/zzz136454872/upgradeable-consensus/pkg/logging"
)

var sigChan = make(chan os.Signal, 1)

func main() {
	// signals to stop nodes
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	// parse command-line flags
	cfgPath := flag.String("c", "cmd/upgrade/server/config.yaml", "path to config yaml file")
	flag.Parse()

	if cfgPath == nil || len(*cfgPath) == 0 {
		fmt.Printf("config path is empty. Use -c to specify the config file")
		os.Exit(2)
	}

	cfg, err := config.NewConfig(*cfgPath)
	if err != nil {
		fmt.Printf("read config failed: %v", err)
		os.Exit(1)
	}

	id := cfg.Node.ID

	logging.Setup(*cfgPath)
	logger := logging.GetLogger().WithField("module", "UPGRADE-SVR")

	logger.Info("=================================================")
	logger.Info("  Upgradeable Consensus Server Starting...")
	logger.Info("=================================================")
	logger.WithField("config_path", *cfgPath).Debug("Configuration loaded successfully")

	logger.WithField("id", id).Debug("Creating node...")
	node := chain.NewNode(id, *cfgPath, logger)
	logger.Info("Node created successfully, server is running")
	logger.Info("-------------------------------------------------")
	logger.Info("  Server is ready to receive transactions and upgrade proposals")
	logger.Info("-------------------------------------------------")

	<-sigChan
	logger.Info("Shutdown signal received, stopping node...")
	logger.WithField("id", id).Debug("Stopping node...")
	node.Stop()
	logger.Info("Server shutdown complete")
}
