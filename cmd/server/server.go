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
	logger := logging.GetLogger()

	// node list
	nodeNum := int64(len(cfg.Nodes))
	if nodeNum <= 0 {
		logger.Error("no nodes defined in config: ", *cfgPath)
		fmt.Fprintln(os.Stderr, "no nodes defined in config")
		os.Exit(1)
	}
	nodes := make([]*chain.Node, nodeNum)
	// create nodes
	for i := int64(0); i < nodeNum; i++ {
		go func(index int64) {
			nodes[index] = chain.NewNode(index)
		}(i)
	}
	// time.Sleep(20 * time.Second)
	// nodes[nodeNum-1] = upgradeable_consensus.NewNode(nodeNum - 1)

	<-sigChan
	logger.Info("[UpgradeableConsensus] Exit...")
	for i := range nodeNum {
		nodes[i].Stop()
	}
}
