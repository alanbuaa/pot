package upgrade

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

func setupTestPersistence(t *testing.T) (*BoltDBPersistence, func()) {
	tmpDir := t.TempDir()
	log := logrus.NewEntry(logrus.New())

	persistence, err := NewBoltDBPersistence(tmpDir, log)
	if err != nil {
		t.Fatalf("Failed to create persistence: %v", err)
	}

	cleanup := func() {
		persistence.Close()
	}

	return persistence, cleanup
}

func TestBoltDBPersistence_SaveAndLoadState(t *testing.T) {
	persistence, cleanup := setupTestPersistence(t)
	defer cleanup()

	// 创建测试状态
	state := &UpgradeState{
		Phase:     PhasePreexecuting,
		Started:   true,
		Completed: false,
		Failed:    false,
		StartTime: time.Now(),
	}

	// 保存状态
	err := persistence.SaveState(state)
	assert.NoError(t, err)

	// 加载状态
	loadedState, err := persistence.LoadState()
	assert.NoError(t, err)
	assert.NotNil(t, loadedState)

	assert.Equal(t, state.Phase, loadedState.Phase)
	assert.Equal(t, state.Started, loadedState.Started)
	assert.Equal(t, state.Completed, loadedState.Completed)
	assert.Equal(t, state.Failed, loadedState.Failed)
}

func TestBoltDBPersistence_SaveAndLoadProposal(t *testing.T) {
	persistence, cleanup := setupTestPersistence(t)
	defer cleanup()

	// 创建测试提案
	proposalID := types.TxHash{1, 2, 3, 4, 5}
	proposal := &UpgradeProposal{
		ProposalID:         proposalID,
		TargetConsensus:    "HotStuff",
		PreexecStartHeight: 100,
		SwitchHeight:       200,
		Timestamp:          time.Now(),
	}

	// 保存提案
	err := persistence.SaveProposal(proposal)
	assert.NoError(t, err)

	// 加载提案
	loadedProposal, err := persistence.LoadProposal(proposalID)
	assert.NoError(t, err)
	assert.NotNil(t, loadedProposal)

	assert.Equal(t, proposal.ProposalID, loadedProposal.ProposalID)
	assert.Equal(t, proposal.TargetConsensus, loadedProposal.TargetConsensus)
	assert.Equal(t, proposal.PreexecStartHeight, loadedProposal.PreexecStartHeight)
	assert.Equal(t, proposal.SwitchHeight, loadedProposal.SwitchHeight)
}

func TestBoltDBPersistence_ListProposals(t *testing.T) {
	persistence, cleanup := setupTestPersistence(t)
	defer cleanup()

	// 创建多个提案
	proposals := []*UpgradeProposal{
		{
			ProposalID:      types.TxHash{1, 0, 0, 0, 0},
			TargetConsensus: "HotStuff",
			SwitchHeight:    100,
		},
		{
			ProposalID:      types.TxHash{2, 0, 0, 0, 0},
			TargetConsensus: "PBFT",
			SwitchHeight:    200,
		},
		{
			ProposalID:      types.TxHash{3, 0, 0, 0, 0},
			TargetConsensus: "PoW",
			SwitchHeight:    300,
		},
	}

	// 保存所有提案
	for _, proposal := range proposals {
		err := persistence.SaveProposal(proposal)
		assert.NoError(t, err)
	}

	// 列出所有提案
	loadedProposals, err := persistence.ListProposals()
	assert.NoError(t, err)
	assert.Equal(t, len(proposals), len(loadedProposals))
}

func TestBoltDBPersistence_DeleteProposal(t *testing.T) {
	persistence, cleanup := setupTestPersistence(t)
	defer cleanup()

	proposalID := types.TxHash{1, 2, 3}
	proposal := &UpgradeProposal{
		ProposalID:      proposalID,
		TargetConsensus: "HotStuff",
	}

	// 保存提案
	err := persistence.SaveProposal(proposal)
	assert.NoError(t, err)

	// 验证存在
	_, err = persistence.LoadProposal(proposalID)
	assert.NoError(t, err)

	// 删除提案
	err = persistence.DeleteProposal(proposalID)
	assert.NoError(t, err)

	// 验证已删除
	_, err = persistence.LoadProposal(proposalID)
	assert.Error(t, err)
}

