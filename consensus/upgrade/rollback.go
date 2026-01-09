package upgrade

import (
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/internal/storage"
)

// RollbackManager 回退管理器
// 负责在预执行出现问题时回退到原共识
type RollbackManager struct {
	candidateID       string // 候选链ID
	multiChainManager *MultiChainManager
	preexecMonitor    *PreexecMonitor
	storage           storage.UpgradeStorage
	log               *logrus.Entry

	// 回退状态
	rolledBack     bool
	rollbackTime   time.Time
	rollbackReason string

	// 通道
	rollbackDoneChan chan struct{}

	mu sync.RWMutex
}

// NewRollbackManager 创建回退管理器
func NewRollbackManager(
	candidateID string,
	multiChainManager *MultiChainManager,
	preexecMonitor *PreexecMonitor,
	log *logrus.Entry,
) *RollbackManager {
	return NewRollbackManagerWithStorage(candidateID, multiChainManager, preexecMonitor, nil, log)
}

// NewRollbackManagerWithStorage 创建带存储的回退管理器
func NewRollbackManagerWithStorage(
	candidateID string,
	multiChainManager *MultiChainManager,
	preexecMonitor *PreexecMonitor,
	upgradeStorage storage.UpgradeStorage,
	log *logrus.Entry,
) *RollbackManager {
	if log == nil {
		log = logrus.NewEntry(logrus.New())
	}

	return &RollbackManager{
		candidateID:       candidateID,
		multiChainManager: multiChainManager,
		preexecMonitor:    preexecMonitor,
		storage:           upgradeStorage,
		log:               log,
		rollbackDoneChan:  make(chan struct{}, 1),
	}
}

// ExecuteRollback 执行回退
func (rm *RollbackManager) ExecuteRollback(reason string) error {
	rm.mu.Lock()
	if rm.rolledBack {
		rm.mu.Unlock()
		return fmt.Errorf("already rolled back")
	}
	rm.mu.Unlock()

	rm.log.WithField("reason", reason).Warn("Starting rollback execution")

	// 1. 停止预执行监控
	if rm.preexecMonitor != nil {
		rm.log.Debug("Stopping preexec monitor")
		rm.preexecMonitor.Stop()
	}

	// 2. 执行候选链回退
	rm.log.WithField("candidate_id", rm.candidateID).Info("Rolling back candidate chain")
	err := rm.multiChainManager.RollbackCandidateChain(rm.candidateID)
	if err != nil {
		return fmt.Errorf("failed to rollback preexec chain: %w", err)
	}

	// 3. 标记回退完成
	rm.mu.Lock()
	rm.rolledBack = true
	rm.rollbackTime = time.Now()
	rm.rollbackReason = reason
	rm.mu.Unlock()

	// 4. 通知回退完成
	select {
	case rm.rollbackDoneChan <- struct{}{}:
	default:
	}

	rm.log.WithFields(logrus.Fields{
		"reason":        reason,
		"rollback_time": rm.rollbackTime,
	}).Info("Rollback completed successfully")

	return nil
}

// MonitorAndRollback 监控并自动回退
// 在单独的 goroutine 中运行，监听回退信号
func (rm *RollbackManager) MonitorAndRollback() {
	rm.log.Info("Starting automatic rollback monitoring")

	if rm.preexecMonitor == nil {
		rm.log.Error("Preexec monitor not initialized, cannot monitor")
		return
	}

	// 定期检查是否需要回退
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if shouldRollback, reason := rm.preexecMonitor.ShouldRollback(); shouldRollback {
				rm.log.WithField("reason", reason).Warn("Rollback condition met")
				err := rm.ExecuteRollback(fmt.Sprintf("Performance degradation: %s", reason))
				if err != nil {
					rm.log.WithError(err).Error("Failed to execute automatic rollback")
				}
				return
			}
		}
	}
}

// WaitForRollback 等待回退完成
func (rm *RollbackManager) WaitForRollback(timeout time.Duration) error {
	select {
	case <-rm.rollbackDoneChan:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for rollback completion")
	}
}

