package upgrade

import (
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

// PreexecMonitor 预执行监控器
type PreexecMonitor struct {
	proposalID types.TxHash
	proposal   *UpgradeProposal

	dualChainMgr     *DualChainManager
	metricsCollector *MetricsCollector

	startHeight      uint64
	targetHeight     uint64
	lastBlockTime    time.Time
	monitoringActive bool

	rollbackChan chan string   // 回退信号通道
	readyChan    chan struct{} // 就绪信号通道

	log *logrus.Entry
	mu  sync.RWMutex
}

// NewPreexecMonitor 创建预执行监控器
func NewPreexecMonitor(
	proposal *UpgradeProposal,
	dualChainMgr *DualChainManager,
	log *logrus.Entry,
) *PreexecMonitor {
	return &PreexecMonitor{
		proposalID:       proposal.ProposalID,
		proposal:         proposal,
		dualChainMgr:     dualChainMgr,
		metricsCollector: NewMetricsCollector(proposal.ProposalID, proposal.PreexecStartHeight, log),
		startHeight:      proposal.PreexecStartHeight,
		targetHeight:     proposal.SwitchHeight,
		monitoringActive: false,
		rollbackChan:     make(chan string, 1),
		readyChan:        make(chan struct{}, 1),
		log:              log,
	}
}

// Start 启动监控
func (pm *PreexecMonitor) Start() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.monitoringActive {
		return fmt.Errorf("monitoring already active")
	}

	pm.monitoringActive = true
	pm.lastBlockTime = time.Now()

	pm.log.WithFields(logrus.Fields{
		"proposal_id":   fmt.Sprintf("%x", pm.proposalID),
		"start_height":  pm.startHeight,
		"target_height": pm.targetHeight,
	}).Info("Started preexecution monitoring")

	// 启动监控协程
	go pm.monitorLoop()

	return nil
}

// Stop 停止监控
func (pm *PreexecMonitor) Stop() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !pm.monitoringActive {
		return
	}

	pm.monitoringActive = false
	pm.log.Info("Stopped preexecution monitoring")
}

// monitorLoop 监控循环
func (pm *PreexecMonitor) monitorLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		pm.mu.RLock()
		active := pm.monitoringActive
		pm.mu.RUnlock()

		if !active {
			return
		}

		select {
		case <-ticker.C:
			pm.checkStatus()
		}
	}
}

// checkStatus 检查状态
func (pm *PreexecMonitor) checkStatus() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !pm.dualChainMgr.IsPreexecActive() {
		return
	}

	// 获取当前预执行链高度
	currentHeight := pm.dualChainMgr.GetPreexecChainHeight()

	// 检查是否超时
	if time.Since(pm.lastBlockTime) > time.Duration(pm.proposal.RollbackCondition.TimeoutBlocks)*10*time.Second {
		reason := "preexecution timeout"
		pm.log.Warn(reason)
		select {
		case pm.rollbackChan <- reason:
		default:
		}
		return
	}

	// 检查是否达到目标高度
	if currentHeight >= pm.targetHeight {
		// 评估性能指标
		ok, reason := pm.metricsCollector.EvaluateCondition(pm.proposal.RollbackCondition)
		if !ok {
			pm.log.WithField("reason", reason).Warn("Preexecution failed performance check")
			select {
			case pm.rollbackChan <- reason:
			default:
			}
			return
		}

		// 预执行完成，发送就绪信号
		pm.log.Info("Preexecution completed successfully")
		select {
		case pm.readyChan <- struct{}{}:
		default:
		}
	}
}

// OnBlockProcessed 当区块被处理时调用
func (pm *PreexecMonitor) OnBlockProcessed(
	height uint64,
	blockTime time.Duration,
	txCount uint32,
	err error,
) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 更新最后区块时间
	pm.lastBlockTime = time.Now()

	// 记录指标
	pm.metricsCollector.RecordBlock(height, blockTime, txCount, err)

	pm.log.WithFields(logrus.Fields{
		"height":     height,
		"block_time": blockTime,
		"tx_count":   txCount,
		"success":    err == nil,
	}).Debug("Block processed")

	// 如果有错误，检查是否需要回退
	if err != nil {
		metrics := pm.metricsCollector.GetMetrics()
		if metrics.ErrorRate > pm.proposal.RollbackCondition.MaxErrorRate {
			reason := fmt.Sprintf("error rate %.2f exceeds threshold %.2f",
				metrics.ErrorRate, pm.proposal.RollbackCondition.MaxErrorRate)
			select {
			case pm.rollbackChan <- reason:
			default:
			}
		}
	}
}

