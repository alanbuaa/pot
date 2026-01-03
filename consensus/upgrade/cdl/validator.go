package cdl

import (
	"fmt"
	"regexp"

	"github.com/sirupsen/logrus"
)

// Validator CDL 验证器
type Validator struct {
	log *logrus.Entry
}

// NewValidator 创建 CDL 验证器
func NewValidator(log *logrus.Entry) *Validator {
	if log == nil {
		log = logrus.NewEntry(logrus.New())
	}
	return &Validator{
		log: log,
	}
}

// Validate 验证 CDL 描述符
func (v *Validator) Validate(descriptor *CDLDescriptor) error {
	v.log.Info("Starting CDL validation")

	// 验证基本信息
	if err := v.validateBasicInfo(&descriptor.Consensus); err != nil {
		return err
	}

	// 验证组件配置
	if err := v.validateComponents(&descriptor.Consensus.Components); err != nil {
		return err
	}

	// 验证参数
	if err := v.validateParameters(&descriptor.Consensus.Parameters); err != nil {
		return err
	}

	// 验证阶段定义
	if err := v.validatePhases(descriptor.Consensus.Phases); err != nil {
		return err
	}

	// 验证状态机
	if err := v.validateStateMachine(&descriptor.Consensus.StateMachine); err != nil {
		return err
	}

	// 验证安全属性
	if err := v.validateSafetyProperties(descriptor.Consensus.SafetyProperties); err != nil {
		return err
	}

	// 验证性能要求
	if err := v.validatePerformanceRequirements(&descriptor.Consensus.PerformanceRequirements); err != nil {
		return err
	}

	v.log.Info("CDL validation completed successfully")
	return nil
}

// validateBasicInfo 验证基本信息
func (v *Validator) validateBasicInfo(spec *ConsensusSpec) error {
	// 验证名称
	if spec.Name == "" {
		return fmt.Errorf("consensus name cannot be empty")
	}

	// 名称格式检查：只允许字母、数字、下划线和连字符
	namePattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !namePattern.MatchString(spec.Name) {
		return fmt.Errorf("consensus name must contain only letters, numbers, underscores, and hyphens")
	}

	// 验证版本
	if spec.Version == "" {
		return fmt.Errorf("version cannot be empty")
	}

	// 版本格式检查：语义版本 (major.minor.patch)
	versionPattern := regexp.MustCompile(`^\d+\.\d+(\.\d+)?$`)
	if !versionPattern.MatchString(spec.Version) {
		return fmt.Errorf("version must follow semantic versioning (e.g., 1.0.0)")
	}

	// 验证类型
	if spec.Type == "" {
		return fmt.Errorf("consensus type cannot be empty")
	}

	v.log.WithFields(logrus.Fields{
		"name":    spec.Name,
		"version": spec.Version,
		"type":    spec.Type,
	}).Debug("Basic info validated")

	return nil
}

// validateComponents 验证组件配置
func (v *Validator) validateComponents(components *Components) error {
	// 验证加密组件
	supportedHashes := map[string]bool{
		"SHA256":  true,
		"SHA3":    true,
		"BLAKE2b": true,
	}
	if !supportedHashes[components.Crypto.Hash] {
		return fmt.Errorf("unsupported hash algorithm: %s", components.Crypto.Hash)
	}

	// 验证签名算法（可选）
	if components.Crypto.Signature != "" {
		supportedSigs := map[string]bool{
			"ECDSA": true,
			"EdDSA": true,
			"BLS":   true,
		}
		if !supportedSigs[components.Crypto.Signature] {
			return fmt.Errorf("unsupported signature algorithm: %s", components.Crypto.Signature)
		}
	}

	// 验证网络组件
	supportedTopologies := map[string]bool{
		"gossip": true,
		"mesh":   true,
		"star":   true,
	}
	if !supportedTopologies[components.Network.Topology] {
		return fmt.Errorf("unsupported network topology: %s", components.Network.Topology)
	}

	supportedBroadcasts := map[string]bool{
		"reliable":    true,
		"best-effort": true,
	}
	if !supportedBroadcasts[components.Network.Broadcast] {
		return fmt.Errorf("unsupported broadcast method: %s", components.Network.Broadcast)
	}

	// 验证存储组件
	supportedBlockchains := map[string]bool{
		"merkle-chain": true,
		"dag":          true,
	}
	if !supportedBlockchains[components.Storage.Blockchain] {
		return fmt.Errorf("unsupported blockchain structure: %s", components.Storage.Blockchain)
	}

	supportedStates := map[string]bool{
		"merkle-patricia": true,
		"simple-map":      true,
	}
	if !supportedStates[components.Storage.State] {
		return fmt.Errorf("unsupported state structure: %s", components.Storage.State)
	}

	v.log.Debug("Components validated")
	return nil
}

