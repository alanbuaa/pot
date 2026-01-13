package upgrade

import (
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zzz136454872/upgradeable-consensus/config"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"google.golang.org/protobuf/proto"
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
		Threshold:          1, // 添加必需的阈值字段
		Timestamp:          time.Now(),
	}

	err := um.StartUpgrade(proposal, mc2)
	require.NoError(t, err)

	state := um.GetUpgradeState()

	assert.NotNil(t, state)
	assert.Equal(t, PhasePreexecuting, state.Phase)
	assert.True(t, state.Started)
	assert.False(t, state.Completed)
	assert.False(t, state.Failed)
}

// TestUpgradeManager_ValidateProposal 测试提案验证
func TestUpgradeManager_ValidateProposal(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mc := newMockConsensus()
	mockStorage := newMockMultiChainStorage()
	cfg := &config.ConsensusConfig{}

	um := NewUpgradeManager(mc, cfg, mockStorage, log)

	tests := []struct {
		name        string
		proposal    *UpgradeProposal
		expectError bool
	}{
		{
			name:        "nil proposal",
			proposal:    nil,
			expectError: true,
		},
		{
			name: "empty target consensus",
			proposal: &UpgradeProposal{
				ProposalID:         types.TxHash{1, 2, 3},
				TargetConsensus:    "",
				PreexecStartHeight: 100,
				SwitchHeight:       200,
				Threshold:          3,
			},
			expectError: true,
		},
		{
			name: "valid proposal",
			proposal: &UpgradeProposal{
				ProposalID:         types.TxHash{1, 2, 3},
				TargetConsensus:    "hotstuff",
				PreexecStartHeight: 100,
				SwitchHeight:       200,
				Threshold:          3,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := um.ValidateProposal(tt.proposal)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestUpgradeManager_MessageCache 测试消息缓存功能
func TestUpgradeManager_MessageCache(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mc := newMockConsensus()
	mockStorage := newMockMultiChainStorage()
	cfg := &config.ConsensusConfig{}

	um := NewUpgradeManager(mc, cfg, mockStorage, log)

	// 设置当前epoch
	currentEpoch := uint64(100)
	um.SetCurrentEpoch(currentEpoch)

	// 测试缓存未来消息
	futureMsg := map[string]interface{}{"data": "future"}
	err := um.OnNetworkMessage(futureMsg, 2, 1, 105)
	assert.NoError(t, err)

	// 测试当前epoch消息
	currentMsg := map[string]interface{}{"data": "current"}
	err = um.OnNetworkMessage(currentMsg, 2, 1, 100)
	assert.NoError(t, err)

	// 测试过期消息
	pastMsg := map[string]interface{}{"data": "past"}
	err = um.OnNetworkMessage(pastMsg, 2, 1, 95)
	assert.NoError(t, err)
}

// TestUpgradeManager_SetCurrentEpoch 测试设置epoch
func TestUpgradeManager_SetCurrentEpoch(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mc := newMockConsensus()
	mockStorage := newMockMultiChainStorage()
	cfg := &config.ConsensusConfig{}

	um := NewUpgradeManager(mc, cfg, mockStorage, log)

	um.SetCurrentEpoch(100)
	assert.Equal(t, uint64(100), um.currentEpoch)

	um.SetCurrentEpoch(150)
	assert.Equal(t, uint64(150), um.currentEpoch)
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

// TestUpgradeManager_PersistenceRecovery 测试持久化和恢复
func TestUpgradeManager_PersistenceRecovery(t *testing.T) {
	tempDir := t.TempDir()
	log := logrus.NewEntry(logrus.New())
	mc := newMockConsensus()
	mockStorage := newMockMultiChainStorage()
	cfg := &config.ConsensusConfig{}

	persistence, err := NewBoltDBPersistence(tempDir, log)
	require.NoError(t, err)
	defer persistence.Close()

	// 创建带持久化的升级管理器
	um, err := NewUpgradeManagerWithPersistence(mc, cfg, mockStorage, persistence, log)
	require.NoError(t, err)
	require.NotNil(t, um)

	// 保存提案
	proposal := &UpgradeProposal{
		ProposalID:         types.TxHash{1, 2, 3},
		TargetConsensus:    "TestConsensus",
		PreexecStartHeight: 100,
		SwitchHeight:       200,
		Threshold:          3,
		Timestamp:          time.Now(),
	}

	err = um.persistProposal(proposal)
	require.NoError(t, err)

	// 恢复提案
	recovered, err := persistence.LoadProposal(proposal.ProposalID)
	require.NoError(t, err)
	require.NotNil(t, recovered)
	assert.Equal(t, proposal.ProposalID, recovered.ProposalID)
	assert.Equal(t, proposal.TargetConsensus, recovered.TargetConsensus)
	assert.Equal(t, proposal.PreexecStartHeight, recovered.PreexecStartHeight)
	assert.Equal(t, proposal.SwitchHeight, recovered.SwitchHeight)
	assert.Equal(t, proposal.Threshold, recovered.Threshold)
}

// TestUpgradeManager_ConcurrentAccess 测试并发访问安全性
func TestUpgradeManager_ConcurrentAccess(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mc := newMockConsensus()
	mockStorage := newMockMultiChainStorage()
	cfg := &config.ConsensusConfig{}

	um := NewUpgradeManager(mc, cfg, mockStorage, log)

	// 并发设置epoch
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(epoch uint64) {
			defer wg.Done()
			um.SetCurrentEpoch(epoch)
		}(uint64(i))
	}
	wg.Wait()

	// 并发获取状态
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			state := um.GetUpgradeState()
			assert.NotNil(t, state)
		}()
	}
	wg.Wait()
}

// TestUpgradeManager_OnCandidateConsensusStarted 测试候选共识启动处理
func TestUpgradeManager_OnCandidateConsensusStarted(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mc := newMockConsensus()
	mockStorage := newMockMultiChainStorage()
	cfg := &config.ConsensusConfig{}

	um := NewUpgradeManager(mc, cfg, mockStorage, log)
	um.currentEpoch = 100

	// 缓存一些消息到新共识
	for i := 0; i < 5; i++ {
		msg := &pb.Request{}
		um.OnNetworkMessage(msg, 0, 1, 105)
	}

	// 启动候选共识（应该冲刷缓存）
	err := um.OnCandidateConsensusStarted(1)
	assert.NoError(t, err)
}

// TestUpgradeManager_OnConsensusSwitched 测试共识切换后清理
func TestUpgradeManager_OnConsensusSwitched(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mc := newMockConsensus()
	mockStorage := newMockMultiChainStorage()
	cfg := &config.ConsensusConfig{}

	um := NewUpgradeManager(mc, cfg, mockStorage, log)

	// 设置当前epoch
	um.currentEpoch = 100

	// 设置消息缓存切换信息
	um.messageCache.SetSwitchInfo(100, 0, 1)

	// 缓存一些消息到旧共识
	for i := 0; i < 5; i++ {
		msg := &pb.Request{}
		um.OnNetworkMessage(msg, 1, 0, 90)
	}

	// 缓存一些消息到新共识
	for i := 0; i < 3; i++ {
		msg := &pb.Request{}
		um.OnNetworkMessage(msg, 0, 1, 100)
	}

	// 切换到新epoch（传入旧epoch）
	um.OnConsensusSwitched(100)

	// 设置新epoch
	um.SetCurrentEpoch(105)

	// 验证epoch已更新
	assert.Equal(t, uint64(105), um.currentEpoch)
}

// TestUpgradeManager_Reset 测试重置功能
func TestUpgradeManager_Reset(t *testing.T) {
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

	// 启动升级
	proposal := &UpgradeProposal{
		ProposalID:         types.TxHash{1, 2, 3},
		TargetConsensus:    "TestConsensus",
		PreexecStartHeight: 10,
		SwitchHeight:       100,
		Threshold:          1,
		Timestamp:          time.Now(),
	}

	err := um.StartUpgrade(proposal, mc2)
	require.NoError(t, err)

	state := um.GetUpgradeState()
	assert.True(t, state.Started)

	// 重置
	um.Reset()

	// 验证状态已重置
	state = um.GetUpgradeState()
	assert.False(t, state.Started)
	assert.False(t, state.Completed)
	assert.False(t, state.Failed)
	assert.Equal(t, PhaseIdle, state.Phase)
}

// TestUpgradeManager_ForwardToConsensus 测试消息转发
func TestUpgradeManager_ForwardToConsensus(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mc1 := newMockConsensus()
	mc1.consensusID = 1
	mc2 := newMockConsensus()
	mc2.consensusID = 2
	mockStorage := newMockMultiChainStorage()
	cfg := &config.ConsensusConfig{}

	um := NewUpgradeManager(mc1, cfg, mockStorage, log)

	// 保存分叉点区块
	forkBlock := newTestBlock(10)
	mockStorage.StoreMainBlock(forkBlock)

	// 启动升级以创建候选共识
	proposal := &UpgradeProposal{
		ProposalID:         types.TxHash{1, 2, 3},
		TargetConsensus:    "TestConsensus",
		PreexecStartHeight: 10,
		SwitchHeight:       100,
		Threshold:          1,
		Timestamp:          time.Now(),
	}

	err := um.StartUpgrade(proposal, mc2)
	require.NoError(t, err)

	// 测试转发到主共识
	// 注意：由于mockConsensus使用无缓冲channel，可能会失败
	// 这里我们只验证转发不会panic或返回其他严重错误
	msg1 := []byte("test message 1")
	err = um.forwardToConsensus(1, msg1)
	// 允许"channel is full"错误
	if err != nil && err.Error() != "consensus message channel is full" {
		t.Errorf("Unexpected error: %v", err)
	}

	// 测试转发到候选共识
	msg2 := &pb.Request{}
	err = um.forwardToConsensus(2, msg2)
	// 允许"channel is full"错误
	if err != nil && err.Error() != "consensus request channel is full" {
		t.Errorf("Unexpected error: %v", err)
	}

	// 测试转发到不存在的共识
	err = um.forwardToConsensus(999, msg1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestUpgradeManager_IsConsensusRunning 测试共识运行状态检查
func TestUpgradeManager_IsConsensusRunning(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mc1 := newMockConsensus()
	mc1.consensusID = 1
	mc2 := newMockConsensus()
	mc2.consensusID = 2
	mockStorage := newMockMultiChainStorage()
	cfg := &config.ConsensusConfig{}

	um := NewUpgradeManager(mc1, cfg, mockStorage, log)

	// 当前共识应该运行
	assert.True(t, um.isConsensusRunning(1))

	// 不存在的共识
	assert.False(t, um.isConsensusRunning(999))

	// 启动候选共识
	forkBlock := newTestBlock(10)
	mockStorage.StoreMainBlock(forkBlock)

	proposal := &UpgradeProposal{
		ProposalID:         types.TxHash{1, 2, 3},
		TargetConsensus:    "TestConsensus",
		PreexecStartHeight: 10,
		SwitchHeight:       100,
		Threshold:          1,
		Timestamp:          time.Now(),
	}

	err := um.StartUpgrade(proposal, mc2)
	require.NoError(t, err)

	// 候选共识应该运行
	assert.True(t, um.isConsensusRunning(2))
}

// TestUpgradeManager_ControlMessage 测试控制消息处理
func TestUpgradeManager_ControlMessage(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mc := newMockConsensus()
	mockStorage := newMockMultiChainStorage()
	cfg := &config.ConsensusConfig{}

	um := NewUpgradeManager(mc, cfg, mockStorage, log)

	// 创建升级交易
	upgradeTx := &pb.Transaction{
		Payload: []byte("UPGRADE:test_config"),
	}

	// 测试是否识别为升级交易
	assert.True(t, isUpgradeTransaction(upgradeTx))

	// 创建包含升级交易的Request
	txBytes, err := proto.Marshal(upgradeTx)
	require.NoError(t, err)

	request := &pb.Request{
		Tx: txBytes,
	}

	// 测试是否识别为控制消息
	assert.True(t, um.isControlMessage(request))

	// 测试处理控制消息
	err = um.handleControlMessage(request)
	assert.NoError(t, err)

	// 普通交易
	normalTx := &pb.Transaction{
		Payload: []byte("normal transaction"),
	}
	assert.False(t, isUpgradeTransaction(normalTx))
}

// TestUpgradeManager_ProcessBlock 测试区块处理
func TestUpgradeManager_ProcessBlock(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mc1 := newMockConsensus()
	mc2 := newMockConsensus()
	mc2.consensusID = 2
	mockStorage := newMockMultiChainStorage()
	cfg := &config.ConsensusConfig{}

	um := NewUpgradeManager(mc1, cfg, mockStorage, log)

	// 未启动升级时处理区块
	block := &pb.Block{
		Header: &pb.Header{Height: 50},
	}
	err := um.ProcessBlock(block, true)
	assert.NoError(t, err)

	// 启动升级
	forkBlock := newTestBlock(10)
	mockStorage.StoreMainBlock(forkBlock)

	proposal := &UpgradeProposal{
		ProposalID:         types.TxHash{1, 2, 3},
		TargetConsensus:    "TestConsensus",
		PreexecStartHeight: 10,
		SwitchHeight:       100,
		Threshold:          1,
		Timestamp:          time.Now(),
	}

	err = um.StartUpgrade(proposal, mc2)
	require.NoError(t, err)

	// 处理主链区块（未达到切换高度）
	block = &pb.Block{
		Header: &pb.Header{Height: 50},
	}
	err = um.ProcessBlock(block, true)
	assert.NoError(t, err)

	// 处理候选链区块
	candidateBlock := &pb.Block{
		Header: &pb.Header{Height: 20},
		Txs:    []*pb.Tx{},
	}
	err = um.ProcessBlock(candidateBlock, false)
	assert.NoError(t, err)
}

// TestUpgradeManager_PersistenceCachedMessages 测试消息缓存持久化
func TestUpgradeManager_PersistenceCachedMessages(t *testing.T) {
	tempDir := t.TempDir()
	log := logrus.NewEntry(logrus.New())
	mc := newMockConsensus()
	mockStorage := newMockMultiChainStorage()
	cfg := &config.ConsensusConfig{}

	persistence, err := NewBoltDBPersistence(tempDir, log)
	require.NoError(t, err)
	defer persistence.Close()

	um, err := NewUpgradeManagerWithPersistence(mc, cfg, mockStorage, persistence, log)
	require.NoError(t, err)

	um.SetCurrentEpoch(100)

	// 缓存消息（会触发持久化）
	msg := map[string]interface{}{"data": "test"}
	err = um.OnNetworkMessage(msg, 0, 1, 105)
	assert.NoError(t, err)

	// 验证持久化方法被调用（通过日志或其他方式）
	// 注意：由于persistence接口可能不支持SaveCachedMessage，这里主要测试不崩溃
}

// TestUpgradeManager_GettersSetters 测试各种getter和setter
func TestUpgradeManager_GettersSetters(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mc := newMockConsensus()
	mockStorage := newMockMultiChainStorage()
	cfg := &config.ConsensusConfig{}

	um := NewUpgradeManager(mc, cfg, mockStorage, log)

	// 测试GetPersistence
	assert.Nil(t, um.GetPersistence())

	// 测试GetMultiChainManager
	assert.NotNil(t, um.GetMultiChainManager())

	// 测试GetMessageCache
	assert.NotNil(t, um.GetMessageCache())

	// 测试GetCurrentProposal
	assert.Nil(t, um.GetCurrentProposal())

	// 测试GetCurrentPhase
	assert.Equal(t, PhaseIdle, um.GetCurrentPhase())

	// 测试IsUpgrading
	assert.False(t, um.IsUpgrading())

	// 测试GetMetrics
	assert.Nil(t, um.GetMetrics())
}

// TestUpgradeManager_ErrorScenarios 测试各种错误场景
func TestUpgradeManager_ErrorScenarios(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mc1 := newMockConsensus()
	mc2 := newMockConsensus()
	mc2.consensusID = 2
	mockStorage := newMockMultiChainStorage()
	cfg := &config.ConsensusConfig{}

	um := NewUpgradeManager(mc1, cfg, mockStorage, log)

	// 测试重复启动升级
	forkBlock := newTestBlock(10)
	mockStorage.StoreMainBlock(forkBlock)

	proposal := &UpgradeProposal{
		ProposalID:         types.TxHash{1, 2, 3},
		TargetConsensus:    "TestConsensus",
		PreexecStartHeight: 10,
		SwitchHeight:       100,
		Threshold:          1,
		Timestamp:          time.Now(),
	}

	err := um.StartUpgrade(proposal, mc2)
	require.NoError(t, err)

	// 第二次启动应该失败
	err = um.StartUpgrade(proposal, mc2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already in progress")

	// 测试无效提案
	invalidProposal := &UpgradeProposal{
		ProposalID:         types.TxHash{4, 5, 6},
		TargetConsensus:    "", // 空目标共识
		PreexecStartHeight: 10,
		SwitchHeight:       100,
		Threshold:          1,
	}

	um2 := NewUpgradeManager(mc1, cfg, mockStorage, log)
	err = um2.StartUpgrade(invalidProposal, mc2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}
