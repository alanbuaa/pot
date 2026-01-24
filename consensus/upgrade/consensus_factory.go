package upgrade

import (
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/consensus"
	"github.com/zzz136454872/upgradeable-consensus/consensus/model"
	"github.com/zzz136454872/upgradeable-consensus/consensus/upgrade/cdl"
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
	newConsensus, _ := consensus.BuildConsensus(
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
	cdlDescriptor *CDLDescriptor,
	baseConfig *config.ConsensusConfig,
) (model.Consensus, error) {
	if cdlDescriptor == nil {
		return nil, fmt.Errorf("CDL descriptor is nil")
	}

	cf.log.WithFields(logrus.Fields{
		"name":    cdlDescriptor.Name,
		"version": cdlDescriptor.Version,
		"type":    cdlDescriptor.Type,
	}).Info("Creating consensus from CDL")

	// 将旧的 CDLDescriptor 转换为新的 cdl.CDLDescriptor
	cdlDesc := cf.convertToCDLDescriptor(cdlDescriptor)

	// 验证 CDL
	validator := cdl.NewValidator(cf.log)
	if err := validator.Validate(cdlDesc); err != nil {
		return nil, fmt.Errorf("CDL validation failed: %w", err)
	}

	// 构建配置
	consensusConfig, err := cf.buildConfigFromCDL(cdlDescriptor, baseConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build config from CDL: %w", err)
	}

	// 使用 CDL 编译器创建运行时
	compiler := cdl.NewCompiler(cf.log)
	runtime, err := compiler.Compile(
		cdlDesc,
		consensusConfig.ConsensusID,
		consensusConfig,
		cf.p2pAdaptor,
	)
	if err != nil {
		return nil, fmt.Errorf("CDL compilation failed: %w", err)
	}

	cf.log.WithFields(logrus.Fields{
		"consensus_id": runtime.GetConsensusID(),
		"type":         runtime.GetConsensusType(),
	}).Info("Successfully created consensus from CDL")

	return runtime, nil
}

// convertToCDLDescriptor 将旧的 CDLDescriptor 转换为新的 cdl.CDLDescriptor
func (cf *ConsensusFactory) convertToCDLDescriptor(old *CDLDescriptor) *cdl.CDLDescriptor {
	// 从旧格式转换到新格式
	desc := &cdl.CDLDescriptor{
		Consensus: cdl.ConsensusSpec{
			Name:    old.Name,
			Version: old.Version,
			Type:    old.Type,
		},
	}

	// 转换组件配置
	if components, ok := old.Components.(map[string]interface{}); ok {
		desc.Consensus.Components = cf.convertComponents(components)
	}

	// 转换参数
	if params, ok := old.Parameters.(map[string]interface{}); ok {
		desc.Consensus.Parameters = cf.convertParameters(params)
	}

	// 转换阶段
	if phases, ok := old.Phases.([]interface{}); ok {
		desc.Consensus.Phases = cf.convertPhases(phases)
	}

	// 转换状态机
	if sm, ok := old.StateMachine.(map[string]interface{}); ok {
		desc.Consensus.StateMachine = cf.convertStateMachine(sm)
	}

	// 转换安全属性
	if props, ok := old.SafetyProperties.([]interface{}); ok {
		desc.Consensus.SafetyProperties = cf.convertSafetyProperties(props)
	}

	// 转换性能要求
	if reqs, ok := old.PerformanceRequirements.(map[string]interface{}); ok {
		desc.Consensus.PerformanceRequirements = cf.convertPerformanceRequirements(reqs)
	}

	return desc
}

// convertComponents 转换组件配置
func (cf *ConsensusFactory) convertComponents(data map[string]interface{}) cdl.Components {
	components := cdl.Components{}

	if crypto, ok := data["crypto"].(map[string]interface{}); ok {
		components.Crypto = cdl.CryptoComponent{
			Hash:         getStringValue(crypto, "hash", "SHA256"),
			Signature:    getStringValue(crypto, "signature", "ECDSA"),
			VRF:          getStringValue(crypto, "vrf", ""),
			VDF:          getStringValue(crypto, "vdf", ""),
			ThresholdSig: getStringValue(crypto, "threshold_sig", ""),
		}
	}

	if network, ok := data["network"].(map[string]interface{}); ok {
		components.Network = cdl.NetworkComponent{
			Topology:  getStringValue(network, "topology", "gossip"),
			Broadcast: getStringValue(network, "broadcast", "reliable"),
		}
	}

	if storage, ok := data["storage"].(map[string]interface{}); ok {
		components.Storage = cdl.StorageComponent{
			Blockchain: getStringValue(storage, "blockchain", "merkle-chain"),
			State:      getStringValue(storage, "state", "simple-map"),
		}
	}

	return components
}

// convertParameters 转换参数
func (cf *ConsensusFactory) convertParameters(data map[string]interface{}) cdl.Parameters {
	params := cdl.Parameters{
		BlockTime:    getStringValue(data, "block_time", "1s"),
		MaxBlockSize: getIntValue(data, "max_block_size", 1048576),
		Custom:       make(map[string]interface{}),
	}

	// 保留其他自定义参数
	for k, v := range data {
		if k != "block_time" && k != "max_block_size" {
			params.Custom[k] = v
		}
	}

	return params
}

// convertPhases 转换阶段
func (cf *ConsensusFactory) convertPhases(data []interface{}) []cdl.Phase {
	var phases []cdl.Phase
	for _, item := range data {
		if phaseMap, ok := item.(map[string]interface{}); ok {
			phase := cdl.Phase{
				Name:  getStringValue(phaseMap, "name", ""),
				Entry: getStringValue(phaseMap, "entry", ""),
				Exit:  getStringValue(phaseMap, "exit", ""),
			}

			if actions, ok := phaseMap["actions"].([]interface{}); ok {
				for _, actionItem := range actions {
					if actionMap, ok := actionItem.(map[string]interface{}); ok {
						action := cdl.Action{
							Type: getStringValue(actionMap, "type", ""),
							Name: getStringValue(actionMap, "name", ""),
							Code: getStringValue(actionMap, "code", ""),
						}
						if params, ok := actionMap["parameters"].(map[string]interface{}); ok {
							action.Parameters = params
						}
						phase.Actions = append(phase.Actions, action)
					}
				}
			}

			phases = append(phases, phase)
		}
	}
	return phases
}

// convertStateMachine 转换状态机
func (cf *ConsensusFactory) convertStateMachine(data map[string]interface{}) cdl.StateMachine {
	sm := cdl.StateMachine{}

	if states, ok := data["states"].([]interface{}); ok {
		for _, state := range states {
			if stateStr, ok := state.(string); ok {
				sm.States = append(sm.States, stateStr)
			}
		}
	}

	if transitions, ok := data["transitions"].([]interface{}); ok {
		for _, item := range transitions {
			if transMap, ok := item.(map[string]interface{}); ok {
				transition := cdl.Transition{
					From:      getStringValue(transMap, "from", ""),
					To:        getStringValue(transMap, "to", ""),
					Event:     getStringValue(transMap, "event", ""),
					Condition: getStringValue(transMap, "condition", ""),
					Action:    getStringValue(transMap, "action", ""),
				}
				sm.Transitions = append(sm.Transitions, transition)
			}
		}
	}

	return sm
}

// convertSafetyProperties 转换安全属性
func (cf *ConsensusFactory) convertSafetyProperties(data []interface{}) []cdl.SafetyProperty {
	var props []cdl.SafetyProperty
	for _, item := range data {
		if propMap, ok := item.(map[string]interface{}); ok {
			prop := cdl.SafetyProperty{
				Name:    getStringValue(propMap, "name", ""),
				Formula: getStringValue(propMap, "formula", ""),
			}
			props = append(props, prop)
		}
	}
	return props
}

// convertPerformanceRequirements 转换性能要求
func (cf *ConsensusFactory) convertPerformanceRequirements(data map[string]interface{}) cdl.PerformanceRequirements {
	return cdl.PerformanceRequirements{
		MinThroughput:  getIntValue(data, "min_throughput", 100),
		MaxLatency:     getIntValue(data, "max_latency", 10),
		FaultTolerance: getFloatValue(data, "fault_tolerance", 0.33),
	}
}

// 辅助函数
func getStringValue(m map[string]interface{}, key, defaultVal string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return defaultVal
}

func getIntValue(m map[string]interface{}, key string, defaultVal int) int {
	if val, ok := m[key].(int); ok {
		return val
	}
	if val, ok := m[key].(float64); ok {
		return int(val)
	}
	return defaultVal
}

func getFloatValue(m map[string]interface{}, key string, defaultVal float64) float64 {
	if val, ok := m[key].(float64); ok {
		return val
	}
	if val, ok := m[key].(int); ok {
		return float64(val)
	}
	return defaultVal
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
		Fault:       baseConfig.Fault,
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
	cdlDescriptor *CDLDescriptor,
	baseConfig *config.ConsensusConfig,
) (*config.ConsensusConfig, error) {
	cf.log.Debug("Building config from CDL")

	// 创建新配置
	newConfig := &config.ConsensusConfig{
		ConsensusID: baseConfig.ConsensusID,
		Type:        cdlDescriptor.Type,
		Keys:        baseConfig.Keys,
		Fault:       baseConfig.Fault,
	}

	// 从 CDL 参数中提取配置
	if params, ok := cdlDescriptor.Parameters.(map[string]interface{}); ok {
		// 解析 block_time
		if blockTimeStr, ok := params["block_time"].(string); ok {
			duration, err := time.ParseDuration(blockTimeStr)
			if err == nil {
				newConfig.BlockTime = int(duration.Seconds())
			}
		} else if blockTimeInt, ok := params["block_time"].(int); ok {
			newConfig.BlockTime = blockTimeInt
		} else if blockTimeFloat, ok := params["block_time"].(float64); ok {
			newConfig.BlockTime = int(blockTimeFloat)
		}

		// 解析 max_block_size
		if maxBlockSize, ok := params["max_block_size"].(int); ok {
			newConfig.MaxBlockSize = maxBlockSize
		} else if maxBlockSizeFloat, ok := params["max_block_size"].(float64); ok {
			newConfig.MaxBlockSize = int(maxBlockSizeFloat)
		}

		// 应用其他自定义参数
		for key, value := range params {
			if key != "block_time" && key != "max_block_size" {
				// 可以在这里添加其他配置映射
				cf.log.WithFields(logrus.Fields{
					"key":   key,
					"value": value,
				}).Debug("Custom CDL parameter")
			}
		}
	}

	// 设置默认值（如果 CDL 没有提供）
	if newConfig.BlockTime == 0 {
		newConfig.BlockTime = 1 // 默认 1 秒
	}
	if newConfig.MaxBlockSize == 0 {
		newConfig.MaxBlockSize = 1048576 // 默认 1MB
	}

	cf.log.WithFields(logrus.Fields{
		"consensus_type": newConfig.Type,
		"block_time":     newConfig.BlockTime,
		"max_block_size": newConfig.MaxBlockSize,
	}).Info("Built config from CDL")

	return newConfig, nil
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
		cf.log.WithFields(logrus.Fields{
			"name":    proposal.CDLDescriptor.Name,
			"version": proposal.CDLDescriptor.Version,
			"type":    proposal.CDLDescriptor.Type,
		}).Info("Validating CDL descriptor")

		// 调用 CDL 验证器
		cdlDesc := cf.convertToCDLDescriptor(proposal.CDLDescriptor)
		validator := cdl.NewValidator(cf.log)

		// 执行完整验证
		if err := validator.Validate(cdlDesc); err != nil {
			cf.log.WithError(err).Error("CDL validation failed")
			return fmt.Errorf("CDL validation failed: %w", err)
		}

		// 验证语义（更深层次的验证）
		if err := validator.ValidateSemantics(cdlDesc); err != nil {
			cf.log.WithError(err).Error("CDL semantic validation failed")
			return fmt.Errorf("CDL semantic validation failed: %w", err)
		}

		cf.log.Info("CDL validation passed")
	}

	return nil
}