func TestBoltDBPersistence_SaveAndQueryEvents(t *testing.T) {
	persistence, cleanup := setupTestPersistence(t)
	defer cleanup()

	proposalID := types.TxHash{1, 2, 3}

	// 创建多个事件
	events := []*UpgradeEvent{
		{
			Type:        EventProposalCreated,
			ProposalID:  proposalID,
			Phase:       PhasePreparing,
			Description: "Proposal created",
			Timestamp:   time.Now(),
		},
		{
			Type:        EventUpgradeStarted,
			ProposalID:  proposalID,
			Phase:       PhasePreexecuting,
			Description: "Upgrade started",
			Timestamp:   time.Now().Add(1 * time.Second),
		},
		{
			Type:        EventSwitchExecuted,
			ProposalID:  proposalID,
			Phase:       PhaseCompleted,
			Description: "Switch executed",
			Timestamp:   time.Now().Add(2 * time.Second),
		},
	}

	// 保存所有事件
	for _, event := range events {
		err := persistence.SaveEvent(event)
		assert.NoError(t, err)
	}

	// 查询所有事件
	loadedEvents, err := persistence.QueryEvents(EventFilter{Limit: 10})
	assert.NoError(t, err)
	assert.Equal(t, len(events), len(loadedEvents))

	// 按提案 ID 过滤
	filteredEvents, err := persistence.QueryEvents(EventFilter{
		ProposalID: &proposalID,
		Limit:      10,
	})
	assert.NoError(t, err)
	assert.Equal(t, len(events), len(filteredEvents))

	// 按事件类型过滤
	eventType := EventUpgradeStarted
	typeFilteredEvents, err := persistence.QueryEvents(EventFilter{
		EventType: &eventType,
		Limit:     10,
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(typeFilteredEvents))
	assert.Equal(t, EventUpgradeStarted, typeFilteredEvents[0].Type)
}

func TestBoltDBPersistence_SaveAndLoadMetrics(t *testing.T) {
	persistence, cleanup := setupTestPersistence(t)
	defer cleanup()

	// 创建测试指标
	metrics := &AggregateMetrics{
		Height:        100,
		AvgBlockTime:  1.5,
		AvgTxPerBlock: 100.0,
		TotalTx:       10000,
		BlockCount:    100,
		MaxBlockTime:  2.0,
		MinBlockTime:  1.0,
	}

	// 保存指标
	err := persistence.SaveMetrics(100, metrics)
	assert.NoError(t, err)

	// 加载指标
	loadedMetrics, err := persistence.LoadMetrics(100)
	assert.NoError(t, err)
	assert.NotNil(t, loadedMetrics)

	assert.Equal(t, metrics.Height, loadedMetrics.Height)
	assert.Equal(t, metrics.AvgBlockTime, loadedMetrics.AvgBlockTime)
	assert.Equal(t, metrics.TotalTx, loadedMetrics.TotalTx)
}

func TestBoltDBPersistence_QueryMetrics(t *testing.T) {
	persistence, cleanup := setupTestPersistence(t)
	defer cleanup()

	// 保存多个高度的指标
	for i := uint64(100); i <= 110; i++ {
		metrics := &AggregateMetrics{
			Height:        i,
			AvgBlockTime:  float64(i) / 100.0,
			AvgTxPerBlock: float64(i),
			TotalTx:       i * 100,
		}
		err := persistence.SaveMetrics(i, metrics)
		assert.NoError(t, err)
	}

	// 查询范围
	result, err := persistence.QueryMetrics(100, 105)
	assert.NoError(t, err)
	assert.Equal(t, 6, len(result)) // 100, 101, 102, 103, 104, 105

	// 验证数据
	assert.NotNil(t, result[100])
	assert.Equal(t, uint64(100), result[100].Height)
	assert.NotNil(t, result[105])
	assert.Equal(t, uint64(105), result[105].Height)
}

func TestBoltDBPersistence_Clear(t *testing.T) {
	persistence, cleanup := setupTestPersistence(t)
	defer cleanup()

	// 保存一些数据
	state := &UpgradeState{Phase: PhasePreexecuting, Started: true}
	err := persistence.SaveState(state)
	assert.NoError(t, err)

	proposal := &UpgradeProposal{
		ProposalID:      types.TxHash{1, 2, 3},
		TargetConsensus: "HotStuff",
	}
	err = persistence.SaveProposal(proposal)
	assert.NoError(t, err)

	// 清空所有数据
	err = persistence.Clear()
	assert.NoError(t, err)

	// 验证数据已清空
	_, err = persistence.LoadState()
	assert.Error(t, err)

	proposals, err := persistence.ListProposals()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(proposals))
}

func TestBoltDBPersistence_Concurrency(t *testing.T) {
	persistence, cleanup := setupTestPersistence(t)
	defer cleanup()

	// 并发保存事件
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(index int) {
			event := &UpgradeEvent{
				Type:        EventPhaseChanged,
				ProposalID:  types.TxHash{byte(index)},
				Description: "Concurrent event",
				Timestamp:   time.Now(),
			}
			err := persistence.SaveEvent(event)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证所有事件都已保存
	events, err := persistence.QueryEvents(EventFilter{Limit: 20})
	assert.NoError(t, err)
	assert.Equal(t, 10, len(events))
}

func TestBoltDBPersistence_EventTypeString(t *testing.T) {
	tests := []struct {
		eventType EventType
		expected  string
	}{
		{EventProposalCreated, "ProposalCreated"},
		{EventUpgradeStarted, "UpgradeStarted"},
		{EventPhaseChanged, "PhaseChanged"},
		{EventSwitchExecuted, "SwitchExecuted"},
		{EventRollbackExecuted, "RollbackExecuted"},
		{EventUpgradeCompleted, "UpgradeCompleted"},
		{EventUpgradeFailed, "UpgradeFailed"},
		{EventType(999), "Unknown"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.eventType.String())
	}
}
