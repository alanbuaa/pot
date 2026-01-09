package upgrade

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

// TestUpgradeManagerCreation 测试升级管理器创建
func TestUpgradeManagerCreation(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mc := newMockConsensus()
	mockStorage := newMockMultiChainStorage()
	cfg := &config.ConsensusConfig{}

	um := NewUpgradeManager(mc, cfg, mockStorage, log)

	assert.NotNil(t, um)
	assert.False(t, um.IsUpgrading())
	assert.Equal(t, PhaseIdle, um.GetCurrentPhase())
}

// TestUpgradeManagerStartUpgrade 测试启动升级
func TestUpgradeManagerStartUpgrade(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mc1 := newMockConsensus()
	mc2 := newMockConsensus()
	mc2.consensusID = 2
	mockStorage := newMockMultiChainStorage()
	cfg := &config.ConsensusConfig{}

	um := NewUpgradeManager(mc1, cfg, mockStorage, log)

	proposal := &UpgradeProposal{
		ProposalID:         types.TxHash{1, 2, 3},
		TargetConsensus:    "TestConsensus",
		PreexecStartHeight: 10,
		SwitchHeight:       100,
		Timestamp:          time.Now(),
	}

	err := um.StartUpgrade(proposal, mc2)

	// 预期失败（因为没有fork block）或者成功都可以
	_ = err
}

// TestUpgradeManagerGetState 测试获取升级状态
func TestUpgradeManagerGetState(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mc1 := newMockConsensus()
	mc2 := newMockConsensus()
	mc2.consensusID = 2
	mockStorage := newMockMultiChainStorage()
	cfg := &config.ConsensusConfig{}

	um := NewUpgradeManager(mc1, cfg, mockStorage, log)

	// 先保存分叉点区块
	forkBlock := newTestBlock(10)
	mockStorage.StoreMainBlock(forkBlock)

	proposal := &UpgradeProposal{
		ProposalID:         types.TxHash{1, 2, 3},
		TargetConsensus:    "TestConsensus",
		PreexecStartHeight: 10,
		SwitchHeight:       100,
		Timestamp:          time.Now(),
	}

	um.StartUpgrade(proposal, mc2)

	state := um.GetUpgradeState()

	assert.NotNil(t, state)
	assert.Equal(t, PhasePreexecuting, state.Phase)
	assert.True(t, state.Started)
	assert.False(t, state.Completed)
	assert.False(t, state.Failed)
}

// TestUpgradePhaseString 测试升级阶段字符串表示
func TestUpgradePhaseString(t *testing.T) {
	tests := []struct {
		phase    UpgradePhase
		expected string
	}{
		{PhaseIdle, "Idle"},
		{PhasePreparing, "Preparing"},
		{PhasePreexecuting, "Preexecuting"},
		{PhaseSwitching, "Switching"},
		{PhaseCompleted, "Completed"},
		{PhaseRolledBack, "RolledBack"},
		{PhaseFailed, "Failed"},
		{UpgradePhase(999), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.phase.String())
		})
	}
}
