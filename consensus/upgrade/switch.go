package upgrade

import (
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/consensus/model"
)

// SwitchManager 切换管理器
// 负责管理从旧共识到新共识的平滑切换过程
type SwitchManager struct {
	dualChainManager *DualChainManager
	preexecMonitor   *PreexecMonitor
	log              *logrus.Entry

	// 切换状态
	switchHeight uint64
	switched     bool
	switching    bool
	switchTime   time.Time

	// 通道
	switchReadyChan chan struct{}
	switchDoneChan  chan struct{}

	mu sync.RWMutex
}

// NewSwitchManager 创建切换管理器
func NewSwitchManager(
	dualChainManager *DualChainManager,
	preexecMonitor *PreexecMonitor,
	log *logrus.Entry,
) *SwitchManager {
	if log == nil {
		log = logrus.NewEntry(logrus.New())
	}

	return &SwitchManager{
		dualChainManager: dualChainManager,
		preexecMonitor:   preexecMonitor,
		log:              log,
		switchReadyChan:  make(chan struct{}, 1),
		switchDoneChan:   make(chan struct{}, 1),
	}
}

// PrepareSwitch 准备切换
// 在达到切换高度前准备切换条件
func (sm *SwitchManager) PrepareSwitch(switchHeight uint64) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.switched {
		return fmt.Errorf("already switched")
	}
	if sm.switching {
		return fmt.Errorf("switch already in progress")
	}

	sm.switchHeight = switchHeight
	sm.log.WithField("switch_height", switchHeight).Info("Preparing consensus switch")

	return nil
}

// ExecuteSwitch 执行切换
// 在达到切换高度时执行实际的共识切换
func (sm *SwitchManager) ExecuteSwitch() error {
	sm.mu.Lock()
	if sm.switched {
		sm.mu.Unlock()
		return fmt.Errorf("already switched")
	}
	if sm.switching {
		sm.mu.Unlock()
		return fmt.Errorf("switch already in progress")
	}
	sm.switching = true
	sm.mu.Unlock()

	sm.log.Info("Starting consensus switch execution")

	// 1. 等待预执行监控器就绪
	sm.log.Debug("Waiting for preexec monitor ready signal")
	err := sm.preexecMonitor.WaitForReady(30 * time.Second)
	if err != nil {
		sm.mu.Lock()
		sm.switching = false
		sm.mu.Unlock()
		return fmt.Errorf("preexec monitor not ready: %w", err)
	}
	sm.log.Info("Preexec monitor ready, proceeding with switch")

	// 2. 检查是否需要回退
	if shouldRollback, reason := sm.preexecMonitor.ShouldRollback(); shouldRollback {
		sm.mu.Lock()
		sm.switching = false
		sm.mu.Unlock()
		return fmt.Errorf("rollback signal received: %s", reason)
	}

	// 3. 执行双链合并
	sm.log.Info("Merging preexecution chain into main chain")
	mergeErr := sm.dualChainManager.MergePreexecChain(sm.switchHeight)
	if mergeErr != nil {
		sm.mu.Lock()
		sm.switching = false
		sm.mu.Unlock()
		return fmt.Errorf("failed to merge preexec chain: %w", err)
	}

	// 4. 标记切换完成
	sm.mu.Lock()
	sm.switched = true
	sm.switching = false
	sm.switchTime = time.Now()
	sm.mu.Unlock()

	// 5. 通知切换完成
	select {
	case sm.switchDoneChan <- struct{}{}:
	default:
	}

	sm.log.WithFields(logrus.Fields{
		"switch_height": sm.switchHeight,
		"switch_time":   sm.switchTime,
	}).Info("Consensus switch completed successfully")

	return nil
}

// WaitForSwitch 等待切换完成
func (sm *SwitchManager) WaitForSwitch(timeout time.Duration) error {
	select {
	case <-sm.switchDoneChan:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for switch completion")
	}
}

// IsSwitched 检查是否已切换
func (sm *SwitchManager) IsSwitched() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.switched
}

// IsSwitching 检查是否正在切换
func (sm *SwitchManager) IsSwitching() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.switching
}

// GetSwitchHeight 获取切换高度
func (sm *SwitchManager) GetSwitchHeight() uint64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.switchHeight
}

// GetSwitchTime 获取切换时间
func (sm *SwitchManager) GetSwitchTime() time.Time {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.switchTime
}

// GetNewConsensus 获取新共识实例
func (sm *SwitchManager) GetNewConsensus() model.Consensus {
	return sm.dualChainManager.GetPreexecConsensus()
}

// GetOldConsensus 获取旧共识实例
func (sm *SwitchManager) GetOldConsensus() model.Consensus {
	return sm.dualChainManager.GetMainConsensus()
}

// Reset 重置切换管理器（用于测试）
func (sm *SwitchManager) Reset() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.switched = false
	sm.switching = false
	sm.switchHeight = 0
	sm.switchTime = time.Time{}

	// 清空通道
	select {
	case <-sm.switchReadyChan:
	default:
	}
	select {
	case <-sm.switchDoneChan:
	default:
	}
}

// ValidateSwitch 验证切换条件
func (sm *SwitchManager) ValidateSwitch() error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.switched {
		return fmt.Errorf("already switched")
	}

	// 检查预执行链是否激活
	if !sm.dualChainManager.IsPreexecActive() {
		return fmt.Errorf("preexec chain not active")
	}

	// 检查监控器状态
	if sm.preexecMonitor == nil {
		return fmt.Errorf("preexec monitor not initialized")
	}

	return nil
}

// GetSwitchStatus 获取切换状态摘要
func (sm *SwitchManager) GetSwitchStatus() *SwitchStatus {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return &SwitchStatus{
		SwitchHeight:  sm.switchHeight,
		Switched:      sm.switched,
		Switching:     sm.switching,
		SwitchTime:    sm.switchTime,
		PreexecActive: sm.dualChainManager.IsPreexecActive(),
	}
}

// SwitchStatus 切换状态
type SwitchStatus struct {
	SwitchHeight  uint64
	Switched      bool
	Switching     bool
	SwitchTime    time.Time
	PreexecActive bool
}
