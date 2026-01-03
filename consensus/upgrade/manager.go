package upgrade

import (
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/consensus/model"
	"github.com/zzz136454872/upgradeable-consensus/internal/storage"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
)

// UpgradeManager 升级管理器
// 统一管理整个升级流程
type UpgradeManager struct {
	// 当前运行的共识
	currentConsensus model.Consensus

	// 升级组件
	dualChainManager *DualChainManager
	metricsCollector *MetricsCollector
	preexecMonitor   *PreexecMonitor
	switchManager    *SwitchManager
	rollbackManager  *RollbackManager
	messageCache     *MessageCache
	consensusFactory *ConsensusFactory

	// 持久化
	persistence UpgradePersistence

	// 升级提案
	currentProposal *UpgradeProposal
	upgradeState    *UpgradeState

	// 配置
	config  *config.ConsensusConfig
	storage storage.DualChainStorage

	log *logrus.Entry
	mu  sync.RWMutex
}

// GetPersistence returns the persistence layer
func (m *UpgradeManager) GetPersistence() UpgradePersistence {
	return m.persistence
}

// UpgradeState 升级状态
type UpgradeState struct {
	Phase           UpgradePhase
	CurrentProposal *UpgradeProposal
	Started         bool
	Completed       bool
	Failed          bool
	FailureReason   string
	StartTime       time.Time
	EndTime         time.Time
}

// UpgradePhase 升级阶段
type UpgradePhase int

const (
	PhaseIdle UpgradePhase = iota
	PhasePreparing
	PhasePreexecuting
	PhaseSwitching
	PhaseCompleted
	PhaseRolledBack
	PhaseFailed
)

func (p UpgradePhase) String() string {
	switch p {
	case PhaseIdle:
		return "Idle"
	case PhasePreparing:
		return "Preparing"
	case PhasePreexecuting:
		return "Preexecuting"
	case PhaseSwitching:
		return "Switching"
	case PhaseCompleted:
		return "Completed"
	case PhaseRolledBack:
		return "RolledBack"
	case PhaseFailed:
		return "Failed"
	default:
		return "Unknown"
	}
}

// NewUpgradeManager 创建升级管理器
func NewUpgradeManager(
	currentConsensus model.Consensus,
	config *config.ConsensusConfig,
	dualChainStorage storage.DualChainStorage,
	log *logrus.Entry,
) *UpgradeManager {
	if log == nil {
		log = logrus.NewEntry(logrus.New())
	}

	um := &UpgradeManager{
		currentConsensus: currentConsensus,
		config:           config,
		storage:          dualChainStorage,
		log:              log,
		upgradeState: &UpgradeState{
			Phase: PhaseIdle,
		},
	}

	// 初始化组件
	um.dualChainManager = NewDualChainManager(currentConsensus, dualChainStorage, log)
	um.messageCache = NewMessageCache(log)

	return um
}

// NewUpgradeManagerWithPersistence 创建带持久化的升级管理器
func NewUpgradeManagerWithPersistence(
	currentConsensus model.Consensus,
	config *config.ConsensusConfig,
	dualChainStorage storage.DualChainStorage,
	persistence UpgradePersistence,
	log *logrus.Entry,
) (*UpgradeManager, error) {
	um := NewUpgradeManager(currentConsensus, config, dualChainStorage, log)
	um.persistence = persistence

	// 尝试恢复状态
	if err := um.recoverState(); err != nil {
		log.WithError(err).Warn("Failed to recover upgrade state, starting fresh")
	}

	return um, nil
}

// recoverState 从持久化存储恢复状态
func (um *UpgradeManager) recoverState() error {
	if um.persistence == nil {
		return nil
	}

	// 加载升级状态
	state, err := um.persistence.LoadState()
	if err != nil {
		return err
	}

	um.upgradeState = state

	// 如果有当前提案，加载提案详情
	if state.CurrentProposal != nil {
		proposal, err := um.persistence.LoadProposal(state.CurrentProposal.ProposalID)
		if err == nil {
			um.currentProposal = proposal
		}
	}

	um.log.WithFields(logrus.Fields{
		"phase":     state.Phase.String(),
		"started":   state.Started,
		"completed": state.Completed,
		"failed":    state.Failed,
	}).Info("Recovered upgrade state from persistence")

	return nil
}

