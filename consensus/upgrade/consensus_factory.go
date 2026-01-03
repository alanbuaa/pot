package upgrade

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/consensus"
	"github.com/zzz136454872/upgradeable-consensus/consensus/model"
	"github.com/zzz136454872/upgradeable-consensus/executor"
	"github.com/zzz136454872/upgradeable-consensus/p2p"
)

// ConsensusFactory 共识工厂
// 负责创建和管理不同类型的共识实例
type ConsensusFactory struct {
	nid        int64
	executor   executor.Executor
	p2pAdaptor p2p.P2PAdaptor
	log        *logrus.Entry

	// 共识实例缓存
	instances map[string]model.Consensus
	mu        sync.RWMutex
}

// NewConsensusFactory 创建共识工厂
func NewConsensusFactory(
	nid int64,
	executor executor.Executor,
	p2pAdaptor p2p.P2PAdaptor,
	log *logrus.Entry,
) *ConsensusFactory {
	if log == nil {
		log = logrus.NewEntry(logrus.New())
	}

	return &ConsensusFactory{
		nid:        nid,
		executor:   executor,
		p2pAdaptor: p2pAdaptor,
		log:        log,
		instances:  make(map[string]model.Consensus),
	}
}

// CreateConsensus 创建共识实例
// 根据提案创建新的共识实例
func (cf *ConsensusFactory) CreateConsensus(
	proposal *UpgradeProposal,
	baseConfig *config.ConsensusConfig,
) (model.Consensus, error) {
	cf.mu.Lock()
	defer cf.mu.Unlock()

	// 检查是否已存在
	key := cf.getInstanceKey(proposal)
	if existing, ok := cf.instances[key]; ok {
		cf.log.WithField("key", key).Debug("Reusing existing consensus instance")
		return existing, nil
	}

	// 创建新配置
	newConfig, err := cf.buildConfig(proposal, baseConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build config: %w", err)
	}

	// 创建共识实例
	cid := int64(proposal.SwitchHeight) // 使用切换高度作为 consensus ID
	newConsensus := consensus.BuildConsensus(
		cf.nid,
		cid,
		newConfig,
		cf.executor,
		cf.p2pAdaptor,
		cf.log,
	)

	if newConsensus == nil {
		return nil, fmt.Errorf("failed to build consensus for type: %s", proposal.TargetConsensus)
	}

	// 缓存实例
	cf.instances[key] = newConsensus

	cf.log.WithFields(logrus.Fields{
		"type": proposal.TargetConsensus,
		"cid":  cid,
		"key":  key,
	}).Info("Created new consensus instance")

	return newConsensus, nil
}

// CreateConsensusFromCDL 从 CDL 创建共识
// 用于自定义共识实现
func (cf *ConsensusFactory) CreateConsensusFromCDL(
	cdl *CDLDescriptor,
	baseConfig *config.ConsensusConfig,
) (model.Consensus, error) {
	if cdl == nil {
		return nil, fmt.Errorf("CDL descriptor is nil")
	}

	// TODO: 实现 CDL 编译和运行时
	// 这需要 Phase 4 的 CDL 引擎支持
	return nil, fmt.Errorf("CDL-based consensus creation not yet implemented")
}

// buildConfig 构建新共识的配置
func (cf *ConsensusFactory) buildConfig(
	proposal *UpgradeProposal,
	baseConfig *config.ConsensusConfig,
) (*config.ConsensusConfig, error) {
	// 复制基础配置
	newConfig := &config.ConsensusConfig{
		ConsensusID: int64(proposal.SwitchHeight),
		Type:        proposal.TargetConsensus,
		Keys:        baseConfig.Keys,
		F:           baseConfig.F,
	}

	// 根据目标共识类型设置特定配置
	switch proposal.TargetConsensus {
	case "hotstuff":
		newConfig.HotStuff = baseConfig.HotStuff
	case "pow":
		newConfig.Pow = baseConfig.Pow
	case "pot":
		newConfig.PoT = baseConfig.PoT
	case "whirly":
		newConfig.Whirly = baseConfig.Whirly
	default:
		// 自定义共识类型
		if proposal.CDLDescriptor != nil {
			// 使用 CDL 中的配置
			return cf.buildConfigFromCDL(proposal.CDLDescriptor, baseConfig)
		}
		return nil, fmt.Errorf("unsupported consensus type: %s", proposal.TargetConsensus)
	}

	// 应用提案中的自定义参数
	if len(proposal.ConsensusParams) > 0 {
		cf.applyCustomParams(newConfig, proposal.ConsensusParams)
	}

	return newConfig, nil
}

// buildConfigFromCDL 从 CDL 构建配置
func (cf *ConsensusFactory) buildConfigFromCDL(
	cdl *CDLDescriptor,
	baseConfig *config.ConsensusConfig,
) (*config.ConsensusConfig, error) {
	// TODO: 实现 CDL 到配置的转换
	// 这需要 Phase 4 的 CDL 引擎支持
	return nil, fmt.Errorf("CDL config building not yet implemented")
}

// applyCustomParams 应用自定义参数
func (cf *ConsensusFactory) applyCustomParams(
	cfg *config.ConsensusConfig,
	params map[string]interface{},
) {
	// 应用通用参数
	if blockTime, ok := params["block_time"]; ok {
		if bt, ok := blockTime.(int); ok {
			cfg.BlockTime = bt
		}
	}

	if maxBlockSize, ok := params["max_block_size"]; ok {
		if mbs, ok := maxBlockSize.(int); ok {
			cfg.MaxBlockSize = mbs
		}
	}

	// 可以根据需要添加更多参数
	cf.log.WithField("params", params).Debug("Applied custom consensus parameters")
}

// getInstanceKey 生成实例缓存键
func (cf *ConsensusFactory) getInstanceKey(proposal *UpgradeProposal) string {
	return fmt.Sprintf("%s-%d", proposal.TargetConsensus, proposal.SwitchHeight)
}

// GetInstance 获取已缓存的实例
func (cf *ConsensusFactory) GetInstance(key string) (model.Consensus, bool) {
	cf.mu.RLock()
	defer cf.mu.RUnlock()
	instance, ok := cf.instances[key]
	return instance, ok
}

// RemoveInstance 移除实例
func (cf *ConsensusFactory) RemoveInstance(key string) {
	cf.mu.Lock()
	defer cf.mu.Unlock()
	delete(cf.instances, key)
	cf.log.WithField("key", key).Debug("Removed consensus instance")
}

// Clear 清空所有实例
func (cf *ConsensusFactory) Clear() {
	cf.mu.Lock()
	defer cf.mu.Unlock()
	cf.instances = make(map[string]model.Consensus)
	cf.log.Info("Cleared all consensus instances")
}

// ValidateProposal 验证提案是否可以创建共识
func (cf *ConsensusFactory) ValidateProposal(proposal *UpgradeProposal) error {
	// 检查目标共识类型
	supportedTypes := map[string]bool{
		"hotstuff": true,
		"pow":      true,
		"pot":      true,
		"whirly":   true,
	}

	if !supportedTypes[proposal.TargetConsensus] && proposal.CDLDescriptor == nil {
		return fmt.Errorf("unsupported consensus type: %s (no CDL provided)", proposal.TargetConsensus)
	}

	// 如果有 CDL，验证 CDL
	if proposal.CDLDescriptor != nil {
		// TODO: 调用 CDL 验证器
		cf.log.Debug("CDL validation not yet implemented")
	}

	return nil
}