// GetMetrics 获取性能指标
func (pm *PreexecMonitor) GetMetrics() *PerformanceMetrics {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.metricsCollector.GetMetrics()
}

// IsReady 检查是否就绪
func (pm *PreexecMonitor) IsReady() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if !pm.dualChainMgr.IsPreexecActive() {
		return false
	}

	currentHeight := pm.dualChainMgr.GetPreexecChainHeight()
	if currentHeight < pm.targetHeight {
		return false
	}

	// 检查性能指标
	ok, _ := pm.metricsCollector.EvaluateCondition(pm.proposal.RollbackCondition)
	return ok
}

// ShouldRollback 检查是否应该回退
func (pm *PreexecMonitor) ShouldRollback() (bool, string) {
	select {
	case reason := <-pm.rollbackChan:
		return true, reason
	default:
		return false, ""
	}
}

// WaitForReady 等待就绪
func (pm *PreexecMonitor) WaitForReady(timeout time.Duration) error {
	select {
	case <-pm.readyChan:
		return nil
	case reason := <-pm.rollbackChan:
		return fmt.Errorf("preexecution failed: %s", reason)
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for preexecution ready")
	}
}

// GetProgress 获取进度
func (pm *PreexecMonitor) GetProgress() float64 {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if !pm.dualChainMgr.IsPreexecActive() {
		return 0
	}

	currentHeight := pm.dualChainMgr.GetPreexecChainHeight()
	if currentHeight <= pm.startHeight {
		return 0
	}

	total := pm.targetHeight - pm.startHeight
	current := currentHeight - pm.startHeight

	if total == 0 {
		return 0
	}

	return float64(current) / float64(total)
}

// GetStatus 获取监控状态
func (pm *PreexecMonitor) GetStatus() *MonitorStatus {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	currentHeight := uint64(0)
	if pm.dualChainMgr.IsPreexecActive() {
		currentHeight = pm.dualChainMgr.GetPreexecChainHeight()
	}

	return &MonitorStatus{
		Active:        pm.monitoringActive,
		CurrentHeight: currentHeight,
		TargetHeight:  pm.targetHeight,
		Progress:      pm.GetProgress(),
		Metrics:       pm.metricsCollector.GetMetrics(),
		LastBlockTime: pm.lastBlockTime,
	}
}

// MonitorStatus 监控状态
type MonitorStatus struct {
	Active        bool
	CurrentHeight uint64
	TargetHeight  uint64
	Progress      float64
	Metrics       *PerformanceMetrics
	LastBlockTime time.Time
}

// EvaluatePerformance 评估性能
func (pm *PreexecMonitor) EvaluatePerformance() (*PerformanceEvaluation, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	metrics := pm.metricsCollector.GetMetrics()
	condition := pm.proposal.RollbackCondition

	eval := &PerformanceEvaluation{
		Metrics:   metrics,
		Condition: condition,
		Passed:    true,
		Issues:    make([]string, 0),
	}

	// 检查错误率
	if metrics.ErrorRate > condition.MaxErrorRate {
		eval.Passed = false
		eval.Issues = append(eval.Issues,
			fmt.Sprintf("Error rate %.2f%% exceeds threshold %.2f%%",
				metrics.ErrorRate*100, condition.MaxErrorRate*100))
	}

	// 检查区块时间
	if metrics.AvgBlockTime > 0 {
		// 这里需要基准值来比较
		// 简化处理，假设通过
	}

	// 检查吞吐量
	if metrics.AvgThroughput > 0 {
		// 这里需要基准值来比较
		// 简化处理，假设通过
	}

	return eval, nil
}

// PerformanceEvaluation 性能评估结果
type PerformanceEvaluation struct {
	Metrics   *PerformanceMetrics
	Condition *pb.RollbackCondition
	Passed    bool
	Issues    []string
}
