package consensus

import (
	"fmt"

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

// BuildConsensus creates and initializes a consensus instance based on configuration
func BuildConsensus(
	nid int64,
	cid int64,
	cfg *config.ConsensusConfig,
	exec executor.Executor,
	p2pAdaptor p2p.P2PAdaptor,
	log *logrus.Entry,
) (model.Consensus, error) {
	log = log.WithField("module", "CONSENSUS").WithField("chain_id", nid).WithField("c_id", cid)
	log.WithField("c_type", cfg.Type).Info("Building consensus instance")

	var c model.Consensus = nil

	switch cfg.Type {
	case "hotstuff":
		log.WithField("variant", cfg.HotStuff.Type).Debug("Initializing HotStuff consensus")
		switch cfg.HotStuff.Type {
		case "basic":
			c = basic.NewBasicHotStuff(nid, cid, cfg, exec, p2pAdaptor, log)
			log.Info("Basic HotStuff consensus created")
		case "chained":
			c = chained.NewChainedHotStuff(nid, cid, cfg, exec, p2pAdaptor, log)
			log.Info("Chained HotStuff consensus created")
		case "event-driven":
			c = eventdriven.NewEventDrivenHotStuff(nid, cid, cfg, exec, p2pAdaptor, log)
			log.Info("Event-driven HotStuff consensus created")
		default:
			log.WithField("variant", cfg.HotStuff.Type).Error("Unsupported HotStuff variant")
			return nil, fmt.Errorf("unsupported hotstuff variant: %s", cfg.HotStuff.Type)
		}
	case "upgradeable":
		log.Debug("Initializing upgradeable consensus")
		c = NewUpgradeableConsensus(nid, cid, cfg, exec, p2pAdaptor, log)
		log.Info("Upgradeable consensus created")
	case "whirly":
		log.WithField("variant", cfg.Whirly.Type).Debug("Initializing Whirly consensus")
		switch cfg.Whirly.Type {
		case "basic":
			c = whirly.NewWhirly(nid, cid, cfg, exec, p2pAdaptor, log)
			log.Info("Basic Whirly consensus created")
		case "simple":
			c = simpleWhirly.NewSimpleWhirlyForLocalTest(nid, cid, cfg, exec, p2pAdaptor, log)
			log.Info("Simple Whirly consensus created")
		default:
			log.WithField("variant", cfg.Whirly.Type).Error("Unsupported Whirly variant")
			return nil, fmt.Errorf("unsupported whirly variant: %s", cfg.Whirly.Type)
		}
	case "pot":
		log.Debug("Initializing PoT consensus")
		c = pot.NewPoTEngine(nid, cid, cfg, exec, p2pAdaptor, log)
		log.Info("PoT consensus created")
	case "pow":
		log.Debug("Initializing PoW consensus")
		c = pow.NewPowEngine(nid, cid, cfg, exec, p2pAdaptor, log)
		log.Info("PoW consensus created")
	default:
		log.WithField("type", cfg.Type).Error("Unsupported consensus type")
		return nil, fmt.Errorf("unsupported consensus type: %s", cfg.Type)
	}

	if c == nil {
		log.Error("Consensus initialization failed, returned nil")
		return nil, fmt.Errorf("failed to initialize consensus")
	}

	log.Info("Consensus instance built successfully")
	return c, nil
}
