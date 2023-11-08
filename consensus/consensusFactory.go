package consensus

import (
	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/consensus/hotstuff/basic"
	"github.com/zzz136454872/upgradeable-consensus/consensus/hotstuff/chained"
	"github.com/zzz136454872/upgradeable-consensus/consensus/hotstuff/eventdriven"
	"github.com/zzz136454872/upgradeable-consensus/consensus/model"
	"github.com/zzz136454872/upgradeable-consensus/consensus/pot"
	"github.com/zzz136454872/upgradeable-consensus/consensus/pow"
	"github.com/zzz136454872/upgradeable-consensus/consensus/whirly/simpleWhirly"
	"github.com/zzz136454872/upgradeable-consensus/consensus/whirly/whirly"
	"github.com/zzz136454872/upgradeable-consensus/executor"
	"github.com/zzz136454872/upgradeable-consensus/p2p"
)

// a mux method to start consensus
func BuildConsensus(
	nid int64,
	cid int64,
	cfg *config.ConsensusConfig,
	exec executor.Executor,
	p2pAdaptor p2p.P2PAdaptor,
	log *logrus.Entry,
) model.Consensus {
	var c model.Consensus = nil
	switch cfg.Type {
	case "hotstuff":
		switch cfg.HotStuff.Type {
		case "basic":
			c = basic.NewBasicHotStuff(nid, cid, cfg, exec, p2pAdaptor, log)
		case "chained":
			c = chained.NewChainedHotStuff(nid, cid, cfg, exec, p2pAdaptor, log)
		case "event-driven":
			c = eventdriven.NewEventDrivenHotStuff(nid, cid, cfg, exec, p2pAdaptor, log)
		default:
			log.Warnf("init consensus type not supported: %s", cfg.HotStuff.Type)
		}
	case "upgradeable":
		c = NewUpgradeableConsensus(nid, cid, cfg, exec, p2pAdaptor, log)
	case "whirly":
		switch cfg.Whirly.Type {
		case "basic":
			c = whirly.NewWhirly(nid, cid, cfg, exec, p2pAdaptor, log)
		case "simple":
			c = simpleWhirly.NewSimpleWhirly(nid, cid, cfg, exec, p2pAdaptor, log)
		default:
			log.Warnf("whirly type not supported: %s", cfg.Whirly.Type)
		}
	case "pot":
		c = pot.NewEngine(nid, cid, cfg, exec, p2pAdaptor, log)
	case "pow":
		c = pow.NewPowEngine(nid, cid, cfg, exec, p2pAdaptor, log)
	default:
		log.Warnf("init consensus type not supported: %s", cfg.Type)
	}
	return c
}