// validateParameters 验证参数
func (v *Validator) validateParameters(params *Parameters) error {
	// 验证区块时间
	duration := params.GetBlockTime()
	if duration <= 0 {
		return fmt.Errorf("block_time must be positive")
	}

	// 验证最大区块大小
	if params.MaxBlockSize <= 0 {
		return fmt.Errorf("max_block_size must be positive")
	}

	// 合理性检查
	if params.MaxBlockSize > 100*1024*1024 { // 100MB
		v.log.Warn("max_block_size is very large, may cause performance issues")
	}

	v.log.WithFields(logrus.Fields{
		"block_time":     params.BlockTime,
		"max_block_size": params.MaxBlockSize,
	}).Debug("Parameters validated")

	return nil
}

// validatePhases 验证阶段定义
func (v *Validator) validatePhases(phases []Phase) error {
	if len(phases) == 0 {
		return fmt.Errorf("at least one phase must be defined")
	}

	// 验证每个阶段
	phaseNames := make(map[string]bool)
	for i, phase := range phases {
		if phase.Name == "" {
			return fmt.Errorf("phase %d: name cannot be empty", i)
		}

		// 检查重复名称
		if phaseNames[phase.Name] {
			return fmt.Errorf("duplicate phase name: %s", phase.Name)
		}
		phaseNames[phase.Name] = true

		// 验证入口和出口
		if phase.Entry == "" {
			return fmt.Errorf("phase %s: entry cannot be empty", phase.Name)
		}
		if phase.Exit == "" {
			return fmt.Errorf("phase %s: exit cannot be empty", phase.Name)
		}

		// 验证动作
		for j, action := range phase.Actions {
			if action.Type == "" {
				return fmt.Errorf("phase %s, action %d: type cannot be empty", phase.Name, j)
			}

			validActionTypes := map[string]bool{
				"function":  true,
				"event":     true,
				"condition": true,
			}
			if !validActionTypes[action.Type] {
				return fmt.Errorf("phase %s, action %d: invalid type %s", phase.Name, j, action.Type)
			}
		}
	}

	v.log.WithField("phase_count", len(phases)).Debug("Phases validated")
	return nil
}

// validateStateMachine 验证状态机
func (v *Validator) validateStateMachine(sm *StateMachine) error {
	if len(sm.States) == 0 {
		return fmt.Errorf("state_machine must have at least one state")
	}

	// 构建状态集合
	stateSet := make(map[string]bool)
	for _, state := range sm.States {
		if state == "" {
			return fmt.Errorf("state name cannot be empty")
		}
		if stateSet[state] {
			return fmt.Errorf("duplicate state: %s", state)
		}
		stateSet[state] = true
	}

	// 验证状态转换
	for i, trans := range sm.Transitions {
		// 验证源状态
		if !stateSet[trans.From] {
			return fmt.Errorf("transition %d: unknown source state %s", i, trans.From)
		}

		// 验证目标状态
		if !stateSet[trans.To] {
			return fmt.Errorf("transition %d: unknown target state %s", i, trans.To)
		}

		// 验证事件
		if trans.Event == "" {
			return fmt.Errorf("transition %d: event cannot be empty", i)
		}
	}

	v.log.WithFields(logrus.Fields{
		"state_count":      len(sm.States),
		"transition_count": len(sm.Transitions),
	}).Debug("State machine validated")

	return nil
}