// IsRolledBack 检查是否已回退
func (rm *RollbackManager) IsRolledBack() bool {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.rolledBack
}

// GetRollbackReason 获取回退原因
func (rm *RollbackManager) GetRollbackReason() string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.rollbackReason
}

// GetRollbackTime 获取回退时间
func (rm *RollbackManager) GetRollbackTime() time.Time {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.rollbackTime
}

// Reset 重置回退管理器（用于测试）
func (rm *RollbackManager) Reset() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.rolledBack = false
	rm.rollbackTime = time.Time{}
	rm.rollbackReason = ""

	// 清空通道
	select {
	case <-rm.rollbackDoneChan:
	default:
	}
}

// ValidateRollback 验证回退条件
func (rm *RollbackManager) ValidateRollback() error {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if rm.rolledBack {
		return fmt.Errorf("already rolled back")
	}

	// 检查候选链是否激活
	if !rm.multiChainManager.IsCandidateActive(rm.candidateID) {
		return fmt.Errorf("candidate chain not active, nothing to rollback")
	}

	return nil
}

// GetRollbackStatus 获取回退状态摘要
func (rm *RollbackManager) GetRollbackStatus() *RollbackStatus {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return &RollbackStatus{
		RolledBack:     rm.rolledBack,
		RollbackTime:   rm.rollbackTime,
		RollbackReason: rm.rollbackReason,
		PreexecActive:  rm.multiChainManager.IsCandidateActive(rm.candidateID),
	}
}

// RollbackStatus 回退状态
type RollbackStatus struct {
	RolledBack     bool
	RollbackTime   time.Time
	RollbackReason string
	PreexecActive  bool
}

// ForceRollback 强制回退（紧急情况使用）
func (rm *RollbackManager) ForceRollback(reason string) error {
	rm.log.WithField("reason", reason).Warn("Forcing immediate rollback")

	// 忽略已回退状态，强制执行
	rm.mu.Lock()
	if rm.rolledBack {
		rm.mu.Unlock()
		rm.log.Warn("Already rolled back, resetting state")
		rm.Reset()
	} else {
		rm.mu.Unlock()
	}

	return rm.ExecuteRollback(reason)
}

// CanRollback 检查是否可以回退
func (rm *RollbackManager) CanRollback() bool {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	// 未回退且候选链激活
	return !rm.rolledBack && rm.multiChainManager.IsCandidateActive(rm.candidateID)
}

// RollbackCheckpoint 回退检查点
// 用于保存回退前的状态
type RollbackCheckpoint struct {
	Height        uint64
	ConsensusType string
	Timestamp     time.Time
	StateHash     []byte
}

// CreateCheckpoint 创建回退检查点
func (rm *RollbackManager) CreateCheckpoint(height uint64, consensusType string, stateHash []byte) *RollbackCheckpoint {
	checkpoint := &RollbackCheckpoint{
		Height:        height,
		ConsensusType: consensusType,
		Timestamp:     time.Now(),
		StateHash:     stateHash,
	}

	// 如果配置了存储，持久化检查点
	if rm.storage != nil {
		storageCheckpoint := &storage.RollbackCheckpoint{
			Height:        height,
			ConsensusType: consensusType,
			Timestamp:     checkpoint.Timestamp,
			StateHash:     stateHash,
			BlockHash:     nil, // TODO: 从区块链获取
			Description:   "Auto-created checkpoint",
		}
		if err := rm.storage.StoreCheckpoint(storageCheckpoint); err != nil {
			rm.log.WithError(err).Error("Failed to persist checkpoint")
		} else {
			rm.log.WithField("height", height).Info("Checkpoint persisted")
		}
	}

	rm.log.WithFields(logrus.Fields{
		"height":         height,
		"consensus_type": consensusType,
	}).Info("Created rollback checkpoint")

	return checkpoint
}

