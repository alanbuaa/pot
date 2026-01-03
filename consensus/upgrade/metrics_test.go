package upgrade

import (
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

func TestNewMetricsCollector(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	proposalID := types.TxHash{1, 2, 3, 4}
	startHeight := uint64(100)

	collector := NewMetricsCollector(proposalID, startHeight, log)

	if collector == nil {
		t.Fatal("NewMetricsCollector returned nil")
	}

	if collector.proposalID != proposalID {
		t.Errorf("proposalID = %v, want %v", collector.proposalID, proposalID)
	}

	if collector.startHeight != startHeight {
		t.Errorf("startHeight = %d, want %d", collector.startHeight, startHeight)
	}

	if collector.GetBlockCount() != 0 {
		t.Errorf("initial block count = %d, want 0", collector.GetBlockCount())
	}
}

func TestMetricsCollector_RecordBlock(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	collector := NewMetricsCollector(types.TxHash{}, 100, log)

	// 记录成功的区块
	collector.RecordBlock(101, 10*time.Second, 100, nil)

	if collector.GetBlockCount() != 1 {
		t.Errorf("block count = %d, want 1", collector.GetBlockCount())
	}

	latest := collector.GetLatestMetric()
	if latest == nil {
		t.Fatal("latest metric is nil")
	}

	if latest.Height != 101 {
		t.Errorf("latest height = %d, want 101", latest.Height)
	}

	if latest.TxCount != 100 {
		t.Errorf("latest tx count = %d, want 100", latest.TxCount)
	}

	if !latest.Success {
		t.Error("latest block should be successful")
	}

	// 记录失败的区块
	collector.RecordBlock(102, 15*time.Second, 50, fmt.Errorf("test error"))

	if collector.GetBlockCount() != 2 {
		t.Errorf("block count = %d, want 2", collector.GetBlockCount())
	}

	if collector.GetFailedBlockCount() != 1 {
		t.Errorf("failed block count = %d, want 1", collector.GetFailedBlockCount())
	}
}

func TestMetricsCollector_ComputeAggregateMetrics(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	collector := NewMetricsCollector(types.TxHash{}, 100, log)

	// 记录多个区块
	for i := 0; i < 10; i++ {
		var err error
		if i == 5 {
			err = fmt.Errorf("test error")
		}
		collector.RecordBlock(
			uint64(101+i),
			10*time.Second,
			100,
			err,
		)
	}

	metrics := collector.ComputeAggregateMetrics()

	if metrics.TotalBlocks != 10 {
		t.Errorf("total blocks = %d, want 10", metrics.TotalBlocks)
	}

	if metrics.FailedBlocks != 1 {
		t.Errorf("failed blocks = %d, want 1", metrics.FailedBlocks)
	}

	if metrics.ErrorRate != 0.1 {
		t.Errorf("error rate = %f, want 0.1", metrics.ErrorRate)
	}

	if metrics.AvgBlockTime != 10.0 {
		t.Errorf("avg block time = %f, want 10.0", metrics.AvgBlockTime)
	}

	expectedThroughput := 1000.0 / 100.0 // 总交易数 / 总时间
	if metrics.AvgThroughput != expectedThroughput {
		t.Errorf("avg throughput = %f, want %f", metrics.AvgThroughput, expectedThroughput)
	}

	if metrics.StartHeight != 100 {
		t.Errorf("start height = %d, want 100", metrics.StartHeight)
	}

	if metrics.EndHeight != 110 {
		t.Errorf("end height = %d, want 110", metrics.EndHeight)
	}
}

func TestMetricsCollector_EvaluateCondition(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	collector := NewMetricsCollector(types.TxHash{}, 100, log)

	// 记录一些成功的区块
	for i := 0; i < 10; i++ {
		collector.RecordBlock(
			uint64(101+i),
			10*time.Second,
			100,
			nil,
		)
	}

	condition := &pb.RollbackCondition{
		MaxErrorRate:       0.05,
		MaxLatencyIncrease: 1.5,
		MinThroughputRatio: 0.8,
		TimeoutBlocks:      100,
	}

	// 应该通过评估（错误率为0）
	ok, reason := collector.EvaluateCondition(condition)
	if !ok {
		t.Errorf("evaluation should pass, got reason: %s", reason)
	}

	// 添加一些失败的区块
	for i := 0; i < 10; i++ {
		collector.RecordBlock(
			uint64(111+i),
			10*time.Second,
			100,
			fmt.Errorf("test error"),
		)
	}

	// 现在错误率是50%，应该失败
	ok, reason = collector.EvaluateCondition(condition)
	if ok {
		t.Error("evaluation should fail with high error rate")
	}

	if reason == "" {
		t.Error("reason should not be empty")
	}
}

func TestMetricsCollector_Reset(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	collector := NewMetricsCollector(types.TxHash{}, 100, log)

	// 记录一些区块
	for i := 0; i < 5; i++ {
		collector.RecordBlock(
			uint64(101+i),
			10*time.Second,
			100,
			nil,
		)
	}

	if collector.GetBlockCount() != 5 {
		t.Errorf("block count = %d, want 5", collector.GetBlockCount())
	}

	// 重置
	collector.Reset()

	if collector.GetBlockCount() != 0 {
		t.Errorf("block count after reset = %d, want 0", collector.GetBlockCount())
	}

	metrics := collector.GetMetrics()
	if metrics.TotalBlocks != 0 {
		t.Errorf("total blocks after reset = %d, want 0", metrics.TotalBlocks)
	}
}

func TestMetricsCollector_GetAverageBlockTime(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	collector := NewMetricsCollector(types.TxHash{}, 100, log)

	// 空的collector
	avgTime := collector.GetAverageBlockTime()
	if avgTime != 0 {
		t.Errorf("average block time for empty collector = %v, want 0", avgTime)
	}

	// 添加不同的区块时间
	collector.RecordBlock(101, 5*time.Second, 100, nil)
	collector.RecordBlock(102, 10*time.Second, 100, nil)
	collector.RecordBlock(103, 15*time.Second, 100, nil)

	avgTime = collector.GetAverageBlockTime()
	expected := 10 * time.Second
	if avgTime != expected {
		t.Errorf("average block time = %v, want %v", avgTime, expected)
	}
}

func TestMetricsCollector_GetTotalTransactions(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	collector := NewMetricsCollector(types.TxHash{}, 100, log)

	// 添加区块
	collector.RecordBlock(101, 10*time.Second, 50, nil)
	collector.RecordBlock(102, 10*time.Second, 75, nil)
	collector.RecordBlock(103, 10*time.Second, 100, nil)

	total := collector.GetTotalTransactions()
	if total != 225 {
		t.Errorf("total transactions = %d, want 225", total)
	}
}

func TestMetricsCollector_EmptyMetrics(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	collector := NewMetricsCollector(types.TxHash{}, 100, log)

	// 获取空collector的指标
	metrics := collector.GetMetrics()

	if metrics.TotalBlocks != 0 {
		t.Errorf("total blocks = %d, want 0", metrics.TotalBlocks)
	}

	if metrics.StartHeight != 100 {
		t.Errorf("start height = %d, want 100", metrics.StartHeight)
	}

	if metrics.EndHeight != 100 {
		t.Errorf("end height = %d, want 100", metrics.EndHeight)
	}
}

func TestMetricsCollector_Concurrency(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	collector := NewMetricsCollector(types.TxHash{}, 100, log)

	// 并发记录区块
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(height int) {
			collector.RecordBlock(
				uint64(101+height),
				10*time.Second,
				100,
				nil,
			)
			done <- true
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证结果
	if collector.GetBlockCount() != 10 {
		t.Errorf("block count = %d, want 10", collector.GetBlockCount())
	}
}
