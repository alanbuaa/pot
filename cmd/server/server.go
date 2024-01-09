package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/zzz136454872/upgradeable-consensus/logging"
	upgradeable_consensus "github.com/zzz136454872/upgradeable-consensus/node"
)

var (
	logger  = logging.GetLogger()
	sigChan = make(chan os.Signal)
)

func main() {
	// signals to stop nodes
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM,
		syscall.SIGQUIT)

	// node list
	nodeNum := int64(1)
	nodes := make([]*upgradeable_consensus.Node, nodeNum)
	// create nodes
	for i := int64(0); i < nodeNum; i++ {
		go func(index int64) {
			nodes[index] = upgradeable_consensus.NewNode(index)
		}(i)
	}

	<-sigChan
	logger.Info("[UpgradeableConsensus] Exit...")
	for i := int64(0); i < nodeNum; i++ {
		nodes[i].Stop()
	}
}