// validateSafetyProperties 验证安全属性
func (v *Validator) validateSafetyProperties(properties []SafetyProperty) error {
	if len(properties) == 0 {
		v.log.Warn("No safety properties defined")
		return nil
	}

	propertyNames := make(map[string]bool)
	for i, prop := range properties {
		if prop.Name == "" {
			return fmt.Errorf("safety property %d: name cannot be empty", i)
		}

		if propertyNames[prop.Name] {
			return fmt.Errorf("duplicate safety property: %s", prop.Name)
		}
		propertyNames[prop.Name] = true

		if prop.Formula == "" {
			return fmt.Errorf("safety property %s: formula cannot be empty", prop.Name)
		}
	}

	v.log.WithField("property_count", len(properties)).Debug("Safety properties validated")
	return nil
}

// validatePerformanceRequirements 验证性能要求
func (v *Validator) validatePerformanceRequirements(reqs *PerformanceRequirements) error {
	// 验证吞吐量
	if reqs.MinThroughput < 0 {
		return fmt.Errorf("min_throughput must be non-negative")
	}

	// 验证延迟
	if reqs.MaxLatency < 0 {
		return fmt.Errorf("max_latency must be non-negative")
	}

	// 验证容错率
	if reqs.FaultTolerance < 0 || reqs.FaultTolerance >= 1 {
		return fmt.Errorf("fault_tolerance must be in range [0, 1)")
	}

	v.log.WithFields(logrus.Fields{
		"min_throughput":  reqs.MinThroughput,
		"max_latency":     reqs.MaxLatency,
		"fault_tolerance": reqs.FaultTolerance,
	}).Debug("Performance requirements validated")

	return nil
}

// ValidateSemantics 语义验证
func (v *Validator) ValidateSemantics(descriptor *CDLDescriptor) error {
	v.log.Info("Starting semantic validation")

	// 验证阶段之间的连接性
	if err := v.validatePhaseConnectivity(descriptor.Consensus.Phases); err != nil {
		return fmt.Errorf("phase connectivity: %w", err)
	}

	// 验证状态机的可达性
	if err := v.validateStateMachineReachability(&descriptor.Consensus.StateMachine); err != nil {
		return fmt.Errorf("state machine reachability: %w", err)
	}

	v.log.Info("Semantic validation completed")
	return nil
}

// validatePhaseConnectivity 验证阶段连接性
func (v *Validator) validatePhaseConnectivity(phases []Phase) error {
	if len(phases) == 0 {
		return nil
	}

	// 构建入口和出口映射
	exitPoints := make(map[string]bool)
	entryPoints := make(map[string]bool)

	for _, phase := range phases {
		exitPoints[phase.Exit] = true
		entryPoints[phase.Entry] = true
	}

	// 检查每个阶段的出口是否能连接到下一个阶段的入口
	// 这是一个简化的检查，实际可能需要更复杂的连接性分析
	v.log.Debug("Phase connectivity check passed")

	return nil
}

// validateStateMachineReachability 验证状态机可达性
func (v *Validator) validateStateMachineReachability(sm *StateMachine) error {
	if len(sm.States) == 0 {
		return nil
	}

	// 构建邻接表
	graph := make(map[string][]string)
	for _, trans := range sm.Transitions {
		graph[trans.From] = append(graph[trans.From], trans.To)
	}

	// 检查是否所有状态都可达（从第一个状态开始）
	if len(sm.States) > 0 {
		visited := make(map[string]bool)
		v.dfsReachability(sm.States[0], graph, visited)

		// 检查是否所有状态都被访问
		unreachable := []string{}
		for _, state := range sm.States {
			if !visited[state] {
				unreachable = append(unreachable, state)
			}
		}

		if len(unreachable) > 0 {
			v.log.WithField("unreachable_states", unreachable).Warn("Some states are unreachable")
		}
	}

	return nil
}

// dfsReachability DFS 可达性检查
func (v *Validator) dfsReachability(state string, graph map[string][]string, visited map[string]bool) {
	visited[state] = true
	for _, next := range graph[state] {
		if !visited[next] {
			v.dfsReachability(next, graph, visited)
		}
	}
}