// StartUpgrade 启动升级流程
func (um *UpgradeManager) StartUpgrade(proposal *UpgradeProposal, newConsensus model.Consensus) error {
	um.mu.Lock()
	defer um.mu.Unlock()

	if um.upgradeState.Started {
		return fmt.Errorf("upgrade already in progress")
	}

	um.log.WithFields(logrus.Fields{
		"proposal_id":      proposal.ProposalID.String(),
		"target_consensus": proposal.TargetConsensus,
		"preexec_height":   proposal.PreexecStartHeight,
		"switch_height":    proposal.SwitchHeight,
	}).Info("Starting upgrade process")

	// 更新状态
	um.currentProposal = proposal
	um.upgradeState.CurrentProposal = proposal
	um.upgradeState.Phase = PhasePreparing
	um.upgradeState.Started = true
	um.upgradeState.StartTime = time.Now()

	// 持久化提案和状态
	if err := um.persistProposal(proposal); err != nil {
		um.log.WithError(err).Warn("Failed to persist proposal")
	}
	if err := um.persistState(); err != nil {
		um.log.WithError(err).Warn("Failed to persist state")
	}
	um.recordEvent(EventUpgradeStarted, "Upgrade process started")

	// 设置消息缓存信息
	um.messageCache.SetSwitchInfo(
		proposal.SwitchHeight,
		um.currentConsensus.GetConsensusID(),
		newConsensus.GetConsensusID(),
	)

	// 启动预执行
	um.log.Info("Starting preexecution chain")
	err := um.dualChainManager.StartPreexecution(proposal.PreexecStartHeight, newConsensus)
	if err != nil {
		um.upgradeState.Phase = PhaseFailed
		um.upgradeState.Failed = true
		um.upgradeState.FailureReason = fmt.Sprintf("Failed to start preexecution: %v", err)
		return err
	}

	// 初始化预执行监控器
	um.preexecMonitor = NewPreexecMonitor(
		proposal,
		um.dualChainManager,
		um.log,
	)

	// 初始化指标收集器（从监控器获取）
	um.metricsCollector = um.preexecMonitor.metricsCollector

	// 初始化切换和回退管理器
	um.switchManager = NewSwitchManager(um.dualChainManager, um.preexecMonitor, um.log)
	um.rollbackManager = NewRollbackManager(um.dualChainManager, um.preexecMonitor, um.log)

	// 准备切换
	err = um.switchManager.PrepareSwitch(proposal.SwitchHeight)
	if err != nil {
		um.upgradeState.Phase = PhaseFailed
		um.upgradeState.Failed = true
		um.upgradeState.FailureReason = fmt.Sprintf("Failed to prepare switch: %v", err)
		return err
	}

	// 启动监控
	um.preexecMonitor.Start()

	// 启动自动回退监控
	go um.rollbackManager.MonitorAndRollback()

	// 更新状态
	um.upgradeState.Phase = PhasePreexecuting

	um.log.Info("Upgrade process started successfully")
	return nil
}

// ProcessBlock 处理区块
func (um *UpgradeManager) ProcessBlock(block *pb.Block, isMainChain bool) error {
	um.mu.RLock()
	defer um.mu.RUnlock()

	if !um.upgradeState.Started {
		return nil // 未启动升级，无需处理
	}

	if isMainChain {
		// 处理主链区块
		// 注意：这里需要将 pb.Block 转换为 types.Block
		// 为了简化，我们假设已经转换或直接使用 pb.Block
		// 实际使用中可能需要添加转换逻辑

		// 检查是否达到切换高度
		if block.Header.Height >= um.currentProposal.SwitchHeight &&
			um.upgradeState.Phase == PhasePreexecuting {
			um.log.Info("Reached switch height, initiating switch")
			go um.executeSwitch()
		}
	} else {
		// 处理预执行链区块
		// 注意：类型转换问题同上

		// 通知监控器
		if um.preexecMonitor != nil {
			// 记录区块处理（时间使用占位值）
			um.preexecMonitor.OnBlockProcessed(
				block.Header.Height,
				time.Duration(0),
				uint32(len(block.Txs)),
				nil,
			)
		}
	}

	return nil
}

// executeSwitch 执行切换（内部方法）
func (um *UpgradeManager) executeSwitch() {
	um.mu.Lock()
	um.upgradeState.Phase = PhaseSwitching
	um.mu.Unlock()

	um.log.Info("Executing consensus switch")

	err := um.switchManager.ExecuteSwitch()
	if err != nil {
		um.log.WithError(err).Error("Switch execution failed")
		um.mu.Lock()
		um.upgradeState.Phase = PhaseFailed
		um.upgradeState.Failed = true
		um.upgradeState.FailureReason = fmt.Sprintf("Switch failed: %v", err)
		um.mu.Unlock()
		return
	}

	// 切换成功，处理缓存的消息
	cachedMessages := um.messageCache.OnSwitch()
	um.log.WithField("cached_count", len(cachedMessages)).Info("Processing cached messages after switch")

	// 更新状态
	um.mu.Lock()
	um.upgradeState.Phase = PhaseCompleted
	um.upgradeState.Completed = true
	um.upgradeState.EndTime = time.Now()
	um.mu.Unlock()

	um.log.Info("Upgrade completed successfully")
}

// GetUpgradeState 获取升级状态
func (um *UpgradeManager) GetUpgradeState() *UpgradeState {
	um.mu.RLock()
	defer um.mu.RUnlock()

	// 返回状态副本
	stateCopy := *um.upgradeState
	return &stateCopy
}

