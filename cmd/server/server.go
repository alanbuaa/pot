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
	cfgPath := flag.String("c", "config/config.yaml", "path to config yaml file")
	flag.Parse()

	if cfgPath == nil || len(*cfgPath) == 0 {
		fmt.Printf("config path is empty. Use -c to specify the config file")
		os.Exit(2)
	}

	cfg, err := config.NewConfig(*cfgPath, 0)
	if err != nil {
		fmt.Printf("read config failed: %v", err)
		os.Exit(1)
	}

	logging.Setup(*cfgPath)
	logger := logging.GetLogger().WithField("module", "POTSVR")

	logger.Info("Server starting...")
	logger.WithField("config_path", *cfgPath).Debug("Configuration loaded successfully")

	// node list
	nodeNum := int64(len(cfg.Nodes))
	if nodeNum <= 0 {
		logger.WithField("config_path", *cfgPath).Error("No nodes defined in configuration")
		fmt.Fprintln(os.Stderr, "no nodes defined in config")
		os.Exit(1)
	}
	logger.WithField("node_count", nodeNum).Info("Initializing nodes...")
	nodes := make([]*chain.Node, nodeNum)
	// Create nodes - each node will have its own dedicated logger instance
	for i := int64(0); i < nodeNum; i++ {
		go func(index int64) {
			logger.WithField("id", index).Debug("Creating node...")
			nodes[index] = chain.NewNode(index, *cfgPath)
		}(i)
	}
	logger.Info("All nodes created successfully, server is running")
	// time.Sleep(20 * time.Second)
	// nodes[nodeNum-1] = upgradeable_consensus.NewNode(nodeNum - 1)

	<-sigChan
	logger.Info("Shutdown signal received, stopping all nodes...")
	for i := int64(0); i < nodeNum; i++ {
		logger.WithField("id", i).Debug("Stopping node...")
		nodes[i].Stop()
	}
	logger.Info("Server shutdown complete")
}
