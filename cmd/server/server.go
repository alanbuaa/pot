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
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM,
		syscall.SIGQUIT)
	total := 4
	nodes := make([]*upgradeable_consensus.Node, total)

	for i := int64(0); i < int64(total); i++ {
		go func(index int64) {
			nodes[index] = upgradeable_consensus.NewNode(index)
		}(i)
	}

	<-sigChan
	logger.Info("[UpgradeableConsensus] Exit...")
	for i := 0; i < total; i++ {
		nodes[i].Stop()
	}
}