// GetCurrentPhase 获取当前阶段
func (um *UpgradeManager) GetCurrentPhase() UpgradePhase {
	um.mu.RLock()
	defer um.mu.RUnlock()
	return um.upgradeState.Phase
}

// IsUpgrading 检查是否正在升级
func (um *UpgradeManager) IsUpgrading() bool {
	um.mu.RLock()
	defer um.mu.RUnlock()
	return um.upgradeState.Started && !um.upgradeState.Completed && !um.upgradeState.Failed
}

// GetMetrics 获取预执行指标
func (um *UpgradeManager) GetMetrics() *PerformanceMetrics {
	um.mu.RLock()
	defer um.mu.RUnlock()

	if um.metricsCollector == nil {
		return nil
	}

	return um.metricsCollector.ComputeAggregateMetrics()
}

// Stop 停止升级管理器
func (um *UpgradeManager) Stop() {
	um.mu.Lock()
	defer um.mu.Unlock()

	um.log.Info("Stopping upgrade manager")

	if um.preexecMonitor != nil {
		um.preexecMonitor.Stop()
	}

	if um.messageCache != nil {
		um.messageCache.Clear()
	}
}

// Reset 重置升级管理器（测试用）
func (um *UpgradeManager) Reset() {
	um.mu.Lock()
	defer um.mu.Unlock()

	um.upgradeState = &UpgradeState{
		Phase: PhaseIdle,
	}
	um.currentProposal = nil

	if um.switchManager != nil {
		um.switchManager.Reset()
	}
	if um.rollbackManager != nil {
		um.rollbackManager.Reset()
	}
	if um.messageCache != nil {
		um.messageCache.Clear()
	}
}

// persistState 持久化当前状态
func (um *UpgradeManager) persistState() error {
	if um.persistence == nil {
		return nil
	}
	return um.persistence.SaveState(um.upgradeState)
}

// persistProposal 持久化提案
func (um *UpgradeManager) persistProposal(proposal *UpgradeProposal) error {
	if um.persistence == nil {
		return nil
	}
	return um.persistence.SaveProposal(proposal)
}

// recordEvent 记录事件
func (um *UpgradeManager) recordEvent(eventType EventType, description string) {
	if um.persistence == nil {
		return
	}

	event := &UpgradeEvent{
		Type:        eventType,
		Phase:       um.upgradeState.Phase,
		Description: description,
		Timestamp:   time.Now(),
	}

	if um.currentProposal != nil {
		event.ProposalID = um.currentProposal.ProposalID
	}

	if err := um.persistence.SaveEvent(event); err != nil {
		um.log.WithError(err).Warn("Failed to save event")
	}
}

// Close 关闭升级管理器
func (um *UpgradeManager) Close() error {
	um.Stop()

	if um.persistence != nil {
		return um.persistence.Close()
	}

	return nil
}

// NewUpgradeManagerWithFactory 创建带 ConsensusFactory 的升级管理器
func NewUpgradeManagerWithFactory(
	currentConsensus model.Consensus,
	config *config.ConsensusConfig,
	dualChainStorage storage.DualChainStorage,
	persistence UpgradePersistence,
	nid int64,
	executor interface{},
	p2pAdaptor interface{},
	log *logrus.Entry,
) (*UpgradeManager, error) {
	um, err := NewUpgradeManagerWithPersistence(
		currentConsensus,
		config,
		dualChainStorage,
		persistence,
		log,
	)
	if err != nil {
		return nil, err
	}

	// 创建 ConsensusFactory (需要正确的类型)
	// 注意: executor 和 p2pAdaptor 需要正确的类型转换
	// 这里暂时跳过,等待类型确认
	if nid != 0 {
		um.log.Debug("ConsensusFactory integration pending proper type conversion")
	}

	return um, nil
}

// SetConsensusFactory 设置共识工厂
func (um *UpgradeManager) SetConsensusFactory(factory *ConsensusFactory) {
	um.mu.Lock()
	defer um.mu.Unlock()
	um.consensusFactory = factory
	um.log.Info("ConsensusFactory configured for upgrade manager")
}

// StartUpgradeWithFactory 使用 ConsensusFactory 启动升级
// 自动创建新共识实例,无需手动传入
func (um *UpgradeManager) StartUpgradeWithFactory(proposal *UpgradeProposal) error {
	um.mu.Lock()

	if um.consensusFactory == nil {
		um.mu.Unlock()
		return fmt.Errorf("ConsensusFactory not configured")
	}

	// 验证提案
	if err := um.consensusFactory.ValidateProposal(proposal); err != nil {
		um.mu.Unlock()
		return fmt.Errorf("invalid proposal: %w", err)
	}

	// 创建新共识实例
	newConsensus, err := um.consensusFactory.CreateConsensus(proposal, um.config)
	if err != nil {
		um.mu.Unlock()
		return fmt.Errorf("failed to create consensus: %w", err)
	}

	um.mu.Unlock()

	// 调用原有的 StartUpgrade
	return um.StartUpgrade(proposal, newConsensus)
}
