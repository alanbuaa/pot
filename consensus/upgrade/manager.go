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
	"google.golang.org/protobuf/proto"
)

// UpgradeManager 升级管理器
// 统一管理整个升级流程
type UpgradeManager struct {
	// 当前运行的共识
	currentConsensus model.Consensus

	// 升级组件
	multiChainManager *MultiChainManager
	metricsCollector  *MetricsCollector
	preexecMonitor    *PreexecMonitor
	switchManager     *SwitchManager
	rollbackManager   *RollbackManager
	messageCache      *MessageCache
	consensusFactory  *ConsensusFactory

	// 持久化
	persistence UpgradePersistence

	// 升级提案
	currentProposal    *UpgradeProposal
	currentCandidateID string // 当前活跃的候选链ID
	upgradeState       *UpgradeState
	currentEpoch       uint64 // 当前处理的epoch

	// 配置
	config  *config.ConsensusConfig
	storage storage.MultiChainStorage

	log *logrus.Entry
	mu  sync.RWMutex
}

// GetPersistence returns the persistence layer
func (m *UpgradeManager) GetPersistence() UpgradePersistence {
	return m.persistence
}

// GetMultiChainManager returns the multi-chain manager
func (m *UpgradeManager) GetMultiChainManager() *MultiChainManager {
	return m.multiChainManager
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
	multiChainStorage storage.MultiChainStorage,
	log *logrus.Entry,
) *UpgradeManager {
	if log == nil {
		log = logrus.NewEntry(logrus.New())
	}

	um := &UpgradeManager{
		currentConsensus: currentConsensus,
		config:           config,
		storage:          multiChainStorage,
		log:              log,
		upgradeState: &UpgradeState{
			Phase: PhaseIdle,
		},
	}

	// 初始化组件
	um.multiChainManager = NewMultiChainManager(currentConsensus, multiChainStorage, log)
	um.messageCache = NewMessageCache(log)

	return um
}

