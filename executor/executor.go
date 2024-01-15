package executor

import (
	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

type Executor interface {
	// CommitTx(tx types.RawTransaction, comsensusID int64)
	// CommitTx(tx types.RawTransaction, idx int64, block types.ConsensusBlock, proof []byte, comsensusID int64)
	CommitBlock(block types.ConsensusBlock, proof []byte, consensusID int64)
	VerifyTx(tx types.RawTransaction) bool
}

func BuildExecutor(cfg *config.ExecutorConfig, log *logrus.Entry) Executor {
	if cfg.Type == "local" {
		return NewLocalExecutor(cfg, log)
	} else if cfg.Type == "remote" {
		return NewRemoteExecutor(cfg, log)
	}
	log.WithField("type", cfg.Type).Warn("executor type error")
	return nil
}
