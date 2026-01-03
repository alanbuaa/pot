package cdl

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/consensus/model"
	"github.com/zzz136454872/upgradeable-consensus/p2p"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
)

// ConsensusRuntime CDL 编译后的共识运行时
type ConsensusRuntime struct {
	// 元信息
	Descriptor  *CDLDescriptor
	ConsensusID int64
	Config      *config.ConsensusConfig
	P2PAdaptor  p2p.P2PAdaptor

	// 编译后的组件
	Components   *RuntimeComponents
	Parameters   *RuntimeParameters
	StateMachine *RuntimeStateMachine

	// 运行时状态
	running bool
	mu      sync.RWMutex
	log     *logrus.Entry

	// 共识实例（实际执行的共识）
	consensusInstance model.Consensus

	// 通道
	requestChan chan interface{}
	msgChan     chan []byte
	stopChan    chan struct{}
}

// NewConsensusRuntime 创建共识运行时
func NewConsensusRuntime(
	descriptor *CDLDescriptor,
	consensusID int64,
	cfg *config.ConsensusConfig,
	p2pAdaptor p2p.P2PAdaptor,
) (*ConsensusRuntime, error) {
	log := logrus.WithFields(logrus.Fields{
		"consensus": descriptor.Consensus.Name,
		"id":        consensusID,
	})

	// 编译 CDL
	compiler := NewCompiler(log)
	runtime, err := compiler.Compile(descriptor, consensusID, cfg, p2pAdaptor)
	if err != nil {
		return nil, fmt.Errorf("failed to compile CDL: %w", err)
	}

	// 初始化通道
	runtime.requestChan = make(chan interface{}, 1000)
	runtime.msgChan = make(chan []byte, 1000)
	runtime.stopChan = make(chan struct{})

	return runtime, nil
}

// Run 启动共识运行时
func (r *ConsensusRuntime) Run() {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return
	}
	r.running = true
	r.mu.Unlock()

	r.log.Info("Starting consensus runtime")

	// 根据 CDL 类型创建具体的共识实例
	var err error
	r.consensusInstance, err = r.createConsensusInstance()
	if err != nil {
		r.log.WithError(err).Error("Failed to create consensus instance")
		return
	}

	// 注意：Consensus 接口没有 Run 方法，由具体实现自行启动

	r.log.Info("Consensus runtime started")
}

// Stop 停止共识运行时
func (r *ConsensusRuntime) Stop() {
	r.mu.Lock()
	if !r.running {
		r.mu.Unlock()
		return
	}
	r.running = false
	r.mu.Unlock()

	r.log.Info("Stopping consensus runtime")

	// 停止共识实例
	if r.consensusInstance != nil {
		r.consensusInstance.Stop()
	}

	// 关闭通道
	close(r.stopChan)

	r.log.Info("Consensus runtime stopped")
}

// createConsensusInstance 根据 CDL 创建共识实例
func (r *ConsensusRuntime) createConsensusInstance() (model.Consensus, error) {
	consensusType := r.Descriptor.Consensus.Type

	r.log.WithField("type", consensusType).Debug("Creating consensus instance")

	// 这里简化处理，实际应该根据 CDL 动态创建
	// 对于已知类型，调用对应的工厂函数
	switch consensusType {
	case "pow":
		// 返回 PoW 实例的包装器
		return r.createPoWInstance()
	case "hotstuff":
		// 返回 HotStuff 实例的包装器
		return r.createHotStuffInstance()
	case "pot":
		// 返回 PoT 实例的包装器
		return r.createPoTInstance()
	default:
		// 对于自定义共识，创建通用的运行时包装器
		return r.createGenericInstance()
	}
}

// createPoWInstance 创建 PoW 实例
func (r *ConsensusRuntime) createPoWInstance() (model.Consensus, error) {
	// 这里简化实现，实际需要调用 PoW 共识的构造函数
	return &genericConsensusWrapper{
		runtime: r,
		name:    "pow",
	}, nil
}

// createHotStuffInstance 创建 HotStuff 实例
func (r *ConsensusRuntime) createHotStuffInstance() (model.Consensus, error) {
	// 这里简化实现，实际需要调用 HotStuff 共识的构造函数
	return &genericConsensusWrapper{
		runtime: r,
		name:    "hotstuff",
	}, nil
}

// createPoTInstance 创建 PoT 实例
func (r *ConsensusRuntime) createPoTInstance() (model.Consensus, error) {
	// 这里简化实现，实际需要调用 PoT 共识的构造函数
	return &genericConsensusWrapper{
		runtime: r,
		name:    "pot",
	}, nil
}

// createGenericInstance 创建通用自定义共识实例
func (r *ConsensusRuntime) createGenericInstance() (model.Consensus, error) {
	return &genericConsensusWrapper{
		runtime: r,
		name:    r.Descriptor.Consensus.Name,
	}, nil
}

// GetRequestEntrance 获取请求入口
func (r *ConsensusRuntime) GetRequestEntrance() chan<- *pb.Request {
	return nil // 暂时返回 nil，实际需要实现
}

// GetMsgByteEntrance 获取消息字节入口
func (r *ConsensusRuntime) GetMsgByteEntrance() chan<- []byte {
	return r.msgChan
}