// NewUpgradeManagerWithPersistence 创建带持久化的升级管理器
func NewUpgradeManagerWithPersistence(
	currentConsensus model.Consensus,
	config *config.ConsensusConfig,
	multiChainStorage storage.MultiChainStorage,
	persistence UpgradePersistence,
	log *logrus.Entry,
) (*UpgradeManager, error) {
	um := NewUpgradeManager(currentConsensus, config, multiChainStorage, log)
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

// ValidateProposal 验证提案
func (um *UpgradeManager) ValidateProposal(proposal *UpgradeProposal) error {
	um.log.Debug("Validating upgrade proposal")
	if proposal == nil {
		um.log.Error("Proposal validation failed: proposal is nil")
		return fmt.Errorf("proposal is nil")
	}

	// 验证目标共识类型
	if proposal.TargetConsensus == "" {
		um.log.Error("Proposal validation failed: target consensus type is empty")
		return fmt.Errorf("target consensus type is empty")
	}

	// 验证高度参数
	if proposal.PreexecStartHeight == 0 {
		um.log.Error("Proposal validation failed: preexec start height is zero")
		return fmt.Errorf("preexec start height is zero")
	}
	if proposal.SwitchHeight == 0 {
		um.log.Error("Proposal validation failed: switch height is zero")
		return fmt.Errorf("switch height is zero")
	}
	if proposal.SwitchHeight <= proposal.PreexecStartHeight {
		um.log.WithFields(logrus.Fields{
			"switch_height":  proposal.SwitchHeight,
			"preexec_height": proposal.PreexecStartHeight,
		}).Error("Proposal validation failed: switch height must be greater than preexec start height")
		return fmt.Errorf("switch height must be greater than preexec start height")
	}

	// 验证阈值
	if proposal.Threshold == 0 {
		um.log.Error("Proposal validation failed: threshold is zero")
		return fmt.Errorf("threshold is zero")
	}

	um.log.WithFields(logrus.Fields{
		"target_consensus": proposal.TargetConsensus,
		"preexec_height":   proposal.PreexecStartHeight,
		"switch_height":    proposal.SwitchHeight,
		"threshold":        proposal.Threshold,
	}).Info("Proposal validation successful")
	return nil
}

// StartUpgrade 启动升级流程
func (um *UpgradeManager) StartUpgrade(proposal *UpgradeProposal, newConsensus model.Consensus) error {
	um.mu.Lock()
	defer um.mu.Unlock()

	if um.upgradeState.Started {
		um.log.Warn("Upgrade already in progress, rejecting new upgrade request")
		return fmt.Errorf("upgrade already in progress")
	}

	// 验证提案
	if err := um.ValidateProposal(proposal); err != nil {
		um.log.WithError(err).Error("Proposal validation failed")
		return fmt.Errorf("proposal validation failed: %w", err)
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

	// 生成候选链ID
	um.currentCandidateID = fmt.Sprintf("%s-upgrade-%x", proposal.TargetConsensus, proposal.ProposalID)
	um.log.WithField("candidate_id", um.currentCandidateID).Info("Generated candidate chain ID")

	// 启动候选链
	um.log.WithFields(logrus.Fields{
		"candidate_id": um.currentCandidateID,
		"fork_height":  proposal.PreexecStartHeight,
	}).Info("Starting candidate chain")
	err := um.multiChainManager.StartCandidateChain(um.currentCandidateID, proposal.PreexecStartHeight, newConsensus)
	if err != nil {
		um.log.WithError(err).Error("Failed to start candidate chain")
		um.upgradeState.Phase = PhaseFailed
		um.upgradeState.Failed = true
		um.upgradeState.FailureReason = fmt.Sprintf("Failed to start candidate chain: %v", err)
		return err
	}
	um.log.Info("Candidate chain started successfully")

	// 候选共识启动后，冲刷缓存的消息（内部调用，不加锁）
	if err := um.onCandidateConsensusStartedNoLock(newConsensus.GetConsensusID()); err != nil {
		um.log.WithError(err).Warn("Failed to flush cached messages to candidate consensus")
	}

	// 初始化预执行监控器
	um.log.Debug("Initializing preexecution monitor")
	um.preexecMonitor = NewPreexecMonitor(
		proposal,
		um.currentCandidateID,
		um.multiChainManager,
		um.log,
	)

	// 初始化指标收集器（从监控器获取）
	um.metricsCollector = um.preexecMonitor.metricsCollector
	um.log.Debug("Metrics collector initialized")

	// 初始化切换和回退管理器
	um.log.Debug("Initializing switch and rollback managers")
	um.switchManager = NewSwitchManager(um.currentCandidateID, um.multiChainManager, um.preexecMonitor, um.log)
	um.rollbackManager = NewRollbackManager(um.currentCandidateID, um.multiChainManager, um.preexecMonitor, um.log)

	// 准备切换
	um.log.WithField("switch_height", proposal.SwitchHeight).Info("Preparing consensus switch")
	err = um.switchManager.PrepareSwitch(proposal.SwitchHeight)
	if err != nil {
		um.log.WithError(err).Error("Failed to prepare switch")
		um.upgradeState.Phase = PhaseFailed
		um.upgradeState.Failed = true
		um.upgradeState.FailureReason = fmt.Sprintf("Failed to prepare switch: %v", err)
		return err
	}
	um.log.Info("Switch preparation completed")

	// 启动监控
	um.log.Info("Starting preexecution monitor")
	um.preexecMonitor.Start()

	// 启动自动回退监控
	um.log.Info("Starting automatic rollback monitoring")
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
		um.log.WithField("height", block.Header.Height).Trace("Processing main chain block")
		// 注意：这里需要将 pb.Block 转换为 types.Block
		// 为了简化，我们假设已经转换或直接使用 pb.Block
		// 实际使用中可能需要添加转换逻辑

		// 检查是否达到切换高度
		if block.Header.Height >= um.currentProposal.SwitchHeight &&
			um.upgradeState.Phase == PhasePreexecuting {
			um.log.WithFields(logrus.Fields{
				"current_height": block.Header.Height,
				"switch_height":  um.currentProposal.SwitchHeight,
			}).Info("Reached switch height, initiating consensus switch")
			go um.executeSwitch()
		}
	} else {
		// 处理候选链区块
		um.log.WithFields(logrus.Fields{
			"height": block.Header.Height,
			"txs":    len(block.Txs),
		}).Trace("Processing candidate chain block")
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
	um.log.Info("Entering switch phase")
	um.upgradeState.Phase = PhaseSwitching
	um.mu.Unlock()

	um.log.Info("Executing consensus switch")

	err := um.switchManager.ExecuteSwitch()
	if err != nil {
		um.log.WithError(err).Error("Consensus switch execution failed")
		um.mu.Lock()
		um.upgradeState.Phase = PhaseFailed
		um.upgradeState.Failed = true
		um.upgradeState.FailureReason = fmt.Sprintf("Switch failed: %v", err)
		um.mu.Unlock()
		return
	}

	// 切换成功，处理缓存的消息
	um.log.Info("Consensus switch executed successfully, processing cached messages")
	cachedMessages := um.messageCache.OnSwitch()
	um.log.WithField("cached_count", len(cachedMessages)).Info("Retrieved cached messages")

	// 更新状态
	um.mu.Lock()
	um.upgradeState.Phase = PhaseCompleted
	um.upgradeState.Completed = true
	um.upgradeState.EndTime = time.Now()
	duration := um.upgradeState.EndTime.Sub(um.upgradeState.StartTime)
	um.mu.Unlock()

	um.log.WithFields(logrus.Fields{
		"duration_seconds": duration.Seconds(),
		"final_phase":      PhaseCompleted.String(),
	}).Info("Upgrade completed successfully")
}

// OnNetworkMessage 处理网络消息（核心入口）
// 当节点收到消息时，判断是否需要缓存或直接转发
func (um *UpgradeManager) OnNetworkMessage(msg interface{}, senderConsensusID, receiverConsensusID int64, epoch uint64) error {
	um.mu.RLock()
	defer um.mu.RUnlock()

	// 1) 判断是否为控制消息（升级相关消息）
	if um.isControlMessage(msg) {
		return um.handleControlMessage(msg)
	}

	// 2) 判断epoch：如果消息的epoch比当前epoch低，直接丢弃
	if epoch < um.currentEpoch {
		um.log.WithFields(logrus.Fields{
			"msg_epoch":     epoch,
			"current_epoch": um.currentEpoch,
		}).Debug("Dropping outdated message")
		return nil
	}

	// 3) 判断接收者共识是否已启动
	if um.isConsensusRunning(receiverConsensusID) {
		// 共识已启动，直接转发
		return um.forwardToConsensus(receiverConsensusID, msg)
	}

	// 4) 共识未启动：如果epoch比当前高，缓存消息
	if epoch >= um.currentEpoch {
		um.log.WithFields(logrus.Fields{
			"receiver_consensus": receiverConsensusID,
			"epoch":              epoch,
			"current_epoch":      um.currentEpoch,
		}).Info("Caching message for future epoch")

		// 写入内存缓存
		if err := um.messageCache.CacheMessage(msg, senderConsensusID, receiverConsensusID, epoch); err != nil {
			return fmt.Errorf("failed to cache message: %w", err)
		}

		// 写入持久化存储（如果需要，可以在MessageCache中实现）
		// 注意：UpgradePersistence接口不直接支持消息缓存持久化
		// 消息缓存的持久化应通过MessageCache自身管理
	}

	return nil
}

// OnCandidateConsensusStarted 当候选共识启动时调用
// 冲刷该共识的所有缓存消息
func (um *UpgradeManager) OnCandidateConsensusStarted(newConsensusID int64) error {
	um.mu.Lock()
	defer um.mu.Unlock()

	return um.onCandidateConsensusStartedNoLock(newConsensusID)
}

// onCandidateConsensusStartedNoLock 内部实现，不加锁
func (um *UpgradeManager) onCandidateConsensusStartedNoLock(newConsensusID int64) error {
	um.log.WithField("consensus_id", newConsensusID).Info("Candidate consensus started, flushing cached messages")

	// 从缓存中取出所有属于该共识的消息
	cached := um.messageCache.PopAllForConsensus(newConsensusID)

	// 逐个转发
	forwardCount := 0
	for _, msg := range cached {
		if err := um.forwardToConsensus(newConsensusID, msg.Message); err != nil {
			um.log.WithError(err).Warn("Failed to forward cached message")
			continue
		}
		forwardCount++

		// 从持久化存储中删除（如果需要）
		// 注意：消息缓存持久化应由MessageCache管理
	}

	um.log.WithField("forwarded_count", forwardCount).Info("Cached messages flushed to consensus")
	return nil
}

// OnConsensusSwitched 当共识切换完成时调用
// 清理旧epoch的消息
func (um *UpgradeManager) OnConsensusSwitched(oldEpoch uint64) error {
	um.mu.Lock()
	defer um.mu.Unlock()

	um.log.WithField("old_epoch", oldEpoch).Info("Consensus switched, cleaning old epoch messages")

	// 清理内存缓存
	cleaned := um.messageCache.DropByEpoch(oldEpoch)

	// 清理持久化存储（如果需要）
	// 注意：消息缓存持久化应由MessageCache管理

	um.log.WithField("cleaned_count", cleaned).Info("Old epoch messages cleaned")
	return nil
}

// SetCurrentEpoch 设置当前epoch
func (um *UpgradeManager) SetCurrentEpoch(epoch uint64) {
	um.mu.Lock()
	defer um.mu.Unlock()

	oldEpoch := um.currentEpoch
	um.currentEpoch = epoch

	um.log.WithFields(logrus.Fields{
		"old_epoch": oldEpoch,
		"new_epoch": epoch,
	}).Debug("Current epoch updated")

	// 处理缓存的消息：将属于当前epoch的消息取出并转发
	go um.processCachedMessagesForEpoch(epoch)
}

// processCachedMessagesForEpoch 处理缓存的指定epoch消息
func (um *UpgradeManager) processCachedMessagesForEpoch(epoch uint64) {
	messages := um.messageCache.GetMessagesByEpoch(epoch)

	if len(messages) == 0 {
		return
	}

	um.log.WithFields(logrus.Fields{
		"epoch": epoch,
		"count": len(messages),
	}).Info("Processing cached messages for current epoch")

	for _, msg := range messages {
		if err := um.forwardToConsensus(msg.ReceiverConsensusID, msg.Message); err != nil {
			um.log.WithError(err).Warn("Failed to forward cached message")
		}
	}
}

// isControlMessage 判断是否为控制消息
func (um *UpgradeManager) isControlMessage(msg interface{}) bool {
	// 检查是否为升级相关的pb消息
	switch m := msg.(type) {
	case *pb.Request:
		// 检查Request中是否包含升级配置交易
		if m.Tx != nil {
			if tx, err := RawTransaction(m.Tx).ToTx(); err == nil {
				// 检查交易类型是否为升级相关
				return isUpgradeTransaction(tx)
			}
		}
	case *pb.Block:
		// 检查区块中是否包含升级配置交易
		for _, tx := range m.Txs {
			if tx != nil && len(tx.Data) > 0 {
				if pbTx, err := RawTransaction(tx.Data).ToTx(); err == nil {
					if isUpgradeTransaction(pbTx) {
						return true
					}
				}
			}
		}
	}
	return false
}

// handleControlMessage 处理控制消息
func (um *UpgradeManager) handleControlMessage(msg interface{}) error {
	um.log.Debug("Handling control message")

	switch m := msg.(type) {
	case *pb.Request:
		// 处理升级配置交易
		if m.Tx != nil {
			if tx, err := RawTransaction(m.Tx).ToTx(); err == nil {
				return um.processUpgradeTransaction(tx)
			}
		}
	case *pb.Block:
		// 处理包含升级交易的区块
		for _, tx := range m.Txs {
			if tx != nil && len(tx.Data) > 0 {
				if pbTx, err := RawTransaction(tx.Data).ToTx(); err == nil {
					if err := um.processUpgradeTransaction(pbTx); err != nil {
						um.log.WithError(err).Warn("Failed to process upgrade transaction")
					}
				}
			}
		}
	}

	return nil
}

// isConsensusRunning 判断共识是否已启动
func (um *UpgradeManager) isConsensusRunning(consensusID int64) bool {
	// 检查是否为当前主链共识
	if um.currentConsensus != nil && um.currentConsensus.GetConsensusID() == consensusID {
		return true
	}

	// 检查是否为已启动的候选共识
	if um.multiChainManager != nil {
		// 获取所有候选链
		candidates := um.multiChainManager.ListCandidateChains()
		for _, candidateID := range candidates {
			candidate := um.multiChainManager.GetCandidateConsensus(candidateID)
			if candidate != nil && candidate.GetConsensusID() == consensusID {
				return true
			}
		}
	}

	return false
}

// forwardToConsensus 转发消息到指定共识
func (um *UpgradeManager) forwardToConsensus(consensusID int64, msg interface{}) error {
	um.log.WithFields(logrus.Fields{
		"consensus_id": consensusID,
	}).Debug("Forwarding message to consensus")

	var targetConsensus model.Consensus

	// 查找目标共识实例
	if um.currentConsensus != nil && um.currentConsensus.GetConsensusID() == consensusID {
		targetConsensus = um.currentConsensus
	} else if um.multiChainManager != nil {
		// 在候选链中查找
		candidates := um.multiChainManager.ListCandidateChains()
		for _, candidateID := range candidates {
			candidate := um.multiChainManager.GetCandidateConsensus(candidateID)
			if candidate != nil && candidate.GetConsensusID() == consensusID {
				targetConsensus = candidate
				break
			}
		}
	}

	if targetConsensus == nil {
		return fmt.Errorf("consensus %d not found", consensusID)
	}

	// 根据消息类型转发
	switch m := msg.(type) {
	case []byte:
		// 字节消息，使用MsgByteEntrance
		select {
		case targetConsensus.GetMsgByteEntrance() <- m:
			return nil
		default:
			return fmt.Errorf("consensus message channel is full")
		}
	case *pb.Request:
		// Request消息
		select {
		case targetConsensus.GetRequestEntrance() <- m:
			return nil
		default:
			return fmt.Errorf("consensus request channel is full")
		}
	default:
		// 尝试转换为字节后发送
		if msgBytes, ok := msg.([]byte); ok {
			select {
			case targetConsensus.GetMsgByteEntrance() <- msgBytes:
				return nil
			default:
				return fmt.Errorf("consensus message channel is full")
			}
		}
		return fmt.Errorf("unsupported message type: %T", msg)
	}
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
	err := um.persistence.SaveState(um.upgradeState)
	if err != nil {
		um.log.WithError(err).Error("Failed to persist state")
	}
	return err
}

// persistProposal 持久化提案
func (um *UpgradeManager) persistProposal(proposal *UpgradeProposal) error {
	if um.persistence == nil {
		return nil
	}
	err := um.persistence.SaveProposal(proposal)
	if err != nil {
		um.log.WithError(err).Error("Failed to persist proposal")
	}
	return err
}

// GetCurrentProposal 获取当前提案
func (um *UpgradeManager) GetCurrentProposal() *UpgradeProposal {
	um.mu.RLock()
	defer um.mu.RUnlock()
	return um.currentProposal
}

// GetMessageCache 获取消息缓存（用于测试）
func (um *UpgradeManager) GetMessageCache() *MessageCache {
	um.mu.RLock()
	defer um.mu.RUnlock()
	return um.messageCache
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
	multiChainStorage storage.MultiChainStorage,
	persistence UpgradePersistence,
	nid int64,
	executor interface{},
	p2pAdaptor interface{},
	log *logrus.Entry,
) (*UpgradeManager, error) {
	um, err := NewUpgradeManagerWithPersistence(
		currentConsensus,
		config,
		multiChainStorage,
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

// RawTransaction 类型别名，用于消息处理
type RawTransaction []byte

// ToTx 将原始交易转换为pb.Transaction
func (rtx RawTransaction) ToTx() (*pb.Transaction, error) {
	tx := new(pb.Transaction)
	err := proto.Unmarshal([]byte(rtx), tx)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

// isUpgradeTransaction 判断交易是否为升级配置交易
func isUpgradeTransaction(tx *pb.Transaction) bool {
	// 检查交易Payload字段，判断是否包含升级配置
	// 这里使用简单的前缀匹配
	if len(tx.Payload) > 0 {
		// 假设升级交易以特定前缀标识
		prefix := []byte("UPGRADE:")
		if len(tx.Payload) >= len(prefix) {
			for i := range prefix {
				if tx.Payload[i] != prefix[i] {
					return false
				}
			}
			return true
		}
	}
	return false
}

// processUpgradeTransaction 处理升级配置交易
func (um *UpgradeManager) processUpgradeTransaction(tx *pb.Transaction) error {
	um.log.WithField("tx_payload_len", len(tx.Payload)).Info("Processing upgrade transaction")

	// 解析升级配置
	// 实际实现中需要反序列化tx.Payload获取UpgradeProposal
	// 这里简化处理

	// 验证交易签名
	// if !um.verifyUpgradeTransactionSignature(tx) {
	//     return fmt.Errorf("invalid upgrade transaction signature")
	// }

	// 记录事件
	um.recordEvent(EventUpgradeStarted, fmt.Sprintf("Received upgrade transaction: %d bytes", len(tx.Payload)))

	return nil
}