// ExecuteRollbackWithCheckpoint 执行带检查点的回退
// 回退到指定的检查点状态
func (rm *RollbackManager) ExecuteRollbackWithCheckpoint(reason string, checkpoint *RollbackCheckpoint) error {
	rm.mu.Lock()
	if rm.rolledBack {
		rm.mu.Unlock()
		return fmt.Errorf("already rolled back")
	}
	rm.mu.Unlock()

	rm.log.WithFields(logrus.Fields{
		"reason":     reason,
		"checkpoint": checkpoint.Height,
	}).Warn("Starting rollback with checkpoint")

	// 1. 验证检查点
	if checkpoint == nil {
		return fmt.Errorf("checkpoint is nil")
	}

	// 2. 停止预执行监控
	if rm.preexecMonitor != nil {
		rm.preexecMonitor.Stop()
	}

	// 3. 执行候选链回退
	err := rm.multiChainManager.RollbackCandidateChain(rm.candidateID)
	if err != nil {
		return fmt.Errorf("failed to rollback preexec chain: %w", err)
	}

	// 4. 标记回退完成
	rm.mu.Lock()
	rm.rolledBack = true
	rm.rollbackTime = time.Now()
	rm.rollbackReason = fmt.Sprintf("%s (checkpoint: %d)", reason, checkpoint.Height)
	rm.mu.Unlock()

	// 5. 通知回退完成
	select {
	case rm.rollbackDoneChan <- struct{}{}:
	default:
	}

	rm.log.Info("Rollback with checkpoint completed successfully")
	return nil
}

// ShouldCreateCheckpoint 判断是否应该创建检查点
// 在关键高度或状态变化时返回 true
func (rm *RollbackManager) ShouldCreateCheckpoint(height uint64, proposalHeight uint64) bool {
	// 在预执行开始前创建检查点
	if height == proposalHeight-1 {
		return true
	}

	// 每 100 个区块创建一个检查点
	if height%100 == 0 {
		return true
	}

	return false
}

// LoadCheckpoint 从持久化存储加载检查点
func (rm *RollbackManager) LoadCheckpoint(height uint64) (*RollbackCheckpoint, error) {
	if rm.storage == nil {
		return nil, fmt.Errorf("storage not configured")
	}

	storageCP, err := rm.storage.GetCheckpoint(height)
	if err != nil {
		return nil, fmt.Errorf("failed to load checkpoint: %w", err)
	}

	return &RollbackCheckpoint{
		Height:        storageCP.Height,
		ConsensusType: storageCP.ConsensusType,
		Timestamp:     storageCP.Timestamp,
		StateHash:     storageCP.StateHash,
	}, nil
}

// LoadLatestCheckpoint 加载最新检查点
func (rm *RollbackManager) LoadLatestCheckpoint() (*RollbackCheckpoint, error) {
	if rm.storage == nil {
		return nil, fmt.Errorf("storage not configured")
	}

	storageCP, err := rm.storage.GetLatestCheckpoint()
	if err != nil {
		return nil, fmt.Errorf("failed to load latest checkpoint: %w", err)
	}

	return &RollbackCheckpoint{
		Height:        storageCP.Height,
		ConsensusType: storageCP.ConsensusType,
		Timestamp:     storageCP.Timestamp,
		StateHash:     storageCP.StateHash,
	}, nil
}

// ListCheckpoints 列出最近的检查点
func (rm *RollbackManager) ListCheckpoints(limit int) ([]*RollbackCheckpoint, error) {
	if rm.storage == nil {
		return nil, fmt.Errorf("storage not configured")
	}

	storageCPs, err := rm.storage.ListCheckpoints(limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list checkpoints: %w", err)
	}

	checkpoints := make([]*RollbackCheckpoint, len(storageCPs))
	for i, scp := range storageCPs {
		checkpoints[i] = &RollbackCheckpoint{
			Height:        scp.Height,
			ConsensusType: scp.ConsensusType,
			Timestamp:     scp.Timestamp,
			StateHash:     scp.StateHash,
		}
	}

	return checkpoints, nil
}