// GetConsensusID 获取共识 ID
func (r *ConsensusRuntime) GetConsensusID() int64 {
	return r.ConsensusID
}

// GetConsensusType 获取共识类型
func (r *ConsensusRuntime) GetConsensusType() string {
	return r.Descriptor.Consensus.Type
}

// GetWeight 获取节点权重
func (r *ConsensusRuntime) GetWeight(nid int64) float64 {
	// 默认权重
	return 1.0
}

// GetMaxAdversaryWeight 获取最大对抗权重
func (r *ConsensusRuntime) GetMaxAdversaryWeight() float64 {
	// 根据容错率计算
	faultTolerance := r.Descriptor.Consensus.PerformanceRequirements.FaultTolerance
	return faultTolerance
}

// VerifyBlock 验证区块
func (r *ConsensusRuntime) VerifyBlock(block []byte, proof []byte) bool {
	// 委托给实际的共识实例
	if r.consensusInstance != nil {
		return r.consensusInstance.VerifyBlock(block, proof)
	}
	return true
}

// UpdateExternalStatus 更新外部状态
func (r *ConsensusRuntime) UpdateExternalStatus(status model.ExternalStatus) {
	if r.consensusInstance != nil {
		r.consensusInstance.UpdateExternalStatus(status)
	}
}

// NewEpochConfirmation 新纪元确认
func (r *ConsensusRuntime) NewEpochConfirmation(epoch int64, proof []byte, committee []string) {
	if r.consensusInstance != nil {
		r.consensusInstance.NewEpochConfirmation(epoch, proof, committee)
	}
}

// RequestLatestBlock 请求最新区块
func (r *ConsensusRuntime) RequestLatestBlock(epoch int64, proof []byte, committee []string) {
	if r.consensusInstance != nil {
		r.consensusInstance.RequestLatestBlock(epoch, proof, committee)
	}
}

// GetDescriptor 获取 CDL 描述符
func (r *ConsensusRuntime) GetDescriptor() *CDLDescriptor {
	return r.Descriptor
}

// GetComponents 获取运行时组件
func (r *ConsensusRuntime) GetComponents() *RuntimeComponents {
	return r.Components
}

// GetParameters 获取运行时参数
func (r *ConsensusRuntime) GetParameters() *RuntimeParameters {
	return r.Parameters
}

// GetStateMachine 获取状态机
func (r *ConsensusRuntime) GetStateMachine() *RuntimeStateMachine {
	return r.StateMachine
}

// IsRunning 检查是否正在运行
func (r *ConsensusRuntime) IsRunning() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.running
}

// Hash 计算运行时配置哈希
func (r *ConsensusRuntime) Hash() []byte {
	return r.Descriptor.Hash()
}

// genericConsensusWrapper 通用共识包装器
// 实现 model.Consensus 接口
type genericConsensusWrapper struct {
	runtime *ConsensusRuntime
	name    string
	running bool
	mu      sync.RWMutex
}

// Run 运行共识
func (w *genericConsensusWrapper) Run() {
	w.mu.Lock()
	w.running = true
	w.mu.Unlock()

	w.runtime.log.WithField("consensus", w.name).Info("Generic consensus started")

	// 这里是简化实现，实际需要根据 CDL 的阶段定义执行
	<-w.runtime.stopChan
}

// Stop 停止共识
func (w *genericConsensusWrapper) Stop() {
	w.mu.Lock()
	w.running = false
	w.mu.Unlock()
}

// GetRequestEntrance 获取请求入口
func (w *genericConsensusWrapper) GetRequestEntrance() chan<- *pb.Request {
	return nil // 暂时返回 nil
}

// GetMsgByteEntrance 获取消息字节入口
func (w *genericConsensusWrapper) GetMsgByteEntrance() chan<- []byte {
	return w.runtime.msgChan
}

// GetConsensusID 获取共识 ID
func (w *genericConsensusWrapper) GetConsensusID() int64 {
	return w.runtime.ConsensusID
}

// GetConsensusType 获取共识类型
func (w *genericConsensusWrapper) GetConsensusType() string {
	return w.runtime.Descriptor.Consensus.Type
}

// GetWeight 获取节点权重
func (w *genericConsensusWrapper) GetWeight(nid int64) float64 {
	return 1.0
}

// GetMaxAdversaryWeight 获取最大对抗权重
func (w *genericConsensusWrapper) GetMaxAdversaryWeight() float64 {
	faultTolerance := w.runtime.Descriptor.Consensus.PerformanceRequirements.FaultTolerance
	return faultTolerance
}

// VerifyBlock 验证区块
func (w *genericConsensusWrapper) VerifyBlock(block []byte, proof []byte) bool {
	// 简化实现，只做基本验证
	if block == nil {
		return false
	}
	return true
}

// UpdateExternalStatus 更新外部状态
func (w *genericConsensusWrapper) UpdateExternalStatus(status model.ExternalStatus) {
	// 简化实现
}

// NewEpochConfirmation 新纪元确认
func (w *genericConsensusWrapper) NewEpochConfirmation(epoch int64, proof []byte, committee []string) {
	// 简化实现
}

// RequestLatestBlock 请求最新区块
func (w *genericConsensusWrapper) RequestLatestBlock(epoch int64, proof []byte, committee []string) {
	// 简化实现
}
