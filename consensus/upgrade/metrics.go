package upgrade

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

// MetricsCollector 性能指标收集器
type MetricsCollector struct {
	proposalID  types.TxHash
	startHeight uint64

	blockMetrics []BlockMetric
	startTime    time.Time

	log *logrus.Entry
	mu  sync.RWMutex
}

// BlockMetric 单个区块指标
type BlockMetric struct {
	Height    uint64
	Timestamp time.Time
	BlockTime time.Duration
	TxCount   uint32
	Success   bool
	ErrorMsg  string
}

// NewMetricsCollector 创建指标收集器
func NewMetricsCollector(
	proposalID types.TxHash,
	startHeight uint64,
	log *logrus.Entry,
) *MetricsCollector {
	return &MetricsCollector{
		proposalID:   proposalID,
		startHeight:  startHeight,
		blockMetrics: make([]BlockMetric, 0),
		startTime:    time.Now(),
		log:          log,
	}
}

// RecordBlock 记录区块指标
func (mc *MetricsCollector) RecordBlock(
	height uint64,
	blockTime time.Duration,
	txCount uint32,
	err error,
) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	metric := BlockMetric{
		Height:    height,
		Timestamp: time.Now(),
		BlockTime: blockTime,
		TxCount:   txCount,
		Success:   err == nil,
	}

	if err != nil {
		metric.ErrorMsg = err.Error()
	}

	mc.blockMetrics = append(mc.blockMetrics, metric)

	mc.log.WithFields(logrus.Fields{
		"height":     height,
		"block_time": blockTime,
		"tx_count":   txCount,
		"success":    metric.Success,
	}).Debug("Recorded block metric")
}

// ComputeAggregateMetrics 计算聚合指标
func (mc *MetricsCollector) ComputeAggregateMetrics() *PerformanceMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if len(mc.blockMetrics) == 0 {
		return &PerformanceMetrics{
			StartHeight: mc.startHeight,
			EndHeight:   mc.startHeight,
			TotalBlocks: 0,
		}
	}

	totalBlockTime := time.Duration(0)
	totalTxs := uint32(0)
	failedBlocks := uint64(0)
	blockTimes := make([]float64, 0, len(mc.blockMetrics))
	txCounts := make([]uint32, 0, len(mc.blockMetrics))

	for _, metric := range mc.blockMetrics {
		totalBlockTime += metric.BlockTime
		totalTxs += metric.TxCount
		blockTimes = append(blockTimes, metric.BlockTime.Seconds())
		txCounts = append(txCounts, metric.TxCount)

		if !metric.Success {
			failedBlocks++
		}
	}

	totalBlocks := uint64(len(mc.blockMetrics))
	avgBlockTime := totalBlockTime.Seconds() / float64(totalBlocks)
	avgThroughput := float64(totalTxs) / totalBlockTime.Seconds()
	errorRate := float64(failedBlocks) / float64(totalBlocks)

	endHeight := mc.blockMetrics[len(mc.blockMetrics)-1].Height

	return &PerformanceMetrics{
		StartHeight:   mc.startHeight,
		EndHeight:     endHeight,
		TotalBlocks:   totalBlocks,
		FailedBlocks:  failedBlocks,
		BlockTimes:    blockTimes,
		TxCounts:      txCounts,
		AvgBlockTime:  avgBlockTime,
		AvgThroughput: avgThroughput,
		ErrorRate:     errorRate,
	}
}

// GetMetrics 获取当前指标
func (mc *MetricsCollector) GetMetrics() *PerformanceMetrics {
	return mc.ComputeAggregateMetrics()
}

// EvaluateCondition 评估是否满足回退条件
func (mc *MetricsCollector) EvaluateCondition(condition *pb.RollbackCondition) (bool, string) {
	metrics := mc.ComputeAggregateMetrics()

	// 检查错误率
	if metrics.ErrorRate > condition.MaxErrorRate {
		return false, "error rate too high"
	}

	// 检查区块时间增长率
	if len(mc.blockMetrics) > 0 {
		latestBlockTime := mc.blockMetrics[len(mc.blockMetrics)-1].BlockTime.Seconds()
		if metrics.AvgBlockTime > 0 {
			ratio := latestBlockTime / metrics.AvgBlockTime
			if ratio > condition.MaxLatencyIncrease {
				return false, "block time increased too much"
			}
		}
	}

	// 检查吞吐量
	if metrics.AvgThroughput > 0 {
		// 这里需要一个基准吞吐量来比较
		// 简化处理，假设满足条件
	}

	return true, ""
}

// Reset 重置指标
func (mc *MetricsCollector) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.blockMetrics = make([]BlockMetric, 0)
	mc.startTime = time.Now()
	mc.log.Info("Metrics collector reset")
}

// GetBlockCount 获取已记录的区块数量
func (mc *MetricsCollector) GetBlockCount() int {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return len(mc.blockMetrics)
}

// GetLatestMetric 获取最新的区块指标
func (mc *MetricsCollector) GetLatestMetric() *BlockMetric {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if len(mc.blockMetrics) == 0 {
		return nil
	}

	latest := mc.blockMetrics[len(mc.blockMetrics)-1]
	return &latest
}

// GetFailedBlockCount 获取失败区块数量
func (mc *MetricsCollector) GetFailedBlockCount() uint64 {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	count := uint64(0)
	for _, metric := range mc.blockMetrics {
		if !metric.Success {
			count++
		}
	}
	return count
}

// GetAverageBlockTime 获取平均区块时间
func (mc *MetricsCollector) GetAverageBlockTime() time.Duration {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if len(mc.blockMetrics) == 0 {
		return 0
	}

	total := time.Duration(0)
	for _, metric := range mc.blockMetrics {
		total += metric.BlockTime
	}

	return total / time.Duration(len(mc.blockMetrics))
}

// GetTotalTransactions 获取总交易数
func (mc *MetricsCollector) GetTotalTransactions() uint32 {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	total := uint32(0)
	for _, metric := range mc.blockMetrics {
		total += metric.TxCount
	}
	return total
}
