package cdl

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/p2p"
)

// Compiler CDL 编译器
type Compiler struct {
	log *logrus.Entry
}

// NewCompiler 创建 CDL 编译器
func NewCompiler(log *logrus.Entry) *Compiler {
	if log == nil {
		log = logrus.NewEntry(logrus.New())
	}
	return &Compiler{
		log: log,
	}
}

// Compile 编译 CDL 到可执行的共识运行时
func (c *Compiler) Compile(
	descriptor *CDLDescriptor,
	consensusID int64,
	cfg *config.ConsensusConfig,
	p2pAdaptor p2p.P2PAdaptor,
) (*ConsensusRuntime, error) {
	c.log.WithFields(logrus.Fields{
		"name":         descriptor.Consensus.Name,
		"version":      descriptor.Consensus.Version,
		"consensus_id": consensusID,
	}).Info("Starting CDL compilation")

	// 验证 CDL
	validator := NewValidator(c.log)
	if err := validator.Validate(descriptor); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// 编译组件
	components, err := c.compileComponents(&descriptor.Consensus.Components)
	if err != nil {
		return nil, fmt.Errorf("failed to compile components: %w", err)
	}

	// 编译参数
	params, err := c.compileParameters(&descriptor.Consensus.Parameters)
	if err != nil {
		return nil, fmt.Errorf("failed to compile parameters: %w", err)
	}

	// 编译状态机
	stateMachine, err := c.compileStateMachine(&descriptor.Consensus.StateMachine)
	if err != nil {
		return nil, fmt.Errorf("failed to compile state machine: %w", err)
	}

	// 创建运行时
	runtime := &ConsensusRuntime{
		Descriptor:   descriptor,
		ConsensusID:  consensusID,
		Config:       cfg,
		P2PAdaptor:   p2pAdaptor,
		Components:   components,
		Parameters:   params,
		StateMachine: stateMachine,
		log:          c.log,
	}

	c.log.Info("CDL compilation completed successfully")
	return runtime, nil
}

// compileComponents 编译组件配置
func (c *Compiler) compileComponents(
	components *Components,
) (*RuntimeComponents, error) {
	c.log.Debug("Compiling components")

	// 编译加密组件
	cryptoComp, err := c.compileCryptoComponent(&components.Crypto)
	if err != nil {
		return nil, err
	}

	// 编译网络组件
	networkComp, err := c.compileNetworkComponent(&components.Network)
	if err != nil {
		return nil, err
	}

	// 编译存储组件
	storageComp, err := c.compileStorageComponent(&components.Storage)
	if err != nil {
		return nil, err
	}

	return &RuntimeComponents{
		Crypto:  cryptoComp,
		Network: networkComp,
		Storage: storageComp,
	}, nil
}

// compileCryptoComponent 编译加密组件
func (c *Compiler) compileCryptoComponent(
	cryptoComp *CryptoComponent,
) (*RuntimeCryptoComponent, error) {
	c.log.WithField("hash", cryptoComp.Hash).Debug("Compiling crypto component")

	// 创建哈希函数
	var hashFunc func([]byte) []byte
	switch cryptoComp.Hash {
	case "SHA256", "SHA3", "BLAKE2b":
		hashFunc = func(data []byte) []byte {
			return crypto.Hash(data)
		}
	default:
		return nil, fmt.Errorf("unsupported hash algorithm: %s", cryptoComp.Hash)
	}

	return &RuntimeCryptoComponent{
		HashAlgorithm:      cryptoComp.Hash,
		SignatureAlgorithm: cryptoComp.Signature,
		VRFAlgorithm:       cryptoComp.VRF,
		VDFAlgorithm:       cryptoComp.VDF,
		HashFunc:           hashFunc,
	}, nil
}

// compileNetworkComponent 编译网络组件
func (c *Compiler) compileNetworkComponent(network *NetworkComponent) (*RuntimeNetworkComponent, error) {
	c.log.WithFields(logrus.Fields{
		"topology":  network.Topology,
		"broadcast": network.Broadcast,
	}).Debug("Compiling network component")

	return &RuntimeNetworkComponent{
		Topology:  network.Topology,
		Broadcast: network.Broadcast,
	}, nil
}

// compileStorageComponent 编译存储组件
func (c *Compiler) compileStorageComponent(storage *StorageComponent) (*RuntimeStorageComponent, error) {
	c.log.WithFields(logrus.Fields{
		"blockchain": storage.Blockchain,
		"state":      storage.State,
	}).Debug("Compiling storage component")

	return &RuntimeStorageComponent{
		BlockchainType: storage.Blockchain,
		StateType:      storage.State,
	}, nil
}

// compileParameters 编译参数
func (c *Compiler) compileParameters(params *Parameters) (*RuntimeParameters, error) {
	c.log.Debug("Compiling parameters")

	blockTime := params.GetBlockTime()

	return &RuntimeParameters{
		BlockTime:    blockTime,
		MaxBlockSize: params.MaxBlockSize,
		Custom:       params.Custom,
	}, nil
}

// compileStateMachine 编译状态机
func (c *Compiler) compileStateMachine(sm *StateMachine) (*RuntimeStateMachine, error) {
	c.log.WithField("state_count", len(sm.States)).Debug("Compiling state machine")

	// 构建状态索引
	stateIndex := make(map[string]int)
	for i, state := range sm.States {
		stateIndex[state] = i
	}

	// 构建转换表
	transitionTable := make(map[string]map[string]*RuntimeTransition)
	for _, trans := range sm.Transitions {
		if transitionTable[trans.From] == nil {
			transitionTable[trans.From] = make(map[string]*RuntimeTransition)
		}

		transitionTable[trans.From][trans.Event] = &RuntimeTransition{
			From:      trans.From,
			To:        trans.To,
			Event:     trans.Event,
			Condition: trans.Condition,
			Action:    trans.Action,
		}
	}

	return &RuntimeStateMachine{
		States:          sm.States,
		StateIndex:      stateIndex,
		TransitionTable: transitionTable,
		CurrentState:    sm.States[0], // 初始状态为第一个状态
	}, nil
}

// CompileAndValidate 编译并验证语义
func (c *Compiler) CompileAndValidate(
	descriptor *CDLDescriptor,
	consensusID int64,
	cfg *config.ConsensusConfig,
	p2pAdaptor p2p.P2PAdaptor,
) (*ConsensusRuntime, error) {
	// 先进行语义验证
	validator := NewValidator(c.log)
	if err := validator.ValidateSemantics(descriptor); err != nil {
		return nil, fmt.Errorf("semantic validation failed: %w", err)
	}

	// 然后编译
	return c.Compile(descriptor, consensusID, cfg, p2pAdaptor)
}

// RuntimeComponents 运行时组件
type RuntimeComponents struct {
	Crypto  *RuntimeCryptoComponent
	Network *RuntimeNetworkComponent
	Storage *RuntimeStorageComponent
}

// RuntimeCryptoComponent 运行时加密组件
type RuntimeCryptoComponent struct {
	HashAlgorithm      string
	SignatureAlgorithm string
	VRFAlgorithm       string
	VDFAlgorithm       string
	HashFunc           func([]byte) []byte
}

// RuntimeNetworkComponent 运行时网络组件
type RuntimeNetworkComponent struct {
	Topology  string
	Broadcast string
}

// RuntimeStorageComponent 运行时存储组件
type RuntimeStorageComponent struct {
	BlockchainType string
	StateType      string
}

// RuntimeParameters 运行时参数
type RuntimeParameters struct {
	BlockTime    interface{}
	MaxBlockSize int
	Custom       map[string]interface{}
}

// RuntimeStateMachine 运行时状态机
type RuntimeStateMachine struct {
	States          []string
	StateIndex      map[string]int
	TransitionTable map[string]map[string]*RuntimeTransition
	CurrentState    string
}

// RuntimeTransition 运行时状态转换
type RuntimeTransition struct {
	From      string
	To        string
	Event     string
	Condition string
	Action    string
}

// Transition 执行状态转换
func (rsm *RuntimeStateMachine) Transition(event string) error {
	transitions, ok := rsm.TransitionTable[rsm.CurrentState]
	if !ok {
		return fmt.Errorf("no transitions from state %s", rsm.CurrentState)
	}

	trans, ok := transitions[event]
	if !ok {
		return fmt.Errorf("no transition for event %s from state %s", event, rsm.CurrentState)
	}

	// 执行转换
	rsm.CurrentState = trans.To
	return nil
}

// GetCurrentState 获取当前状态
func (rsm *RuntimeStateMachine) GetCurrentState() string {
	return rsm.CurrentState
}

// CanTransition 检查是否可以执行转换
func (rsm *RuntimeStateMachine) CanTransition(event string) bool {
	transitions, ok := rsm.TransitionTable[rsm.CurrentState]
	if !ok {
		return false
	}

	_, ok = transitions[event]
	return ok
}

// GetAvailableEvents 获取可用事件
func (rsm *RuntimeStateMachine) GetAvailableEvents() []string {
	transitions, ok := rsm.TransitionTable[rsm.CurrentState]
	if !ok {
		return []string{}
	}

	events := make([]string, 0, len(transitions))
	for event := range transitions {
		events = append(events, event)
	}
	return events
}

// OptimizationHints 优化提示
type OptimizationHints struct {
	EnableParallelExecution bool
	EnableCaching           bool
	EnablePipelining        bool
}

// GetOptimizationHints 获取优化提示
func (c *Compiler) GetOptimizationHints(descriptor *CDLDescriptor) *OptimizationHints {
	hints := &OptimizationHints{
		EnableParallelExecution: true,
		EnableCaching:           true,
		EnablePipelining:        false,
	}

	// 根据共识类型调整优化策略
	switch descriptor.Consensus.Type {
	case "pow":
		hints.EnableParallelExecution = false // PoW 通常是串行的
	case "hotstuff", "pbft":
		hints.EnablePipelining = true // BFT 类型可以使用流水线
	}

	return hints
}
